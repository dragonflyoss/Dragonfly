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
import Queue
import os
import random
import re
import requests
import socket
import sys
import tempfile
import threading
import time

import core
import env
import exception
from component import constants
from component import fileutil
from component import httputil
from component import log
from component import md5computer
from component import netutil
from component import paramparser
from component import ratelimiter
from component import stdshower


class P2PDownloader(threading.Thread):
    def __init__(self, node, task_id, nodes, url, port, finished, result):
        threading.Thread.__init__(self)
        self.node = node
        self.task_id = task_id
        self.finished = finished
        self.result = result
        self.target_file = env.real_target
        self.task_file_name = env.task_file_name
        self.http_path = core.get_http_path(self.task_file_name)

        self.nodes = nodes
        self.url = url
        self.task_url = env.task_url
        self.port = port
        self.md5 = paramparser.cmdparam.md5
        self.identifier = paramparser.cmdparam.identifier

        self.qu = Queue.Queue()
        self.qu.put(core.create_item(task_id, node, status=constants.TASK_STATUS_START))  # first request piece task

        self.success_set = set([])  # success rangeTask set
        self.running_set = set([])  # running rangeTask set

        self.service_file_path = core.get_service_file(self.task_file_name)
        self.client_file_path = core.get_task_file(self.task_file_name)

        self.client_qu = Queue.Queue(constants.QU_CLIENT_SIZE)
        self.finish_ev = threading.Event()
        self.client_writer = ClientWriter(self.task_file_name, self.client_qu, self.finish_ev, env.cid)
        self.client_writer.setDaemon(True)
        self.client_writer.start()

        self.rate_limiter = None
        self.pull_rate_time = None

        self.total = 0

    def pull_rate(self, piece_task):
        if not self.pull_rate_time or time.time() - self.pull_rate_time > 3.0:
            local_rate = core.get_local_rate(piece_task)
            req_rate = netutil.request_local(self.port, constants.LOCAL_HTTP_PATH_RATE, self.task_file_name,
                                             {"rateLimit": local_rate})
            if req_rate:
                local_rate = req_rate
            if local_rate:
                local_rate = int(local_rate)
            if self.rate_limiter:
                self.rate_limiter.refresh(local_rate)
            else:
                self.rate_limiter = ratelimiter.RateLimiter(local_rate)

            self.pull_rate_time = time.time()

    def start_task(self, piece_task):
        tasker = PowerClient(self.task_id, self.node, piece_task, self.qu,
                             self.task_file_name, self.client_qu, self.rate_limiter,
                             self.client_writer)
        tasker.setDaemon(True)
        tasker.start()

    def get_item(self, latest_item):
        need_merge = True
        try:
            cur_item = self.qu.get(True, 2)
            if "pieceSize" in cur_item and cur_item["pieceSize"] != env.piece_size_history[1]:
                return False, latest_item

            if cur_item["superNode"] != self.node:
                cur_item["dstCid"] = ""
                cur_item["superNode"] = self.node
                cur_item["taskId"] = self.task_id

            piece_range = cur_item["range"]
            if piece_range:
                if piece_range in self.running_set:
                    self.running_set.remove(piece_range)
                elif piece_range not in self.success_set:
                    log.client_logger.warn("pieceRange:%s not in runningSet and successSet",
                                           piece_range)
                    return False, latest_item
                if cur_item["result"] in (
                        constants.RESULT_SUC, constants.RESULT_SEMISUC):
                    if piece_range not in self.success_set:
                        for cont in cur_item["pieceCont"]:
                            self.total += len(cont)
                        self.success_set.add(piece_range)
            latest_item = cur_item
        except Queue.Empty:
            log.client_logger.warn("get item timeout(2s) from queue.")
            need_merge = False

            # merge request super
        if latest_item:
            if latest_item["result"] in (
                    constants.RESULT_SUC, constants.RESULT_FAIL,
                    constants.RESULT_INVALID):
                need_merge = False
        else:
            return False, latest_item
        if need_merge and (self.qu.qsize() > 0 or len(self.running_set) > 2):
            return False, latest_item

        return True, latest_item

    def process_piece(self, super_result, cur_item):
        self.refresh(cur_item)

        has_task = False
        if "data" in super_result and isinstance(super_result["data"], list):
            suc_count = 0
            for piece_task in super_result["data"]:
                piece_range = piece_task["range"]
                if piece_range in self.success_set:
                    suc_count += 1
                    self.qu.put(core.create_item(self.task_id, self.node,
                                                 piece_task["cid"], piece_range,
                                                 constants.RESULT_SEMISUC,
                                                 constants.TASK_STATUS_RUNNING))
                    continue
                if piece_range not in self.running_set:
                    # new piece task must be in runningSet
                    self.running_set.add(piece_range)
                    self.pull_rate(piece_task)
                    self.start_task(piece_task)
                    has_task = True
        if not has_task:
            log.client_logger.warn("has not available pieceTask,maybe resource lack")
        if suc_count > 0:
            # async writer report is slow
            log.client_logger.warn("already suc item count:%d after a request super", suc_count)

    def finish_task(self, super_result):
        log.client_logger.info("remaining writed piece count:%d", self.client_qu.qsize())
        self.client_qu.put({"last": ""})
        wait_start = time.time()
        self.finish_ev.wait()
        log.client_logger.info("wait client writer finish cost %.3f,main qu size:%d,client qu size:%d",
                               time.time() - wait_start, self.qu.qsize(), self.client_qu.qsize())
        log.client_logger.info("current alive thread count:%d", threading.activeCount())

        if env.back_reason:
            raise exception.NeedBack()
        super_md5 = super_result["data"]["md5"]
        log.client_logger.info("super down md5:%s", super_md5)

        if self.client_writer.across_write > 0:
            src = env.branch_target
        else:
            if not os.path.exists(self.client_file_path):
                log.client_logger.warn("client file path:%s not found", self.client_file_path)
                if not fileutil.do_link(self.service_file_path, self.client_file_path):
                    fileutil.copy_file(self.service_file_path, self.client_file_path)
            src = self.client_file_path
        if fileutil.mv_file(src, self.target_file, super_md5):
            log.client_logger.info("download successfully from dragonfly")
            self.result["success"] = True
        self.finished.set()

    def refresh(self, cur_item):
        need_reset = False
        if env.piece_size_history[0] != env.piece_size_history[1]:
            env.piece_size_history[0] = env.piece_size_history[1]
            need_reset = True
        if need_reset:
            self.client_qu.put({"reset": ""})
            self.success_set.clear()
            self.running_set.clear()
            self.total = 0
            stdshower.StdShower.reset()
        if self.node != cur_item["superNode"]:
            self.node = cur_item["superNode"]
            self.task_id = cur_item["taskId"]

    def run(self):
        latest_item = None
        while True:
            try:
                go_next, latest_item = self.get_item(latest_item)
                if not go_next:
                    continue

                cur_item = latest_item
                latest_item = None

                cur_item = cur_item.copy()
                del cur_item["pieceCont"]
                super_result = httputil.http_request.pull_piece_task(cur_item, self.nodes, self.url,
                                                                     self.task_url, self.port,
                                                                     self.http_path, self.md5,
                                                                     self.identifier)

                code = super_result["code"]
                if code == constants.TASK_CODE_CONTINUE:
                    self.process_piece(super_result, cur_item)
                elif code == constants.TASK_CODE_FINISH:
                    self.finish_task(super_result)
                    return
                else:
                    log.client_logger.warn("request piece task result:%s", super_result)
                    if code == constants.TASK_CODE_SOURCE_ERROR:
                        env.back_reason = constants.REASON_SOURCE_ERROR
            except:
                log.client_logger.exception("p2p fail")
                if not env.back_reason:
                    env.back_reason = constants.REASON_DOWN_ERROR
            if env.back_reason:
                back_downloader = BackDownloader(self.finished, self.result, self.task_id, self.node)
                back_downloader.setDaemon(True)
                back_downloader.start()
                return


