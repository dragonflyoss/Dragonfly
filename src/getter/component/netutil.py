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
import json
import socket
import time
from urllib2 import Request
from urllib2 import urlopen

import constants
import env
import log


def check_connect(ip, port, retry=1, timeout=0.5):
    while retry:
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.settimeout(timeout)
            s.connect((ip, port))
            return s.getsockname()[0]
        except:
            log.client_logger.exception("connect to ip:%s port:%d fail", ip, port)
        finally:
            try:
                s.close()
            except:
                pass
        retry -= 1
    return None


def request_local(port, path, param, header={}):
    result = ""
    if port > 0:
        start_time = time.time()
        try:
            url = "http://%s:%d%s%s" % (env.ip, port, path, param)
            req = Request(url)

            req.add_header("param", json.dumps(header))

            res = urlopen(req)
            if res.code == constants.SUCCESS:
                result = res.read().decode("UTF-8")
        except:
            log.client_logger.exception("request local http path:%s error", path)
        log.client_logger.info("local http result:%s for path:%s and cost:%.3f", result, path,
                               time.time() - start_time)
    return result
