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
import random
import re
import socket
import subprocess
import tempfile
import time

import component
import core
import exception
from component import configutil
from component import constants
from component import fileutil
from component import httputil
from component import log
from component import netutil
from component import paramparser
from core import server


def compute_limit(rate_limit):
    matcher = re.match(r'^(\d+)([kKmM])$', rate_limit)
    if not matcher:
        raise exception.ParamError("--locallimit or -s format is invalid")
    if matcher.group(2) in ("k", "K"):
        return int(matcher.group(1)) * 1024
    else:
        return int(matcher.group(1)) * 1024 * 1024


start_time = time.time()
usr_home = os.path.expanduser('~' + os.sep + '.small-dragonfly' + os.sep)
data_dir = None
system_data_dir = usr_home + "data" + os.sep
real_target = None
branch_target = None  # same dir with real target
meta_path = usr_home + "meta" + os.sep + "host.meta"

ip = ""
meta_cache = {}

sys_lang = "UTF-8"

host_name = socket.getfqdn()

local_limit = compute_limit(paramparser.cmdparam.locallimit) if paramparser.cmdparam.locallimit else 0
total_limit = compute_limit(paramparser.cmdparam.totallimit) if paramparser.cmdparam.totallimit else 0
execute_sign = "%d-%.3f" % (os.getpid(), start_time)

task_file_name = ""
cid = ""
task_url = ""
nodes = None

call_system = paramparser.cmdparam.callsystem if paramparser.cmdparam.callsystem else "UNKNOWN"
back_reason = constants.REASON_NONE

download_pattern = paramparser.cmdparam.pattern if paramparser.cmdparam.pattern else "p2p"

file_length = -1

piece_size_history = [4 * 1024 * 1024, 4 * 1024 * 1024]
client_in_user = False
service_in_user = False


def register(port, http_path, md5, identifier):
    node, task_id, http_length, piece_size, url = httputil.http_request.register(
        nodes, paramparser.cmdparam.url, task_url, port, http_path, md5, identifier)
    return {"node": node, "nodes": nodes, "taskId": task_id, "httpLength": http_length,
            "pieceSize": piece_size,
            "url": url, "port": port}


def init():
    global data_dir
    global real_target
    global branch_target
    global ip
    global sys_lang
    global task_file_name
    global cid
    global task_url
    global back_reason
    global file_length
    global nodes

    fileutil.create_directories(usr_home)
    data_dir = system_data_dir
    fileutil.create_directories(system_data_dir)
    real_target = paramparser.cmdparam.output
    log.client_logger.info("target file path:%s", real_target)
    target_dir = os.path.dirname(real_target)
    fileutil.create_directories(target_dir)
    branch_target = make_branch_target(target_dir)
    fileutil.create_directories(os.path.dirname(meta_path))

    nodes = paramparser.cmdparam.node.split(",") if paramparser.cmdparam.node else None
    if not nodes:
        try:
            nodes = configutil.client_config["node"]["address"].split(",")
        except:
            log.client_logger.exception("/etc/dragonfly.conf not found or has not data")
            raise
    if nodes:
        random.shuffle(nodes)
        nodes *= 2
        ip = parse_super()

    if not ip:
        back_reason = constants.REASON_NODE_EMPTY
        raise exception.DownError("nodes is invalid")

    sys_lang = os.getenv("LANG", "en_US.UTF-8")
    comma_index = sys_lang.rfind(".")
    if comma_index >= 0:
        sys_lang = sys_lang[comma_index + 1:]
    log.client_logger.info("sysLang:%s", sys_lang)

    task_file_name = os.path.basename(real_target) + "-" + execute_sign
    cid = ip + "-" + execute_sign
    log.client_logger.info("taskFileName:%s and cid:%s", task_file_name, cid)

    task_url = component.filter_url(paramparser.cmdparam.url)
    log.client_logger.info("task url:%s", task_url)

    port = 0
    if download_pattern == "p2p":
        port = server.launch(task_file_name)

    try:
        register_result = register(port, core.get_http_path(task_file_name), paramparser.cmdparam.md5,
                                   paramparser.cmdparam.identifier)
    except SystemExit:
        raise
    except:
        back_reason = constants.REASON_REGISTER_FAIL
        raise

    file_length = register_result["httpLength"]
    piece_size_history[0], piece_size_history[1] = register_result["pieceSize"], register_result["pieceSize"]

    assert_space(port)

    return register_result


def make_branch_target(target_dir):
    if target_dir[-1] != os.sep:
        target_dir += os.sep
    try:
        fd, name = tempfile.mkstemp("", "dfget-" + execute_sign, target_dir)
    except:
        name = target_dir + "dfget-" + execute_sign
        fd = os.open(name, os.O_CREAT | os.O_EXCL)
    try:
        os.close(fd)
    except:
        pass
    return name


def assert_space(port):
    global back_reason
    if file_length >= 0:
        data_dir_space = available_space(data_dir)
        if 0 < data_dir_space <= file_length + 100 * 1024 * 1024:
            component.redirect_data_dir(port)
            data_dir_space = available_space(data_dir)
            if 0 < data_dir_space <= 2 * file_length + 100 * 1024 * 1024:
                back_reason = constants.REASON_NO_SPACE
                raise exception.SpaceLackError(
                    "space lack,free:%d want:%d(2*fileLength)" % (data_dir_space, 2 * file_length))
            else:
                return
        app_dir_space = available_space(os.path.dirname(real_target))
        if 0 < app_dir_space <= file_length + 100 * 1024 * 1024:
            back_reason = constants.REASON_NO_SPACE
            raise exception.SpaceLackError("space lack,free:%d want:%d" % (app_dir_space, file_length))


def available_space(target_dir):
    try:
        if hasattr(os, "statvfs"):
            fs_info = os.statvfs(target_dir)
            return int(fs_info[0]) * int(fs_info[4])
    except:
        log.client_logger.exception("statvfs:%s error", target_dir)
        try:
            p = subprocess.Popen("df %s" % target_dir, shell=True, stdout=subprocess.PIPE,
                                 stderr=subprocess.STDOUT)

            pout, _ = p.communicate()
            lines = pout.decode(sys_lang).splitlines()
            disk_info = re.split("\s+", lines[-1].strip())
            return int(disk_info[-3]) * 1024
        except:
            log.client_logger.exception("df %s error", target_dir)
    return -1


def parse_super():
    nodes_len = len(nodes) if nodes else 0
    while nodes_len > 0:
        node = nodes.pop(0)
        nodes_len -= 1
        if node:
            addr_fields = node.split(":")
            if len(addr_fields) == 1:
                addr_fields.append(8002)
            local_ip = netutil.check_connect(addr_fields[0], int(addr_fields[1]), timeout=2)
            if local_ip:
                nodes.insert(0, node)
                return local_ip
    return None
