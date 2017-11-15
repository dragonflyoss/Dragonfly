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
import time
from hashlib import md5


import log


class Md5Computer(object):
    def __init__(self):
        self.m5 = md5()

    def update(self, data):
        if data:
            self.m5.update(data)  # byte array

    def md5(self):
        return self.m5.hexdigest()

    def md5_file(self, path):
        start_time = time.time()
        md5_value = ""
        try:
            with open(path, "rb") as f:
                cont = f.read(4 * 1024 * 1024)
                while cont:
                    self.update(cont)
                    cont = f.read(4 * 1024 * 1024)
                md5_value = self.md5()
        except:
            log.client_logger.exception("compute md5 error for file:%s", path)

        log.client_logger.info("compute raw md5:%s for file:%s cost:%.3f", md5_value, path,
                               time.time() - start_time)
        return md5_value
