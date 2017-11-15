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
import sys
import time

import env
import paramparser


class StdShower(object):
    _value = 0
    _progress_len = 0
    _start_time = 0

    @staticmethod
    def reset():
        StdShower._value = 0

    @staticmethod
    def update(increment=0):

        if paramparser.cmdparam.showbar:
            if increment <= 0:
                return
            StdShower._value += increment
            if StdShower._start_time == 0:
                StdShower._start_time = time.time()
                print("====================start====================")

            progress = "progress[%d/%d   in %.3fs]" % (
                StdShower._value, env.file_length,
                time.time() - StdShower._start_time)

            if len(progress) < StdShower._progress_len:
                progress += " " * (StdShower._progress_len - len(progress))
            else:
                StdShower._progress_len = len(progress)
            sys.stdout.write(progress + "\r")
            sys.stdout.flush()

    @staticmethod
    def finish():
        if paramparser.cmdparam.showbar:
            print("\n=====================end=====================")

    @staticmethod
    def print_info(msg):
        if StdShower._progress_len and paramparser.cmdparam.showbar:
            sys.stdout.write(" " * StdShower._progress_len + "\r")
            StdShower._progress_len = 0
        print(msg)
