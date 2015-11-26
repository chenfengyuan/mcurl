#!/usr/bin/env python
# coding=utf-8

from sqlalchemy import (
    Column,
    String,
    Integer,
    DateTime,
    PickleType
)
from sqlalchemy import create_engine
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker, scoped_session
from sqlalchemy.orm.attributes import flag_modified
from sqlalchemy.sql.functions import now
from collections import namedtuple
from mcurl.utils.alogrithm import find
from io import FileIO
import time
import json
import urllib.parse
from itertools import chain
Base = declarative_base()
engine = create_engine('sqlite:///file_info.sqlite')

__author__ = 'chenfengyuan'

DBSession = scoped_session(sessionmaker(bind=engine))

Request = namedtuple('Request', ('url', 'headers'))


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
    __mutable_fields__ = {'requests', 'blocks'}

    id = Column(Integer, primary_key=True, nullable=False)
    filename = Column(String, nullable=False, unique=True, index=True)
    filesize = Column(Integer, nullable=False, default=0)
    requests = Column(PickleType, nullable=False)
    """:type: List[Request]"""
    blocks = Column(PickleType, nullable=False)
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

    def sync(self):
        for field in self.__mutable_fields__:
            flag_modified(self, field)

    @classmethod
    def get(cls, filename):
        obj = DBSession().query(cls).filter(cls.filename == filename).first()
        """:type: FileInfo"""
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
            obj.blocks = blocks
            DBSession().add(obj)
            DBSession().commit()
        return obj

    def write(self, chunk: FileChunk):
        assert chunk.start % self.ChunkSize == 0
        end = chunk.start + len(chunk.size)
        assert end % self.ChunkSize == 0 or end == self.filesize

        self.fp.seek(chunk.start)
        for tmp in chunk.data:
            self.fp.write(tmp)
        self.fp.flush()

        block_i = chunk.start // self.ChunkSize
        while block_i * self.ChunkSize < end:
            self.blocks[block_i] = True
        DBSession().add(self)
        DBSession().commit()

    def _get_undownload_ranges(self):
        start = 0
        rv = []
        while True:
            start = find(self.blocks, start, False)
            if start is None:
                break
            end = find(self.blocks, start, True)
            if end is None:
                end = len(self.blocks)
            rv.append((start, end))
        return rv

    def get_ranges(self):
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

Base.metadata.create_all(engine)
del FileIO
