#!/usr/bin/env python
# coding=utf-8
from mcurl.utils import monkey_patch
import zmq.green as zmq
import sys
import json
__author__ = 'chenfengyuan'

monkey_patch.dummy()


def client(port):
    host = '127.0.0.1'
    context = zmq.Context()

    socket = context.socket(zmq.REQ)
    socket.connect("tcp://%s:%s" % (host, port))

    socket.send(b'')
    message = socket.recv()
    socket.close()
    data = json.loads(message.decode('utf-8'))
    """:type: list"""
    data.sort(key=lambda x: x['filename'])
    for info in data:
        print('%s %.3fGB %.2f%%' % (info['filename'], info['filesize'] / 1024 / 1024 / 1024, info['percentage'] * 100))


def main():
    port = int(sys.argv[1])
    client(port)

if __name__ == '__main__':
    main()

