#!/bin/bash
DFDAEMON_BINARY_NAME=dfdaemon
DFGET_BINARY_NAME=dfget

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return
BUILD_SOURCE_HOME=$(cd ".." && pwd)
BIN_DIR="${BUILD_SOURCE_HOME}/bin"

. ./env.sh

build-dfdaemon-local() {
    echo "BUILD: dfdaemon in ${BIN_DIR}/${DFDAEMON_BINARY_NAME}"
    test -f "${BIN_DIR}/${DFDAEMON_BINARY_NAME}" && rm -f "${BIN_DIR}/${DFDAEMON_BINARY_NAME}"
    cd "${BUILD_SOURCE_HOME}/cmd/dfdaemon" || return
    go build -o "${BIN_DIR}/${DFDAEMON_BINARY_NAME}" || return
    chmod a+x "${BIN_DIR}/${DFDAEMON_BINARY_NAME}"
}

build-dfget-local() {
    echo "BUILD: dfget in ${BIN_DIR}/${DFGET_BINARY_NAME}"
    test -f "${BIN_DIR}/${DFGET_BINARY_NAME}" && rm -f "${BIN_DIR}/${DFGET_BINARY_NAME}"
    cd "${BUILD_SOURCE_HOME}/cmd/dfget"	|| return
    go build -o "${BIN_DIR}/${DFGET_BINARY_NAME}" || return
    chmod a+x "${BIN_DIR}/${DFGET_BINARY_NAME}"
}

main() {
    case "$1" in
        build-dfdaemon-local)
            build-dfdaemon-local
        ;;
        build-dfget-local)
            build-dfget-local
        ;;
        *)
            build-dfget-local && build-dfdaemon-local
        ;;
    esac
}

main "$@"