class PowerClient(threading.Thread):
    def __init__(self, task_id, node, piece_task, qu, task_file_name, client_qu, rate_limiter,
                 client_writer):
        threading.Thread.__init__(self)
        self.task_id = task_id
        self.node = node
        self.piece_task = piece_task
        self.task_file_name = task_file_name
        self.qu = qu
        self.client_qu = client_qu
        self.rate_limiter = rate_limiter
        self.client_writer = client_writer

    def run(self):
        total = 0

        piece_range = self.piece_task["range"]
        dst_ip = self.piece_task["peerIp"]

        piece_num = self.piece_task["pieceNum"]
        piece_size = self.piece_task["pieceSize"]
        piece_meta_arr = self.piece_task["pieceMd5"].split(":")
        piece_md5 = piece_meta_arr[0]
        piece_len = int(piece_meta_arr[1])
        piece_start = piece_num * piece_size
        piece_end = piece_start + piece_len - 1
        real_range = str(piece_start) + "-" + str(piece_end)
        if dst_ip == self.node:
            read_time = piece_len / (128.0 * 1024) + 1.0
        else:
            read_time = piece_len / (1.5 * 1024 * 1024) + 1.0
        piece_cont = []

        try:
            if dst_ip == self.node or netutil.check_connect(dst_ip, self.piece_task["peerPort"]):
                url = "http://%s:%d%s" % (
                    dst_ip, self.piece_task["peerPort"], self.piece_task["path"])
                headers = {"Range": "bytes=" + real_range, "pieceNum": str(piece_num), "pieceSize": str(piece_size)}
                start_time = time.time()
                r = requests.get(url, headers=headers, timeout=1.5, stream=True)
                try:
                    for cont in r.iter_content(256 * 1024):
                        if cont:
                            total += len(cont)
                            if time.time() - start_time > read_time:
                                raise exception.ReadTimeoutError()
                            piece_cont.append(cont)
                            self.rate_limiter.acquire(256 * 1024)
                finally:
                    try:
                        r.close()
                    except:
                        log.client_logger.exception("close requests response fail")
                read_finish = time.time()
                m5 = md5computer.Md5Computer()
                for tmp_cont in piece_cont:
                    m5.update(tmp_cont)

                if m5.md5() == piece_md5:
                    item = core.create_item(self.task_id, self.node, self.piece_task["cid"], piece_range,
                                            constants.RESULT_SEMISUC,
                                            constants.TASK_STATUS_RUNNING, piece_cont)
                    item["pieceSize"] = piece_size
                    item["pieceNum"] = piece_num

                    self.client_qu.put(item)

                    self.qu.put(item)

                    end_time = time.time()
                    if end_time - start_time > 2.0:
                        log.client_logger.warn(
                            "client range:%s cost:%.3f from peer:%s,its readCost:%.3f,cont length:%d",
                            piece_range, end_time - start_time, dst_ip, read_finish - start_time, total)
                    return
                else:
                    log.client_logger.error(
                        "piece range:%s error,realMd5:%s,expectedMd5:%s,dstIp:%s,total:%d",
                        piece_range, m5.md5(), piece_md5,
                        dst_ip, total)
        except Exception:
            e_msg = sys.exc_info()[1]
            log.client_logger.error("read piece cont error:%s from dst:%s", e_msg, dst_ip)
            if dst_ip == self.node:
                if str(e_msg).lower().find(constants.RANGE_NOT_EXIST_DESC) != -1:
                    time.sleep(random.uniform(1, 3))
                    log.client_logger.info("sleep (1~3)s because range:%s from %s not exist", piece_range, dst_ip)

        self.qu.put(core.create_item(self.task_id, self.node, self.piece_task["cid"],
                                     piece_range, constants.RESULT_FAIL,
                                     constants.TASK_STATUS_RUNNING))


