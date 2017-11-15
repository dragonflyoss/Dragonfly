# coding=utf8
# Copyright 1999-2017 Alibaba Group.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import exception
import logging
import os
import sys
from logging import StreamHandler
from logging.handlers import RotatingFileHandler

import env
import paramparser

LOG_LEVEL_DEBUG = "DEBUG"
LOG_LEVEL_INFO = "INFO"

LOG_NAME_CLIENT = "client"
LOG_NAME_SERVER = "server"

client_logger = None
server_logger = None


def _build_logger(usr_home, log_name, log_file, log_level, fmt):
    log_dir = usr_home + "logs" + os.sep
    if not os.path.exists(log_dir):
        os.makedirs(log_dir)
    if not os.path.isdir(log_dir):
        raise exception.DirError("dir:%s not found" % log_dir)
    log_path = log_dir + log_file

    tmp_logger = logging.getLogger(log_name)
    tmp_logger.propagate = False
    if log_level == LOG_LEVEL_DEBUG:
        tmp_logger.setLevel(logging.DEBUG)
    else:
        tmp_logger.setLevel(logging.INFO)
    if paramparser.cmdparam.console and log_name == LOG_NAME_CLIENT:
        console_handler = StreamHandler(sys.stdout)
        console_handler.setFormatter(logging.Formatter(fmt))
        tmp_logger.addHandler(console_handler)
    file_handler = RotatingFileHandler(log_path, maxBytes=16777216)
    file_handler.setFormatter(logging.Formatter(fmt))
    tmp_logger.addHandler(file_handler)
    return tmp_logger


def init_log(name):
    global client_logger
    global server_logger
    fmt = '[%(asctime)s] %(levelname)s sign:' + env.execute_sign + ' lineno:%(lineno)d : %(message)s'
    if name == LOG_NAME_CLIENT:
        client_logger = _build_logger(env.usr_home, "client", "dfclient.log", LOG_LEVEL_INFO, fmt)
    elif name == LOG_NAME_SERVER:
        server_logger = _build_logger(env.usr_home, "server", "dfserver.log", LOG_LEVEL_INFO, fmt)
