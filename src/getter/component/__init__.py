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
import os

import constants
import core
import env
import log
import netutil
import paramparser


def filter_url(url):
    task_url = url
    filter_param = paramparser.cmdparam.filter
    if filter_param:
        fields = filter_param.split("&")
        log.client_logger.info("filter fields:%s", fields)

        param_index = url.find("?")
        try:
            if param_index >= 0:
                task_url = url[:param_index + 1]
                params = url[param_index + 1:].split("&")
                task_url_params = []
                for kv in params:
                    if kv.split("=")[0] not in fields:
                        task_url_params.append(kv)
                if task_url_params:
                    for param in task_url_params:
                        if param:
                            task_url += param + "&"
                    if task_url.endswith("&"):
                        task_url = task_url[:-1]
        except:
            log.client_logger.exception("filter url:%s error", url)
    return task_url


def redirect_data_dir(port):
    log.client_logger.info("redirect data dir")
    target_dir = os.path.dirname(env.real_target)
    if target_dir[-1] != os.sep:
        target_dir += os.sep
    env.data_dir = target_dir

    client_path = core.get_task_file(env.task_file_name)
    fd = os.open(client_path, os.O_EXCL | os.O_CREAT)
    try:
        os.close(fd)
    except:
        pass
    env.client_in_user = True
    service_path = core.get_service_file(env.task_file_name)
    fd = os.open(service_path, os.O_EXCL | os.O_CREAT)
    try:
        os.close(fd)
    except:
        pass
    env.service_in_user = True

    if port > 0:
        netutil.request_local(port, constants.LOCAL_HTTP_PATH_CHECK, env.task_file_name,
                              {"dataDir": env.data_dir, "totalLimit": 0})
