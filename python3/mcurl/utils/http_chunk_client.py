#!/usr/bin/env python
# coding=utf-8
import requests
import gevent.timeout
import logging
__author__ = 'chenfengyuan'

PROXIES = {
    'http': 'http://127.0.0.1:9999',
    'https': 'http://127.0.0.1:9999',
}
PROXIES = {}
logger = logging.getLogger(__name__)


class Chunk:
    def __init__(self, start, chunk_size):
        self.chunk_max_size = chunk_size
        self.chunk = []
        self.chunk_size = 0
        self.start = start

    def consume(self, data_iter):
        for data in data_iter:
            if self.chunk_size + len(data) > self.chunk_max_size:
                remain = data[self.chunk_max_size - self.chunk_size:]
                self.chunk.append(data[:self.chunk_max_size - self.chunk_size])
                rv = self.chunk
                self.chunk = [remain]
                self.chunk_size = len(remain)
                yield self.start, rv,
                self.start += self.chunk_max_size
            else:
                self.chunk_size += len(data)
                self.chunk.append(data)
        yield self.start, self.chunk


class HttpChunkClient:

    def __init__(self, url, headers, range_start, range_end, chunk_size, chunk_timeout, filesize):
        headers = headers + [['Range', 'bytes=%d-' % range_start]]
        headers = dict(headers)
        if 'baidu.com' in headers['Host']:
            headers['User-Agent'] = 'netdisk;5.2.7.2;PC;PC-Windows;6.2.9200;WindowsBaiduYunGuanJia'
        del headers['Host']
        self.range_end = range_end
        self.range_start = range_start
        self.headers = headers
        self.url = url
        self.r = None
        self.chunk_timeout = chunk_timeout
        self.chunk_size = chunk_size
        self.filesize = filesize

    def get_resp_with_redirect_headers(self):
        url = self.url
        for redirect_times in range(10):
            r = requests.get(url, headers=self.headers, stream=True, allow_redirects=False, proxies=PROXIES)
            if r.status_code == 302:
                url = r.headers['location']
                continue
            assert r.status_code // 100 == 2, 'get status code: %s' % r.status_code
            assert int(r.headers['content-length']) + self.range_start == self.filesize
            return r

    def iter_chunk(self):
        try:
            chunk = Chunk(self.range_start, self.chunk_size)
            self.r = self.get_resp_with_redirect_headers()
            t = gevent.timeout.Timeout.start_new(self.chunk_timeout)
            downloaded_size = 0
            for chunk_ in chunk.consume(self.r.iter_content(4096)):
                downloaded_size += self.chunk_size
                assert downloaded_size <= self.range_end - self.range_start
                t.cancel()
                yield chunk_
                if downloaded_size == self.range_start - self.range_end:
                    self.r = None
                    yield True,
                    return
                t = gevent.timeout.Timeout.start_new(self.chunk_timeout)
        except Exception:
            logger.error('failed to download %s', self.url, exc_info=True)
            self.r = None
            yield False,
            return
