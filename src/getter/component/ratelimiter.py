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
import threading
import time


class RateLimiter(object):
    def __init__(self, rate=0, window=0.002):
        """
        This is a RateLimiter

        :param rate: token produce rate per second
        :param window: token will be produced at per window
        """
        self.capacity = rate
        self.rate = int(rate * window)  # rate per window
        self.window = window
        self.__lock = threading.RLock()
        self.last = time.time()
        self.__bucket = 0  # current token count
        self.raw = rate

    def acquire(self, token=1, blocking=True):
        """
        :param token: token to acquire
        :param blocking: blocking or not when token is not satisfied
        :return: available token return
        """
        # zero represent not limit
        if self.capacity <= 0:
            return token
        self.__lock.acquire()
        try:
            if self.capacity < token:
                self.capacity = token

            now = time.time()
            diff = now - self.last
            new_tokens = int(diff / self.window) * self.rate

            cur_total = self.__bucket + new_tokens

            if cur_total > self.capacity:
                cur_total = self.capacity

            if cur_total >= token:
                self.__bucket = cur_total - token
                self.last = now
                return token
            elif blocking:
                RateLimiter.__blocker((token - cur_total + self.rate - 1) // self.rate * self.window)
                return self.acquire(token, blocking)
            else:
                return -1
        finally:
            self.__lock.release()

    def refresh(self, rate):
        if self.raw != rate:
            self.capacity = rate
            self.rate = int(rate * self.window)
            self.raw = rate

    @staticmethod
    def __blocker(sleep_time=None):
        if sleep_time:
            time.sleep(sleep_time)
