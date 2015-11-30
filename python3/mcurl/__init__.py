#!/usr/bin/env python
# coding=utf-8
# no proxy
import urllib.request
urllib.request.getproxies_environment = lambda: {'_': '_'}
__author__ = 'chenfengyuan'
