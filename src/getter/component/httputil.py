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
import random
import requests
import sys
import time
from requests.adapters import HTTPAdapter

import constants
import env
import log
import paramparser
import stdshower


class HttpRequests(object):
    def __init__(self):
        session = requests.Session()
        session.mount('http://', HTTPAdapter(max_retries=2))
        session.mount('https://', HTTPAdapter(max_retries=2))
        self.session = session

    def register(self, nodes, url, task_url, port, http_path, md5=None, identifier=None,
                 schema=constants.SCHEMA_HTTP):
        start_time = time.time()
        params = {"rawUrl": url, "taskUrl": task_url}
        if md5:
            params["md5"] = md5
        elif identifier:
            params["identifier"] = identifier
        params["version"] = constants.VERSION
        params["port"] = port
        params["path"] = http_path
        params["callSystem"] = env.call_system
        params["cid"] = env.cid
        params["ip"] = env.ip
        params["hostName"] = env.host_name
        if paramparser.cmdparam.header:
            params["headers"] = paramparser.cmdparam.header

        params["dfdaemon"] = "false"
        if paramparser.cmdparam.dfdaemon:
            params["dfdaemon"] = "true"

        result = None
        node_len = len(nodes) if nodes else 0
        while node_len > 0:
            node_len -= 1
            try:
                node = nodes.pop(0)
                params["superNodeIp"] = node
                log.client_logger.info("do register to %s,remainder:%s", node, nodes)
                while True:
                    r = self.session.post("%s://%s:8002/peer/registry" % (
                        schema, node), data=params, timeout=(2.0, 5.0))
                    r.raise_for_status()
                    r.encoding = "UTF-8"
                    result = r.json()
                    if result["code"] == constants.TASK_CODE_WAIT_AUTH:
                        time.sleep(2.5)
                        log.client_logger.info("wait auth...")
                    else:
                        break

                if result["code"] in (constants.SUCCESS, constants.TASK_CODE_NEED_AUTH):
                    break

            except:
                log.client_logger.exception("register to node:%s error", node)

        if result and result["code"] == constants.TASK_CODE_NEED_AUTH:
            sys.exit(22)
        if not result or result["code"] != constants.SUCCESS:
            raise Exception("register result:%s" % result)

        data = result["data"]
        task_id = data["taskId"]
        file_length = data["fileLength"]
        piece_size = data["pieceSize"]

        log.client_logger.info("do register result:%s and cost %.3f", result, time.time() - start_time)
        return node, task_id, file_length, piece_size, url

    def pull_piece_task(self, cur_item, nodes, url, task_url, port, http_path, md5,
                        identifier, schema=constants.SCHEMA_HTTP):
        cur_item["srcCid"] = env.cid
        result = None
        while True:
            try:
                r = self.session.get("%s://%s:8002/peer/task" % (
                    schema, cur_item["superNode"]), params=cur_item, timeout=(2.0, 3.0))
                r.raise_for_status()
                r.encoding = "UTF-8"
                result = r.json()
                if result["code"] == constants.TASK_CODE_WAIT:
                    sleep_time = random.uniform(0.6, 2.0)
                    log.client_logger.info("pull piece task result:%s and sleep %.3f ...", result,
                                           sleep_time)
                    time.sleep(sleep_time)
                    continue
            except:
                log.client_logger.exception("pull piece task error")
            break
        if not result or result["code"] not in (
                constants.TASK_CODE_CONTINUE, constants.TASK_CODE_FINISH,
                constants.TASK_CODE_LIMITED, constants.SUCCESS):
            log.client_logger.error("pull piece task fail:%s and will migrate", result)
            cur_item["superNode"], cur_item["taskId"], file_length, piece_size, url = self.register(nodes,
                                                                                                    url,
                                                                                                    task_url,
                                                                                                    port,
                                                                                                    http_path,
                                                                                                    md5,
                                                                                                    identifier)
            del file_length
            env.piece_size_history[1] = piece_size
            cur_item["status"] = constants.TASK_STATUS_START
            stdshower.StdShower.print_info("migrated to node:%s" % cur_item["superNode"])
            return self.pull_piece_task(cur_item, nodes, url, task_url, port,
                                        http_path, md5, identifier)
        return result

    # report service downing
    def down_service(self, task_id, cid, node, schema=constants.SCHEMA_HTTP):
        try:
            if node and node != "UNKNOWN" and task_id and task_id != "UNKNOWN":
                params = {"taskId": task_id, "cid": cid}
                self.session.get("%s://%s:8002/peer/service/down" % (schema, node), params=params, timeout=(2.0, 3.0))
        except:
            log.client_logger.exception("down service error")
        self.close()

    # report service suc
    def suc_piece(self, task_id, cid, dst_cid, piece_range, node,
                  schema=constants.SCHEMA_HTTP):
        try:
            params = {"taskId": task_id, "cid": cid, "dstCid": dst_cid, "pieceRange": piece_range}
            self.session.get("%s://%s:8002/peer/piece/suc" % (schema, node), params=params, timeout=(2.0, 3.0))
        except:
            log.client_logger.exception("suc piece error")

    def close(self):
        try:
            self.session.close()
        except:
            pass


http_request = HttpRequests()
