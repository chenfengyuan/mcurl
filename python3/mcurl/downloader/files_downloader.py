#!/usr/bin/env python
# coding=utf-8
from mcurl.utils import download_info_client
from mcurl.downloader import file_info
__author__ = 'chenfengyuan'


class FilesDownloader:
    def __init__(self, tasks_nums, host, port, max_concurrent):
        self.client = download_info_client.DownloadInfo(host, port)
        self.tasks_nums = tasks_nums
        self.tasks = []

    def start_download(self):
        filenames = set()
        for nums in self.tasks_nums:
            download_infos = list(map(self.client.get_info, map(lambda x: x.encode('utf-8'), nums)))
            for info in download_infos:
                obj = file_info.FileInfo.create_from_download_info(info)
                filenames.add(obj.filename)
        file_infos = list(map(file_info.FileInfo.get, filenames))
        print(file_infos[0].requests)