class ClientWriter(threading.Thread):
    def __init__(self, task_file_name, client_qu, finish_ev, cid):
        threading.Thread.__init__(self)
        self.qu = client_qu
        self.finish_ev = finish_ev
        self.cid = cid
        self.service_file = core.get_service_file(task_file_name)
        self.file_obj = open(self.service_file, "wb", 4 * 1024 * 1024)

        self.across_write = 0  # not across
        client_file = core.get_task_file(task_file_name)
        if not fileutil.do_link(env.branch_target, client_file):
            self.across_write = 2  # force across

        fileutil.do_link(self.service_file, client_file)

        self.result = True

        self.target_qu = Queue.Queue(constants.QU_CLIENT_SIZE)
        self.target_ev = threading.Event()
        target_writer = TargetWriter(env.branch_target, self.target_qu, self.target_ev)
        target_writer.setDaemon(True)
        target_writer.start()

        self.piece_index = 0

        self.sync_qu = None
        if hasattr(os, "fdatasync"):
            self.sync_qu = Queue.Queue()
            start_sync(self.sync_qu)

    def do_write(self, item, writed_file, start_time=None):
        self.piece_index += 1
        piece_cont = item["pieceCont"]
        total = 0
        start = item["pieceNum"] * (item["pieceSize"] - 5)
        writed_file.seek(start, 0)
        cont_count = len(piece_cont)
        for i in range(cont_count):
            if i == 0:
                if i < cont_count - 1:
                    processedcont = piece_cont[0][4:]
                else:
                    processedcont = piece_cont[0][4:-1]
            elif i < cont_count - 1:
                processedcont = piece_cont[i]
            else:
                processedcont = piece_cont[i][:-1]
            writed_file.write(processedcont)
            total += len(processedcont)

        writed_file.flush()

        if start_time:
            self.suc_piece(item["taskId"], item["dstCid"], item["range"], item["superNode"], start_time)

        if self.across_write > 0:
            self.target_qu.put(item)
        else:
            stdshower.StdShower.update(total)
            if self.piece_index % 4 == 0 and self.sync_qu:
                self.sync_qu.put(writed_file.fileno())

    def suc_piece(self, task_id, dst_cid, range_str, super_node, start_time):
        httputil.http_request.suc_piece(task_id, self.cid, dst_cid, range_str, super_node)
        if time.time() - start_time > 2.0:
            log.client_logger.info("async writer and report suc from dst:%s... cost:%.3f for range:%s",
                                   dst_cid[:25],
                                   time.time() - start_time,
                                   range_str)

    def run(self):
        while True:
            try:
                item = self.qu.get()
                if "last" in item:
                    if self.across_write <= 0:
                        try:
                            if hasattr(os, "fsync"):
                                os.fsync(self.file_obj.fileno())
                        except:
                            pass
                    break

                if "reset" in item:
                    self.file_obj.truncate(0)
                    if self.across_write > 0:
                        self.target_qu.put(item)
                    continue

                if item["pieceSize"] != env.piece_size_history[1]:
                    continue

                if not self.result:
                    continue

                self.do_write(item, self.file_obj, time.time())

            except Exception, e:
                if "pieceCont" in item:
                    del item["pieceCont"]
                log.client_logger.error("write item:%s error:%s", item, e.message)
                env.back_reason = constants.REASON_WRITE_ERROR
                self.result = False

        try:
            self.file_obj.close()
        except:
            pass

        try:
            self.target_qu.put({"last": ""})
            self.target_ev.wait()
        except:
            log.client_logger.exception("wait target event error")

        self.finish_ev.set()


