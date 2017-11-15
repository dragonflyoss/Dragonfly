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
import json
import os
import pickle
import struct
import sys
import thread
import threading
import time
from BaseHTTPServer import BaseHTTPRequestHandler
from SocketServer import ThreadingTCPServer

import core
import env
from component import constants
from component import fileutil
from component import httputil
from component import log
from component import metafile
from component import netutil
from component import ratelimiter

alive_qu = Queue.Queue()
rate_limiter = None

checker_lock = threading.RLock()
gc_lock = threading.RLock()


def check_port(port, task_file_name):
    result = netutil.request_local(port, constants.LOCAL_HTTP_PATH_CHECK,
                                   task_file_name, {"dataDir": env.data_dir,
                                                    "totalLimit": env.total_limit})
    if result:
        if result.endswith("@" + constants.VERSION):
            return result[:len(result) - len("@" + constants.VERSION)]
        else:
            log.server_logger.warn("checked result:%s for client version:%s", result,
                                   constants.VERSION)
    return ""


def generate_port(dport=constants.SERVER_PORT_DOWN,
                  uport=constants.SERVER_PORT_UP):
    port = int(time.time() / 300) % (uport - dport) + dport
    while True:
        yield port
        port += 1


def trans_to_daemon(readfd):
    try:
        os.close(readfd)
        os.close(0)
        os.close(1)
        os.close(2)
        sys.stdout = sys.stderr = open(os.devnull, 'w')
    except:
        log.server_logger.exception("close std error")
    pid = 0
    try:
        os.chdir("/")
        if hasattr(os, "setsid"):
            os.setsid()
        os.umask(0)
        pid = os.fork()
    except:
        log.server_logger.exception("trans to daemon error")
    if pid > 0:
        log.server_logger.info("trans to daemon and its pid is %d", pid)
        os._exit(0)


def server_gc(interval):
    log.server_logger.info("start server gc")
    while True:
        try:
            for cur, dirs, files in os.walk(env.system_data_dir):
                del dirs
                for file_name in files:
                    gc_lock.acquire()
                    try:
                        file_path = os.path.join(cur, file_name)
                        expire_time = 180
                        exist = False
                        task_name = core.get_task_name(file_name)
                        if sync_task_map.has(task_name):
                            exist = True
                            if not sync_task_map.read(task_name, "finished"):
                                continue
                        else:
                            expire_time = 3600
                        fd = os.open(file_path, os.O_RDONLY)
                        os.fsync(fd)
                        os.close(fd)
                        hittime = os.path.getatime(file_path)
                        if hittime < os.path.getmtime(file_path):
                            hittime = os.path.getmtime(file_path)
                        if time.time() - hittime > expire_time:
                            log.server_logger.info("delete expired:(%ds) file:%s", expire_time, file_path)
                            fileutil.delete_file(file_path)
                            if exist:
                                httputil.http_request.down_service(sync_task_map.read(task_name, "taskId"),
                                                                   sync_task_map.read(task_name, "cid"),
                                                                   sync_task_map.read(task_name, "superNode"))
                                sync_task_map.delete(task_name)
                    except:
                        pass
                    finally:
                        gc_lock.release()
            time.sleep(interval)
        except:
            pass


def check_alive(interval):
    log.server_logger.info("checking alive")
    thread.start_new_thread(server_gc, (interval,))
    while True:
        try:
            alive_qu.get(True, 5 * 60.0)
        except:
            log.server_logger.info("acquire checker lock")
            checker_lock.acquire()
            try:
                if alive_qu.qsize() > 0:
                    continue

                meta_file = metafile.MetaFile("finishService")
                meta_file.load()
                if "servicePort" in env.meta_cache:
                    del env.meta_cache["servicePort"]
                meta_file.dump()
            finally:
                checker_lock.release()
                log.server_logger.info("release checker lock")

            log.server_logger.info("acquire gc lock")
            gc_lock.acquire()
            log.server_logger.info("server down")
            os._exit(0)


def send_port(wf, port):
    os.write(wf, pickle.dumps(port))
    os.close(wf)


