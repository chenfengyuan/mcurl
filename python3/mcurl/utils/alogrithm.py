#!/usr/bin/env python
# coding=utf-8
__author__ = 'chenfengyuan'


def find(arr: list, start, value):
    for i in range(start, len(arr)):
        if arr[i] == value:
            return i
    else:
        return None
