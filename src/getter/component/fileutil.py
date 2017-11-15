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
import os
import time

import constants
import env
import log
import md5computer


def create_directories(dir_path):
    if not os.path.exists(dir_path):
        os.makedirs(dir_path, 0755)

    if not os.path.isdir(dir_path):
        raise exception.DirError("create dir:%s error" % dir_path)


def delete_file(file_path):
    if not file_path:
        return False
    try:
        if os.path.isfile(file_path):
            os.remove(file_path)
    except:
        log.client_logger.exception("delete file:%s error", file_path)
    if os.path.exists(file_path):
        return False
    return True


def delete_files(*files):
    if len(files) > 0:
        for file_path in files:
            delete_file(file_path)


def open_file(path, mode="wb+", buffer_size=-1):
    file_obj = None
    try:
        if path and not os.path.isdir(path):
            create_directories(os.path.dirname(path))
            file_obj = open(path, mode, buffer_size)
    except:
        log.client_logger.exception("open file:%s fail", path)
    return file_obj


def do_link(src, link_name):
    try:
        if delete_file(link_name):
            os.link(src, link_name)
            return True
    except:
        log.client_logger.exception("can not link %s to %s", link_name, src)
    return False


def copy_file(src, dst):
    try:
        with open(src, "rb") as src_file:
            with open(dst, "wb") as dst_file:
                cont = src_file.read(8 * 1024 * 1024)
                while cont:
                    dst_file.write(cont)
                    cont = src_file.read(8 * 1024 * 1024)
                return True
    except:
        log.client_logger.exception("copy src:%s to dst:%s fail", src, dst)
    return False


def mv_file(src, dst, md5=None):
    start_time = time.time()
    result = False
    if md5:
        m5 = md5computer.Md5Computer()
        real_md5 = m5.md5_file(src)
        if real_md5 != md5:
            env.back_reason = constants.REASON_MD5_NOT_MATCH
            raise exception.Md5NotMatchError("real:%s and expect:%s" % (real_md5, md5))
    try:
        delete_file(dst)
        os.rename(src, dst)
        result = True
    except:
        log.client_logger.exception("rename src:%s to dst:%s", src, dst)
        if copy_file(src, dst):
            delete_file(src)
            result = True
    if result and not os.path.isfile(dst):
        env.back_reason = constants.REASON_HOST_SYS_ERROR
        raise exception.FileIOError("dst:%s is not a file after move but result is success" % dst)
    log.client_logger.info("move src:%s to dst:%s result:%s cost:%.3f", src, dst, result,
                           time.time() - start_time)
    return result
