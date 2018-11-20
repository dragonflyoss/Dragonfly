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

#
# env.sh must be executed in its parent directory
#
curDir=$(pwd)

export DRAGONFLY_HOME=${curDir%/build/client*}
export BUILD_GOPATH=/tmp/dragonfly/build
export BUILD_SOURCE_HOME="${BUILD_GOPATH}/src/github.com/dragonflyoss/Dragonfly"

export INSTALL_HOME="/opt/dragonfly"

export CONFIGURED_VARIABLES_FILE="${BUILD_GOPATH}/configured_variables.sh"

#
# source directories
#
export GO_SOURCE_DIRECTORIES=( \
    "dfdaemon" \
    "dfget" \
    "version" \
)

