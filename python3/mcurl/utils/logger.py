# coding=utf-8
import logging
import logging.handlers
logger = logging.getLogger('mcurl')
handler = logging.StreamHandler()
handler.setLevel(logging.DEBUG)
fmt = logging.Formatter(fmt='%(levelname)s[%(process)d][%(filename)s:%(lineno)d]%(message)s')
handler.setFormatter(fmt)
logger.addHandler(handler)
logger.propagate = False
logger.setLevel(logging.DEBUG)
del logging
