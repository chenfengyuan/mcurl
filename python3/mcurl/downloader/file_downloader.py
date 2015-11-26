#!/usr/bin/env python
# coding=utf-8
from gevent.queue import Queue
import json
__author__ = 'chenfengyuan'



class FileDownloader:
    def __init__(self, infos, worker_queue):
        """
        :type infos: list[dict]
        :type worker_queue: Queue
        """
        self.file_info = FileInfo(infos[0]['filename'],
                                  infos[0]['content_length'])



