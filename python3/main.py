#!/usr/bin/env python
# coding=utf-8
from mcurl.downloader.files_downloader import FilesDownloader
from mcurl.utils.monkey_patch import dummy
import mcurl.utils.logger as logger
import sys


def main():
    files_downloader = FilesDownloader(sys.argv[1:], '127.0.0.1', 9991, 10)
    files_downloader.init_tasks()
    print(files_downloader.exit_event.is_set())
    files_downloader.exit_event.wait()


if __name__ == '__main__':
    del logger
    dummy()
    main()
