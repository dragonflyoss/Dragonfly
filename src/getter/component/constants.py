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
import version

VERSION = version.v

SUCCESS = 200  # http code and result code

# report result
RESULT_FAIL = 500
RESULT_SUC = 501
RESULT_INVALID = 502  # unknown
RESULT_SEMISUC = 503

# report status
TASK_STATUS_START = 700
TASK_STATUS_RUNNING = 701
TASK_STATUS_FINISH = 702

# response task code
TASK_CODE_FINISH = 600  # task finish
TASK_CODE_CONTINUE = 601  # continue download
TASK_CODE_WAIT = 602  # wait task continual
TASK_CODE_LIMITED = 603  # peer down limited
TASK_CODE_NEED_AUTH = 608  # need auth
TASK_CODE_WAIT_AUTH = 609  # wait auth

SCHEMA_HTTP = "http"  # default request schema

SERVER_PORT_DOWN = 15000
SERVER_PORT_UP = 65000  # decrease to 15000

RANGE_NOT_EXIST_DESC = "range not satisfiable"
ADDR_USED_DESC = 'address already in use'

PEER_HTTP_PATH_PREFIX = "/peer/file/"
CDN_PATH_PREFIX = "/qtdown/"

LOCAL_HTTP_PATH_CHECK = "/check/"
LOCAL_HTTP_PATH_CLIENT = "/client/"
LOCAL_HTTP_PATH_RATE = "/rate/"

REASON_NONE = 0
REASON_REGISTER_FAIL = 1
REASON_MD5_NOT_MATCH = 2
REASON_DOWN_ERROR = 3
REASON_NO_SPACE = 4
REASON_INIT_ERROR = 5
REASON_WRITE_ERROR = 6
REASON_HOST_SYS_ERROR = 7

REASON_ADDITION = 1000  # reason code + this when force not back source

QU_CLIENT_SIZE = 6