def start_sync(sync_qu):
    sync_writer = SyncWriter(sync_qu)
    sync_writer.setDaemon(True)
    sync_writer.start()


class TargetWriter(threading.Thread):
    def __init__(self, dst, item_qu, finish_ev):
        threading.Thread.__init__(self)
        self.dst = dst
        self.item_qu = item_qu
        self.finish_ev = finish_ev
        self.file_obj = fileutil.open_file(dst, "wb", 4 * 1024 * 1024)
        if not self.file_obj:
            raise Exception("open target file:%s fail." % dst)

        self.fd = self.file_obj.fileno()

        self.piece_index = 0

        self.result = True

        self.sync_qu = None
        if hasattr(os, "fdatasync"):
            self.sync_qu = Queue.Queue()
            start_sync(self.sync_qu)

    def get_raw_item(self, item):
        raw_cont = []
        piece_cont = item["pieceCont"]

        cont_count = len(piece_cont)
        for i in range(cont_count):
            if i == 0:
                if i < cont_count - 1:
                    processedcont = piece_cont[0][4:]
                else:
                    processedcont = piece_cont[0][4:-1]
            elif i < cont_count - 1:
                processedcont = piece_cont[i]
            else:
                processedcont = piece_cont[i][:-1]

            raw_cont.append(processedcont)

        item1 = item.copy()
        item1["pieceCont"] = raw_cont

        return item1

    def do_write(self, file_obj, item):
        self.piece_index += 1
        piece_cont = item["pieceCont"]
        start = item["pieceNum"] * (item["pieceSize"] - 5)
        file_obj.seek(start, 0)
        total = 0
        for cont in piece_cont:
            file_obj.write(cont)
            total += len(cont)
        file_obj.flush()

        stdshower.StdShower.update(total)

        if self.piece_index % 4 == 0 and self.sync_qu:
            self.sync_qu.put(self.fd)

    def run(self):
        while True:
            try:
                item = self.item_qu.get()
                if "last" in item:
                    try:
                        if hasattr(os, "fsync"):
                            os.fsync(self.fd)
                    except Exception:
                        pass
                    break

                if not self.result:
                    continue

                if "reset" in item:
                    self.file_obj.truncate(0)
                    continue

                self.do_write(self.file_obj, self.get_raw_item(item))

            except Exception, e:
                if "pieceCont" in item:
                    del item["pieceCont"]
                log.client_logger.error("write item:%s error:%s", item, e.message)
                env.back_reason = constants.REASON_WRITE_ERROR
                self.result = False

        try:
            self.file_obj.close()
        except Exception:
            pass
        self.finish_ev.set()


