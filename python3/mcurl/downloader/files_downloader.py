#!/usr/bin/env python
# coding=utf-8
from mcurl.utils import download_info_client
from mcurl.downloader.file_info import FileInfo
from gevent.queue import Queue
import gevent
import gevent.pool
import gevent.event
from mcurl.downloader.queue_data_types import Classification
import logging
from mcurl.downloader.file_downloader import FileDownloader
from mcurl.downloader import DBSession
__author__ = 'chenfengyuan'


logger = logging.getLogger(__name__)


class FilesDownloader:
    def __init__(self, tasks_str, host, port, max_concurrent):
        self.client = download_info_client.DownloadInfo(host, port)
        self.tasks_str = tasks_str
        self.tasks = []
        self.max_concurrent = max_concurrent
        self.exit_event = gevent.event.Event()

    def init_tasks(self):
        filenames = list()
        for task_str in self.tasks_str:
            if task_str.isnumeric():
                info = self.client.get_info(task_str.encode('utf-8'))
                obj = FileInfo.create_from_download_info(info)
                if obj.filename not in filenames:
                    filenames.append(obj.filename)
            else:
                if task_str not in filenames:
                    filenames.append(task_str)
        file_infos = list(map(FileInfo.get, filenames))
        DBSession().expunge_all()
        inq = Queue()
        filename_outq_map = {}
        """:type: Dict[str, Queue]"""
        filename_info_map = {}
        """:type: Dict[str, FileInfo"""
        for filename in filenames:
            filename_outq_map[filename] = Queue()
        for info in file_infos:
            filename_info_map[info.filename] = info

        undownload_filenames = list(filenames)
        for _ in range(min(self.max_concurrent, len(undownload_filenames))):
            inq.put((Classification.FILE_FINISHED, ))
        downloading_files = 0
        while undownload_filenames:
            data = inq.get()
            assert data[0] == Classification.FILE_FINISHED
            filename = undownload_filenames.pop(0)
            g = gevent.pool.Group()
            obj = FileDownloader(filename_info_map[filename], filename_outq_map[filename], inq, g)
            g.spawn(obj.start)
            downloading_files += 1

        while downloading_files:
            data = inq.get()
            assert data[0] == Classification.FILE_FINISHED
            downloading_files -= 1
        self.exit_event.set()
