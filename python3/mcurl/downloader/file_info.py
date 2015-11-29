#!/usr/bin/env python
# coding=utf-8

import json
import logging
import time
import urllib.parse
from collections import namedtuple
from io import FileIO
from itertools import chain

from sqlalchemy import (
    Column,
    String,
    Integer,
    DateTime,
    PickleType
)
from sqlalchemy.sql.functions import now

from mcurl.downloader import DBSession, engine, Base
from mcurl.utils.alogrithm import find

__author__ = 'chenfengyuan'

Request = namedtuple('Request', ('url', 'headers'))
Range = namedtuple('Range', ('start', 'end'))

logger = logging.getLogger(__name__)


class FileChunk:
    def __init__(self, start, data):
        """
        :type start: int
        :type data: list[bytes]
        """
        self.start = start
        self.data = data
        len_ = 0
        for x in self.data:
            len_ += len(x)
        self.size = len_


class FileInfo(Base):
    __tablename__ = 'file_info'

    id = Column(Integer, primary_key=True, nullable=False)
    filename = Column(String, nullable=False, unique=True, index=True)
    filesize = Column(Integer, nullable=False, default=0)
    requests = Column(PickleType, nullable=False)
    """:type: List[Request]"""
    chunks = Column(PickleType, nullable=False)
    """:type: List[Bool]"""
    created_at = Column(DateTime(timezone=True), default=now())
    updated_at = Column(DateTime(timezone=True), default=now(), onupdate=now())

    ChunkSize = 1024 * 1024
    MinBlockLength = 100
    MaxBlockDownloadTime = ChunkSize / (50 * 1024)
    MinBlockSplitTime = ChunkSize / (800 * 1024)

    def __init__(self, **kwargs):
        super(FileInfo, self).__init__(**kwargs)
        self.fp = None
        """:type: FileIO"""
        self.start_downloading_time = None

    def init(self):
        self.start_downloading_time = [0] * len(self.chunks)
        try:
            self.fp = open(self.filename, 'r+b')
        except FileNotFoundError:
            self.fp = open(self.filename, 'w+b')

    @classmethod
    def create_from_download_info(cls, data):
        data = json.loads(data.decode('utf-8'))
        headers = data['headers']
        filename = data['filename']
        filesize = data['content_length']
        url = data['url']
        requests = [Request(url, headers)]
        blocks = [False] * (filesize // cls.ChunkSize)
        if filesize % cls.ChunkSize != 0:
            blocks.append(False)
        return cls.get_or_new(filename, filesize, requests, blocks)

    def save(self):
        DBSession().add(self)
        DBSession().commit()

    @classmethod
    def get(cls, filename):
        obj = DBSession().query(cls).filter(cls.filename == filename).first()
        """:type: FileInfo"""
        if obj:
            obj.init()
        return obj

    def merge_requests(self, requests):
        """
        :type requests: List[Request]
        """
        host_requests_map = {}
        old_requests = self.requests
        assert isinstance(old_requests, list)
        for req in chain(old_requests, requests):
            assert isinstance(req, Request)
            host_requests_map[urllib.parse.urlparse(req.url).hostname] = req

        self.requests = list(host_requests_map.values())

    @classmethod
    def get_or_new(cls, filename, filesize=None, requests=None, blocks=None):
        obj = cls.get(filename)
        if obj:
            if filesize:
                assert obj.filesize == filesize
            if requests:
                obj.merge_requests(requests)
        else:
            assert requests
            assert blocks
            obj = FileInfo(filename=filename, filesize=filesize)
            obj.requests = requests
            obj.chunks = blocks
            obj.init()
        DBSession().add(obj)
        DBSession().commit()
        return obj

    def chunk_is_downloaded_before(self, chunk: FileChunk):
        assert chunk.start % self.ChunkSize == 0
        end = chunk.start + chunk.size
        assert end % self.ChunkSize == 0 or end == self.filesize
        block_i = chunk.start // self.ChunkSize
        return self.chunks[block_i] == True

    def write(self, chunk: FileChunk):
        DBSession().merge(self)
        assert chunk.start % self.ChunkSize == 0
        end = chunk.start + chunk.size
        assert end % self.ChunkSize == 0 or end == self.filesize

        self.fp.seek(chunk.start)
        for tmp in chunk.data:
            self.fp.write(tmp)
        self.fp.flush()
        block_i = chunk.start // self.ChunkSize
        self.chunks[block_i] = True
        logger.debug('dbsession: %s %s', DBSession().dirty, DBSession().new)
        DBSession().commit()

    def _get_undownload_ranges(self):
        start = 0
        rv = []
        while start < len(self.chunks):
            start = find(self.chunks, start, False)
            if start is None:
                break
            end = find(self.chunks, start, True)
            if end is None:
                end = len(self.chunks)
            rv.append((start, end))
            start = end
        return rv

    def get_range(self):
        ranges = [(self.start_downloading_time[x[0]], x) for x in self._get_undownload_ranges()]
        ranges.sort()
        now_ = time.time()
        for range_ in ranges:
            if range_[0] + self.MaxBlockDownloadTime < now_:
                return range_[1]
        for range_ in ranges:
            if range_[0] + self.MinBlockSplitTime < now_ and \
                    range_[1][1] - range_[1][0] >= self.MinBlockLength:
                start = (range_[1][1] - range_[1][0]) // 2 + range_[1][0]
                return start, range_[1][1],

    def get_range_and_mark_downloading_time(self):
        rv = self.get_range()
        if rv:
            self.start_downloading_time[rv[0]] = time.time()
            return Range(rv[0] * self.ChunkSize, rv[1] * self.ChunkSize)
        else:
            return rv

    def is_finished(self):
        return all(self.chunks)

Base.metadata.create_all(engine)
del FileIO
