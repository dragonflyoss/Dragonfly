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
import argparse
import getpass
import os
import re
import sys
import time

import stdshower


def parse():
    parser = argparse.ArgumentParser(description="dragonfly is a file distribution system based p2p")

    parser.add_argument("--url", "-u", help="will download a file from this url")
    parser.add_argument("--output", "-O", "-o",
                        help="output path that not only contains the dir part but also name part")
    parser.add_argument("--md5", "-m", help="expected file md5")
    parser.add_argument("--callsystem",
                        help="system name that executes dfget,its format is company_department_appName")
    parser.add_argument("--notbs", action="store_true", help="not back source when p2p fail")
    parser.add_argument("--locallimit", "-s",
                        help="rate limit about a single download task,its format is 20M/m/K/k")
    parser.add_argument("--totallimit", help="rate limit about the whole host,its format is 20M/m/K/k")
    parser.add_argument("--identifier", "-i",
                        help="identify download task,it is available merely when md5 param not exist")
    parser.add_argument("--timeout", "--exceed", "-e", help="download timeout(second)", type=int)
    parser.add_argument("--filter", "-f",
                        help="filter some query params of url ,e.g. -f 'key&sign' will filter key and sign query param.in this way,different urls correspond one same download task that can use p2p mode")
    parser.add_argument("--showbar", "-b", action="store_true", help="show progress bar")
    parser.add_argument("--pattern", "-p", choices=["p2p", "cdn"],
                        help="download pattern,cdn pattern not support totallimit")
    parser.add_argument("--version", "-v", action="store_true", help="version")
    parser.add_argument("--node", "-n", help="specify nodes")
    parser.add_argument("--console", help="show log on console", action="store_true")
    parser.add_argument("--header", help="http header, e.g. --header=\"Accept: *\" --header=\"Host: abc\"",
                        action="append")
    parser.add_argument("--dfdaemon", action="store_true", help="caller is from df-daemon")

    return parser.parse_args()


cmdparam = parse()


def default_output():
    name_start = cmdparam.url.rfind("/")
    if name_start == -1 or name_start == len(cmdparam.url) - 1:
        return cmdparam.url[9:]  # skip http:// or https://
    else:
        return cmdparam.url[name_start + 1:]


if cmdparam.version:
    import constants

    print(constants.VERSION)
    sys.exit(0)

assert re.match(r"(https?|HTTPS?)://(.+?)(:(\d+))?(/.*$|\?.*$|$)",
                cmdparam.url if cmdparam.url else ""), "please specify the cmd param(--url or -u)"

if not cmdparam.output:
    cmdparam.output = default_output()
cmdparam.output = os.path.abspath(cmdparam.output)
assert not os.path.isdir(cmdparam.output), "output cannot be a dir but a file path"

stdshower.StdShower.print_info("--%s--  %s" % (time.strftime('%Y-%m-%d %H:%M:%S'), cmdparam.url))
try:
    stdshower.StdShower.print_info("current user[%s] output path[%s]" % (getpass.getuser(), cmdparam.output))
except:
    stdshower.StdShower.print_info("output path[%s]" % cmdparam.output)