def launch(task_file_name):
    """
    launch server to send piece data

    :param task_file_name:
    :return: port
    """
    log.init_log(log.LOG_NAME_SERVER)

    port = 0
    pid = -1
    try:
        if hasattr(os, "fork"):
            meta_file = metafile.MetaFile("checkService")
            meta_file.load()
            if "servicePort" in env.meta_cache:
                result = check_port(env.meta_cache["servicePort"], task_file_name)
                if result == task_file_name:
                    port = env.meta_cache["servicePort"]
                    log.server_logger.info("reuse exist service with port:%d", port)
                else:
                    log.server_logger.warn("not found process on port:%d,version:%s", env.meta_cache["servicePort"],
                                           constants.VERSION)
            if port <= 0:
                rf, wf = os.pipe()  # port pipe
                pid = os.fork()
                # child process
                if not pid:
                    try:
                        log.server_logger.info("******************************")
                        log.server_logger.info("server process is loading")

                        trans_to_daemon(rf)

                        thread.start_new_thread(check_alive, (15,))

                        # child
                        for tmp_port in generate_port():
                            try:
                                server_address = (env.ip, tmp_port)
                                httpd = P2PServer(server_address, SimpleHttpRequestHandler)
                                sa = httpd.socket.getsockname()
                                log.server_logger.info("server on %s port %d", sa[0], sa[1])
                                send_port(wf, tmp_port)
                                httpd.serve_forever()
                                break
                            except Exception:
                                e_msg = sys.exc_info()[1]
                                if (str(e_msg).lower()).find(constants.ADDR_USED_DESC) == -1:
                                    raise
                                elif check_port(tmp_port, task_file_name) == task_file_name:
                                    send_port(wf, tmp_port)
                                    log.server_logger.info("reuse exist service with port:%d", tmp_port)
                                    break
                    except:
                        log.server_logger.exception("load server fail")

                    # server process direct exit
                    log.server_logger.info("server direct exit")

                    os._exit(11)
                else:
                    os.close(wf)
                    read_ev = threading.Event()
                    port_bytes = []

                    def read_port(buf):
                        port_bytes.append(os.read(rf, buf))  # 15000-65000
                        read_ev.set()

                    thread.start_new_thread(read_port, (64,))
                    read_ev.wait(5.0)
                    log.server_logger.info("read port finish from pipe")
                    if port_bytes and port_bytes[0]:
                        port = pickle.loads(port_bytes[0])
                        if check_port(port, task_file_name) == task_file_name:
                            env.meta_cache["servicePort"] = port
                            meta_file.dump()
                        else:
                            port = 0
        else:
            log.server_logger.error("the os unsupport fork,so server can'nt launch")
    except:
        log.server_logger.exception("launch server error")

    log.server_logger.info("service port:%d and pid:%d", port, pid)

    return port


class P2PServer(ThreadingTCPServer):
    """
    P2PServer offer file-block to other clients
    """
    daemon_threads = True
    request_queue_size = 16
    allow_reuse_address = True


