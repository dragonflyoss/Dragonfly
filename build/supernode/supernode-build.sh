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

curDir=`cd $(dirname $0) && pwd`
SUPERNODE_SOURCE_HOME="${curDir}/../../src/supernode"

ACTION=$1

. ${curDir}/../log.sh

usage() {
    echo "Usage: $0 {source|image|all}"
    echo "  source      compile supernode's source"
    echo "  image       build docker image of supernode"
    echo "  all         compile source and build image"
    exit 2; # bad usage
}

if [ $# -lt 1 ]; then
    usage
fi

compileSupernode() {
    echo "====================================================================="
    info "supernode:source" "compiling source..."
    mvn clean package
}

buildDockerImage() {
    echo "====================================================================="
    info "supernode:image" "building image..."
    mvn clean package -DskipTests docker:build
}

check() {
    which docker > /dev/null && docker ps > /dev/null 2>&1 \
        || (echo "Please install docker and start docker daemon first." && exit 3)
}

main() {
    cd ${SUPERNODE_SOURCE_HOME}
    case "${ACTION}" in
        source)
            compileSupernode
        ;;
        image)
            check && buildDockerImage
        ;;
        all)
            check && compileSupernode && buildDockerImage
        ;;
        *)
            usage
        ;;
    esac
}

main

