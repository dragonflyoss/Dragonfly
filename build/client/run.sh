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

set -e

curDir=`cd $(dirname $0) && pwd`
cd ${curDir}

#
# init build environment variables
#
. ./env.sh

#
# init configured variables
#
test -e ${CONFIGURED_VARIABLES_FILE} || (echo "ERROR: must execute './configure' before '$0'" && exit 2)
. ${CONFIGURED_VARIABLES_FILE}

#
# =============================================================================
#

BIN_DIR=${BUILD_GOPATH}/bin
PKG_DIR=${BUILD_GOPATH}/package

DFDAEMON_BINARY_NAME=dfdaemon

PKG_NAME=df-client

pre() {
    echo "PRE: clean and create ${BIN_DIR}"
    createDir ${BIN_DIR}
}

dfdaemon() {
    echo "BUILD: dfdaemon"
    test -f ${BIN_DIR}/${DFDAEMON_BINARY_NAME} && rm -f ${BIN_DIR}/${DFDAEMON_BINARY_NAME}
    export GOPATH=${BUILD_GOPATH}
    cd ${BUILD_SOURCE_HOME}/dfdaemon
    go build -o ${BIN_DIR}/${DFDAEMON_BINARY_NAME}
    chmod a+x ${BIN_DIR}/${DFDAEMON_BINARY_NAME}
}

dfget() {
    echo "BUILD: dfget"
    dfgetDir=${BIN_DIR}/${PKG_NAME}
    createDir ${dfgetDir}
    cp -r ${BUILD_SOURCE_HOME}/src/getter/* ${dfgetDir}
    find ${dfgetDir} -name '*.pyc' | xargs rm -f
    chmod a+x ${dfgetDir}/dfget
}

package() {
    createDir ${PKG_DIR}/${PKG_NAME}
    cp -r ${BIN_DIR}/${PKG_NAME}/*          ${PKG_DIR}/${PKG_NAME}/
    cp ${BIN_DIR}/${DFDAEMON_BINARY_NAME}   ${PKG_DIR}/${PKG_NAME}/

    cd ${PKG_DIR} && tar czf ${INSTALL_HOME}/${PKG_NAME}.tar.gz ./${PKG_NAME}
    rm -rf ${PKG_DIR}
}

install() {
    installDir=${INSTALL_HOME}/${PKG_NAME}
    echo "INSTALL: ${installDir}"
    createDir ${installDir}
    cp -r ${BIN_DIR}/${PKG_NAME}/*          ${installDir}
    cp ${BIN_DIR}/${DFDAEMON_BINARY_NAME}   ${installDir}
}

uninstall() {
    echo "uninstall dragonfly: ${INSTALL_HOME}"
    test -d ${INSTALL_HOME} && rm -rf ${INSTALL_HOME}
}

clean() {
    test -d ${BUILD_GOPATH} && rm -rf ${BUILD_GOPATH}
}

#
# =============================================================================
#

createDir() {
    test -e $1 && rm -rf $1
    mkdir -p $1
}

usage() {
    echo "Usage: $0 [pre|daemon|dfget|package|install|uninstall]"
    exit 1
}


main() {
    cmd="pre dfdaemon dfget package install uninstall clean"
    action=`echo ${cmd} | grep -w "$1"`
    test -z $1 && usage
    $1
}

main "$@"
