#!/usr/bin/env python
# coding=utf-8
from gevent.queue import Queue
from gevent.pool import Group
from gevent import spawn_later
from mcurl.downloader.file_info import FileInfo, FileChunk
from mcurl.downloader.range_downloader import RangeDownloader
from mcurl.downloader.queue_data_types import Classification
import logging
__author__ = 'chenfengyuan'


logger = logging.getLogger(__name__)


class FileDownloader:
    def __init__(self, file_info, inq, outq, group):
        """
        :type file_info: FileInfo
        :type inq: Queue
        :type outq: Queue
        :type group: Group
        """
        self.file_info = file_info
        self.inq = inq
        self.outq = outq
        self.group = group

    def start(self):
        logger.debug('start downloading file %s , get %d req', self.file_info.filename, len(self.file_info.requests))
        for req in self.file_info.requests:
            range_downloader = RangeDownloader(self.file_info.filesize, req, self.inq, self.file_info.ChunkSize,
                                               self.file_info.filename)
            self.group.spawn(range_downloader.start)

        while True:
            data = self.inq.get()
            cls = data[0]
            assert isinstance(cls, Classification)
            if cls == Classification.REQUEST_DOWNLOAD_REANGE:
                range_ = self.file_info.get_range_and_mark_downloading_time()
                if range_:
                    data[1].put((Classification.DOWNLOAD_RANGE, range_))
                else:
                    data[1].put((Classification.NO_RANGE_FOR_NOW, range_))
            elif cls == Classification.NEW_DOWNLOADED_CHUNK:
                start = data[1]
                chunk_data = data[2]
                q = data[3]
                chunk_ = FileChunk(start, chunk_data)
                if not self.file_info.chunk_is_downloaded_before(chunk_):
                    self.file_info.write(chunk_)
                    q.put(Classification.NEXT_CHUNK_WANTED)
                else:
                    q.put(Classification.CURREN_OR_NEXT_CHUNK_IS_DOWNLOADED)
            if self.file_info.is_finished():
                self.outq.put((Classification.FILE_FINISHED, self.file_info.filename))
                spawn_later(0, lambda g: g.kill(), self.group)
                return
