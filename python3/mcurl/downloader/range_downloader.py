#!/usr/bin/env python
# coding=utf-8
from gevent.queue import Queue
from mcurl.downloader.queue_data_types import Classification
from mcurl.downloader.file_info import Range, Request
from mcurl.utils.http_chunk_client import HttpChunkClient
import logging
import time
__author__ = 'chenfengyuan'

logger = logging.getLogger(__name__)


class RangeDownloader:

    NO_RANGE_RETRY_TIME = 3
    REQUEST_NEW_RANGE_WAIT_TIME = 1
    FAIL_WAIT_TIMEOUT = 62

    def __init__(self, filesize: int, req: Request, outq: Queue, chunk_size, filename: str, number: int):
        self.outq = outq
        self.req = req
        self.filesize = filesize
        self.chunk_size = chunk_size
        self.stop_exc = Exception()
        self.number = '%s[%d]' % (filename, number)

    def start(self):
        while True:
            try:
                self._start()
            except Exception as e:
                if e is self.stop_exc:
                    break
                else:
                    logger.error('unexpected exception', exc_info=True)

    def _start(self):
        q = Queue()
        self.outq.put((Classification.REQUEST_DOWNLOAD_REANGE, q))
        data = q.get()
        cls = data[0]
        """:type: Classification"""
        if cls == Classification.DOWNLOAD_RANGE:
            range_ = data[1]
            assert isinstance(range_, Range)
            logger.debug('%s:new range:%s', self.number,
                         range_)
            c = HttpChunkClient(self.req.url, self.req.headers, range_[0], range_[1], self.chunk_size, 60,
                                self.filesize)
            for data in c.iter_chunk():
                if data[0] is True:
                    logger.debug('%s: waiting to request new range %d', self.number,
                                 self.REQUEST_NEW_RANGE_WAIT_TIME)
                    time.sleep(self.REQUEST_NEW_RANGE_WAIT_TIME)
                    return
                elif data[0] is False:
                    logger.debug('%s: fail waiting %d', self.number,
                                 self.FAIL_WAIT_TIMEOUT)
                    time.sleep(self.FAIL_WAIT_TIMEOUT)
                    return
                else:
                    start = data[0]
                    chunk = data[1]
                    logger.debug('%s:chunk downloaded:%s', self.number, start)
                    self.outq.put((Classification.NEW_DOWNLOADED_CHUNK, start, chunk, q))
                    data = q.get()
                    if data == Classification.NEXT_CHUNK_WANTED:
                        logger.debug('%s:downloading next chunk', self.number)
                        continue
                    elif data == Classification.CURREN_OR_NEXT_CHUNK_IS_DOWNLOADED:
                        logger.debug('%s:range downloader exiting', self.number)
                        return
                    else:
                        logger.error("%s:unexpected queue data: %s", self.number, data)
                        raise self.stop_exc
        elif cls == Classification.NO_RANGE_FOR_NOW:
            logger.debug('%s:no range for now', self.number)
            time.sleep(self.NO_RANGE_RETRY_TIME)
            return
        else:
            logger.error('%s:unknow classification and payload: %s', self.number, data)

