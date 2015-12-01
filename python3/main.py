#!/usr/bin/env python
# coding=utf-8
from mcurl.downloader.files_downloader import FilesDownloader
from mcurl.utils.monkey_patch import dummy
import mcurl.utils.logger as logger
import sys
import argparse


def main():
    parser = argparse.ArgumentParser(description='mcurl')
    parser.add_argument('-f', '--filename')
    parser.add_argument('tasks', nargs='+')
    args = parser.parse_args()

    files_downloader = FilesDownloader(args.tasks, '127.0.0.1', 9991, 10, args.filename)
    files_downloader.init_tasks()
    print(files_downloader.exit_event.is_set())
    files_downloader.exit_event.wait()


if __name__ == '__main__':
    del logger
    dummy()
    main()