class SyncWriter(threading.Thread):
    def __init__(self, sync_qu):
        threading.Thread.__init__(self)
        self.sync_qu = sync_qu

    def run(self):
        while True:
            try:
                fd = self.sync_qu.get()
                os.fdatasync(fd)
                try:
                    while True:
                        self.sync_qu.get_nowait()
                except Queue.Empty:
                    pass
            except Exception:
                pass


class BackDownloader(threading.Thread):
    def __init__(self, finished, result, task_id="UNKNOWN", node="UNKNOWN"):
        threading.Thread.__init__(self)
        self.url = paramparser.cmdparam.url
        self.target = env.real_target
        self.md5 = paramparser.cmdparam.md5
        self.finished = finished
        self.result = result
        self.task_id = task_id
        self.node = node
        self.total = 0

    def run(self):
        if paramparser.cmdparam.notbs or env.back_reason == constants.REASON_NO_SPACE:
            log.client_logger.info("download fail and not back source")
            env.back_reason += constants.REASON_ADDITION
        else:
            try:
                socket.setdefaulttimeout(None)  # None is also ok
                log.client_logger.info("start download %s from the source station", os.path.basename(self.target))
                stdshower.StdShower.print_info("download from source")
                stdshower.StdShower.reset()
                if self.md5:
                    m5 = md5computer.Md5Computer()

                rate_limiter = ratelimiter.RateLimiter(
                    env.local_limit if env.local_limit else 10 * 1024 * 1024)

                fd, name = tempfile.mkstemp(suffix=".backsource", dir=os.path.dirname(self.target))
                with os.fdopen(fd, "wb+", 8 * 1024 * 1024) as f:
                    r = requests.get(self.url, headers=fill_headers(), timeout=3.0, stream=True)
                    try:
                        for cont in r.iter_content(512 * 1024):
                            if cont:
                                self.total += len(cont)
                                if self.md5:
                                    m5.update(cont)
                                f.write(cont)
                                stdshower.StdShower.update(len(cont))
                            rate_limiter.acquire(512 * 1024)
                    finally:
                        try:
                            r.close()
                        except:
                            log.client_logger.exception("close requests response fail")
                if not self.md5 or (self.md5 == m5.md5()):
                    self.result["success"] = fileutil.mv_file(name, self.target)
            except:
                log.client_logger.exception("back down fail")

        self.finished.set()


def fill_headers():
    headers = {}
    if paramparser.cmdparam.header:
        for one_head in paramparser.cmdparam.header:
            head_arr = re.split("\\s*:\\s*", one_head, 1)
            head_arr[0], head_arr[1] = head_arr[0].strip(), head_arr[1].strip()
            if head_arr[0] in headers:
                if head_arr[1]:
                    headers[head_arr[0]] += "," + head_arr[1]
            else:
                headers[head_arr[0]] = head_arr[1]
    return headers
