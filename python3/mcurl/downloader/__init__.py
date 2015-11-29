#!/usr/bin/env python
# coding=utf-8
from sqlalchemy import create_engine
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import scoped_session, sessionmaker

__author__ = 'chenfengyuan'
engine = create_engine('sqlite:///file_info.sqlite')
DBSession = scoped_session(sessionmaker(bind=engine))
Base = declarative_base()
