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
import copy
import os
import pickle
from hashlib import sha1

import env
import log


class MetaFile(object):
    def __init__(self, tag):
        self.path = env.meta_path
        self.tag = tag

    def load(self):
        try:
            meta_cache = copy.copy(env.meta_cache)
            fd = os.open(self.path, os.O_RDONLY | os.O_CREAT, 420)
            with os.fdopen(fd, "rb") as meta_file:
                data_sign = meta_file.read(40)
                assert len(data_sign) == 40, "data_sign length not equal 40"
                data_sign = data_sign.decode("ascii")
                cont = meta_file.read(os.path.getsize(self.path) - 40)
                obj = pickle.loads(cont)
                if isinstance(obj, dict) and obj:
                    cont_sha1 = sha1()
                    cont_sha1.update(cont)
                    if data_sign == cont_sha1.hexdigest():
                        meta_cache.update(obj)
                    else:
                        log.client_logger.warn("meta sign not match,real:%s but expect:%s for tag:%s",
                                               cont_sha1.hexdigest(), data_sign, self.tag)
        except:
            log.client_logger.exception("read meta file fail for tag:%s", self.tag)
        env.meta_cache = meta_cache

    def dump(self):
        try:
            fd = os.open(self.path, os.O_WRONLY | os.O_TRUNC | os.O_CREAT, 420)
            with os.fdopen(fd, "wb") as meta_file:
                cont_sha1 = sha1()
                cont = pickle.dumps(env.meta_cache)
                cont_sha1.update(cont)
                data_sign = cont_sha1.hexdigest()
                meta_file.write(data_sign.encode("ascii"))
                meta_file.write(cont)
        except:
            log.client_logger.exception("write meta file fail for tag:%s", self.tag)
