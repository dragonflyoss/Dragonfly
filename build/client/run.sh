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
# scripts' variables

export GOPATH=${BUILD_GOPATH}:${GOPATH}

BIN_DIR=${BUILD_GOPATH}/bin
PKG_DIR=${BUILD_GOPATH}/package

DFDAEMON_BINARY_NAME=dfdaemon
DFGET_BINARY_NAME=dfget-go

PKG_NAME=df-client

#
# =============================================================================
# build commands

pre() {
    echo "PRE: clean and create ${BIN_DIR}"
    createDir ${BIN_DIR}
}

check() {
    cd ${BUILD_SOURCE_HOME}
    exclude="vendor/"

    # gofmt
    echo "CHECK: gofmt, check code formats"
    result=`find . -name '*.go' | grep -vE "${exclude}" | xargs gofmt -s -l 2>/dev/null`
    [ ${#result} -gt 0 ] && (echo "${result}" \
        && echo "CHECK: please format Go code with 'gofmt -s -w .'" && false)

    # golint
    which golint > /dev/null || export PATH=${BUILD_GOPATH}:$PATH
    which golint > /dev/null || (echo "CHECK: install golint" \
        && go get -u golang.org/x/lint/golint; \
            cp ${BUILD_GOPATH}/bin/golint ${BUILD_GOPATH}/)

    echo "CHECK: golint, check code style"
    result=`go list ./... | grep -vE "${exclude}" | sed 's/^_//' | xargs golint`
    [ ${#result} -gt 0 ] && (echo "${result}" && false)

    # go vet check
    echo "CHECK: go vet, check code syntax"
    packages=`go list ./... | grep -vE "${exclude}" | sed 's/^_//'`
    go vet ${packages} 2>&1
}

dfdaemon() {
    echo "BUILD: dfdaemon"
    test -f ${BIN_DIR}/${DFDAEMON_BINARY_NAME} && rm -f ${BIN_DIR}/${DFDAEMON_BINARY_NAME}
    cd ${BUILD_SOURCE_HOME}/cmd/dfdaemon
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

dfget-go() {
    echo "BUILD: dfget-go"
    test -f ${BIN_DIR}/${DFGET_BINARY_NAME} && rm -f ${BIN_DIR}/${DFGET_BINARY_NAME}
    cd ${BUILD_SOURCE_HOME}/cmd/dfget
    go build -o ${BIN_DIR}/${DFGET_BINARY_NAME}
    chmod a+x ${BIN_DIR}/${DFGET_BINARY_NAME}
}

unit-test() {
    echo "TEST: unit test"
    cd ${BUILD_SOURCE_HOME}
    go test -i ./...

    cmd="go list ./... | grep 'github.com/alibaba/Dragonfly/'"
    sources=`echo ${GO_SOURCE_DIRECTORIES[@]} | sed 's/ /|/g'`
    test -n "${sources}" && cmd+=" | grep -E '${sources}'"

    for d in $(eval ${cmd})
    do
        go test -race -coverprofile=profile.out -covermode=atomic ${d}
        if [ -f profile.out ] ; then
            cat profile.out >> coverage.txt
            rm profile.out > /dev/null 2>&1
        fi
    done
}

package() {
    createDir ${PKG_DIR}/${PKG_NAME}
    cp -r ${BIN_DIR}/${PKG_NAME}/*          ${PKG_DIR}/${PKG_NAME}/
    cp ${BIN_DIR}/${DFDAEMON_BINARY_NAME}   ${PKG_DIR}/${PKG_NAME}/
    cp ${BIN_DIR}/${DFGET_BINARY_NAME}      ${PKG_DIR}/${PKG_NAME}/

    cd ${PKG_DIR} && tar czf ${INSTALL_HOME}/${PKG_NAME}.tar.gz ./${PKG_NAME}
    rm -rf ${PKG_DIR}
}

install() {
    installDir=${INSTALL_HOME}/${PKG_NAME}
    echo "INSTALL: ${installDir}"
    createDir ${installDir}
    # cp -r ${BIN_DIR}/${PKG_NAME}/*          ${installDir}
    cp ${BIN_DIR}/${DFDAEMON_BINARY_NAME}   ${installDir}
    cp ${BIN_DIR}/${DFGET_BINARY_NAME}      ${installDir}
}

uninstall() {
    echo "uninstall dragonfly: ${INSTALL_HOME}"
    test -d ${INSTALL_HOME} && rm -rf ${INSTALL_HOME}
}

clean() {
    echo "delete ${BUILD_GOPATH}"
    test -d ${BUILD_GOPATH} && rm -rf ${BUILD_GOPATH}
}

#
# =============================================================================
#

createDir() {
    test -e $1 && rm -rf $1
    mkdir -p $1
}

COMMANDS="pre|check|dfdaemon|dfget|dfget-go|unit-test|package|install|uninstall|clean"
usage() {
    echo "Usage: $0 [${COMMANDS}]"
    exit 1
}


main() {
    cmd="${COMMANDS}"
    action=`echo ${cmd} | grep -w "$1"`
    test -z $1 && usage
    $1
}

main "$@"
