#!/usr/bin/env python
# coding=utf-8
from enum import Enum
__author__ = 'chenfengyuan'


class AutoNumber(Enum):
    def __new__(cls):
        value = len(cls.__members__) + 1
        obj = object.__new__(cls)
        obj._value_ = value
        return obj


class Classification(AutoNumber):
    FILE_FINISHED = ()
    FILE_NOT_COMPLETE = ()

    REQUEST_DOWNLOAD_REANGE = ()
    DOWNLOAD_RANGE = ()
    NEW_DOWNLOADED_CHUNK = ()
    NO_RANGE_FOR_NOW = ()
    NEXT_CHUNK_WANTED = ()
    CURREN_OR_NEXT_CHUNK_IS_DOWNLOADED = ()