class SimpleHttpRequestHandler(BaseHTTPRequestHandler):
    rbufsize = 1024 * 1024

    server_version = "SimpleHttp/" + constants.VERSION
    default_request_version = "HTTP/1.0"
    protocol_version = "HTTP/1.0"

    def do_get(self):
        self.do_GET()

    def do_GET(self):
        try:
            param = None
            method = None
            method, param = self._parse_method()
            if method and hasattr(self, method):
                method = getattr(self, method)
                method(param)
                return
        except:
            log.server_logger.exception("process error for method:%s and param:%s", method, param)
        self.send_error(500, "param:%s" % param)

    @staticmethod
    def __parse_range(params, need_pad):
        # Range:bytes=start-end
        try:
            range_param = params['range']
        except KeyError:
            return None, None, None, None
        range_str = range_param.split('=')[1]
        range_arr = range_str.split('-')
        start = int(range_arr[0])
        end = int(range_arr[1])
        piece_len = end - start + 1
        if need_pad:
            start -= params["pieceNum"] * 5
            end = start + (piece_len - 5) - 1
        return range_str, start, end, piece_len

    def send_upload_head(self, task_file_name, piece_len):
        task_path = core.get_service_file(task_file_name, sync_task_map.read(task_file_name, "dataDir"))
        ctype = 'application/octet-stream'
        f = fileutil.open_file(task_path, "rb", 4 * 1024 * 1024)
        if not f:
            log.server_logger.error("file:%s not found", task_path)
            self.send_error(404, "File Not Found")
        else:
            self.send_response(200)
            self.send_header("Content-type", ctype)
            self.send_header("Content-Length", str(piece_len))
            self.end_headers()
        return f

    def send_success(self):
        ctype = 'application/octet-stream'
        self.send_response(200)
        self.send_header("Content-type", ctype)
        self.end_headers()
        return

    def _parse_method(self):
        index = self.path.rfind("/")
        param = {}
        method = None
        if index != -1:
            method_pattern = self.path[:index + 1]

            if method_pattern == constants.PEER_HTTP_PATH_PREFIX:
                method = "upload"
                param["taskFileName"] = self.path[index + 1:]
                if "pieceNum" in self.headers:
                    param["pieceNum"] = int(self.headers['pieceNum'])
                if "pieceSize" in self.headers:
                    param["pieceSize"] = int(self.headers['pieceSize'])
                param["range"] = self.headers['Range']
            else:
                param = json.loads(self.headers['param'])
                if method_pattern == constants.LOCAL_HTTP_PATH_RATE:
                    method = "parse_rate"
                    param["taskFileName"] = self.path[index + 1:]
                elif method_pattern == constants.LOCAL_HTTP_PATH_CHECK:
                    method = "check"
                    param["taskFileName"] = self.path[index + 1:]
                elif method_pattern == constants.LOCAL_HTTP_PATH_CLIENT:
                    if self.path[index + 1:] == "finish":
                        method = "one_finish"

        return method, param

    def upload(self, param):
        task_file_name = param["taskFileName"]
        need_pad = True
        f = None
        range_str, start, end, piece_len = self.__parse_range(param, need_pad)
        try:
            read_len = end - start + 1

            f = self.send_upload_head(task_file_name, piece_len)
            if f:
                alive_qu.put('.')
                if need_pad:
                    self.wfile.write(struct.pack("!i", read_len | (param["pieceSize"] << 4)))

                total = 0
                f.seek(start, 0)
                diff = read_len - total
                buf_size = 256 * 1024
                while diff > 0:
                    if diff < buf_size:
                        cont = f.read(diff)
                    else:
                        cont = f.read(buf_size)
                    if cont:
                        if rate_limiter:
                            rate_limiter.acquire(len(cont))
                        self.wfile.write(cont)
                    else:
                        if total == 0:
                            log.server_logger.error("range:%s content is empty", range_str)
                        break
                    total += len(cont)
                    diff = read_len - total
                if need_pad:
                    self.wfile.write(struct.pack("!b", 0x7f))
        except:
            log.server_logger.exception("send range:%s error", range_str)
        finally:
            try:
                if f:
                    f.close()
            except:
                pass
        return

    def parse_rate(self, params):
        alive_qu.put('.')
        self.send_success()
        task_file_name = params["taskFileName"]
        sync_task_map.update(task_file_name, {"rateLimit": params["rateLimit"]})
        self.wfile.write(sync_task_map.parse_rate(task_file_name).encode("UTF-8"))

    def check(self, param):
        alive_qu.put('.')
        checker_lock.acquire()
        try:
            global rate_limiter
            self.send_success()
            if 'totalLimit' in param and param["totalLimit"] > 0:
                if not rate_limiter:
                    rate_limiter = ratelimiter.RateLimiter(param["totalLimit"])
                else:
                    rate_limiter.refresh(param["totalLimit"])

                SyncTaskMap.update_total_limit(param["totalLimit"])
                log.server_logger.info("update total limit to %d", param["totalLimit"])
            task_file_name = param["taskFileName"]
            sync_task_map.update(task_file_name, {"taskFileName": task_file_name,
                                                  "dataDir": param["dataDir"],
                                                  "rateLimit": 0, "finished": False})
            self.wfile.write((task_file_name + "@" + constants.VERSION).encode("UTF-8"))
        finally:
            checker_lock.release()

    def one_finish(self, param):
        def finish_client(params):
            task_file_name = params["taskFileName"]
            sync_task_map.update(task_file_name, {"taskId": params["taskId"], "cid": params["cid"],
                                                  "superNode": params["superNode"], "rateLimit": 0, "finished": True})

        self.send_success()
        thread.start_new_thread(finish_client, (param,))
        self.wfile.write("success".encode("UTF-8"))


class SyncTaskMap(object):
    total_limit = None

    def __init__(self):
        self.map_data = {}  # {first_key:taskFileName}
        self.map_lock = threading.RLock()

    def lock(self):
        self.map_lock.acquire()

    def unlock(self):
        self.map_lock.release()

    def read(self, first_key, second_key):
        self.lock()
        try:
            if first_key and first_key in self.map_data:
                task_data = self.map_data[first_key]
                if second_key and task_data and second_key in task_data:
                    return task_data[second_key]
        finally:
            self.unlock()
        return None

    def has(self, first_key, second_key=None):
        self.lock()
        result = False
        try:
            if first_key and first_key in self.map_data:
                result = True
                if second_key:
                    result = False
                    task_data = self.map_data[first_key]
                    if task_data and second_key in task_data:
                        result = True
        finally:
            self.unlock()
        return result

    def update(self, first_key, data):
        self.lock()
        try:
            if first_key and data:
                if first_key in self.map_data:
                    self.map_data[first_key].update(data)
                else:
                    self.map_data[first_key] = data
        finally:
            self.unlock()

    def delete(self, first_key):
        self.lock()
        try:
            if first_key in self.map_data:
                del self.map_data[first_key]
        finally:
            self.unlock()

    def parse_rate(self, first_key):
        self.lock()
        try:
            client_rate = self.read(first_key, "rateLimit")
            if not client_rate:
                return ""
            if not self.total_limit:
                return str(client_rate)
            total = 0
            for first_key in self.map_data:
                tmp_rate = self.map_data[first_key]["rateLimit"]
                total += tmp_rate
            if total > self.total_limit:
                client_rate = (client_rate * self.total_limit + total - 1) // total
            return str(client_rate)
        finally:
            self.unlock()

    @staticmethod
    def update_total_limit(total_limit):
        SyncTaskMap.total_limit = total_limit


sync_task_map = SyncTaskMap()
