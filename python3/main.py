#!/usr/bin/env python
# coding=utf-8
from mcurl.downloader import FilesDownloader
import sys


def main():
    files_downloader = FilesDownloader(sys.argv[1:], '127.0.0.1', 9991, 10)
    files_downloader.start_download()


if __name__ == '__main__':
    main()
