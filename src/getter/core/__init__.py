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
import env
from component import constants


def get_task_file(task_file_name, data_dir=None):
    if not data_dir:
        data_dir = env.data_dir
    if task_file_name:
        return data_dir + task_file_name


def get_service_file(task_file_name, data_dir=None):
    if task_file_name:
        return get_task_file(task_file_name, data_dir) + ".service"


def get_http_path(task_file_name):
    if task_file_name:
        return constants.PEER_HTTP_PATH_PREFIX + task_file_name


def get_task_name(file_path):
    index = file_path.rfind(".service")
    if index != -1:
        return file_path[:index]
    return file_path


def create_item(task_id, node, dst_cid='', piece_range='',
                result=constants.RESULT_INVALID, status=constants.TASK_STATUS_RUNNING,
                piece_cont=[]):
    return {'dstCid': dst_cid, 'range': piece_range, 'result': result, 'taskId': task_id,
            'superNode': node, 'status': status, "pieceCont": piece_cont}


def get_local_rate(piece_task):
    if env.local_limit:
        return env.local_limit
    return int(piece_task["downLink"]) * 1024
