#!/usr/bin/env bash

# Copyright 1999-2018 Alibaba Group.
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

# color: normal, red, yellow, green
CN="\033[0;;m"
CR="\033[1;31;m"
CY="\033[1;33;m"
CG="\033[1;32;m"

log() {
    filed=$1
    msg=$2
    echo -e "${CY}BUILD(${filed})${CN}: ${msg}"
}

info() {
    log "$1" "${CG}$2${CN}"
}

error() {
    log "$1" "${CR}$2${CN}"
}

