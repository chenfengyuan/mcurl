#!/usr/bin/env python
# coding=utf-8
from mcurl.utils import monkey_patch
import zmq.green as zmq
import sys
__author__ = 'chenfengyuan'

monkey_patch.dummy()


class DownloadInfo:
    def __init__(self, host, port):
        self.context = zmq.Context()
        self.socket = self.context.socket(zmq.REQ)
        self.socket.connect("tcp://%s:%s" % (host, port))

    def get_info(self, num):
        """
        :type num: bytes
        """
        self.socket.send(num)
        msg = self.socket.recv()
        return msg

    def __del__(self):
        self.socket.close()
