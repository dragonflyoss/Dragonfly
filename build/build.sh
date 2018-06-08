#!/usr/bin/env bash

curDir=`cd $(dirname $0) && pwd`
BASE_HOME=`cd ${curDir}/.. && pwd`
SOURCE_DIR="${BASE_HOME}/src"
BUILD_DIR="${BASE_HOME}/build"

INSTALL_PREFIX="${HOME}/.dragonfly"

. ${BUILD_DIR}/log.sh

printResult() {
    filed=$1
    result=$2
    if [ ${result} -eq 0 ]; then
        info "${filed}" "SUCCESS"
    else
        error "${filed}" "FAILURE"
    fi
}
buildClient() {
    field="dfdaemon&dfget"
    echo "====================================================================="
    info "${field}" "compiling dfdaemon and dfget..."
    cd ${BUILD_DIR}/client
    test -d ${INSTALL_PREFIX} || mkdir -p ${INSTALL_PREFIX}
    ./configure --prefix=${INSTALL_PREFIX}
    make && make install && export PATH=$PATH:${INSTALL_PREFIX}/df-client
    retCode=$?
    make clean
    printResult "${field}" ${retCode}
    return ${retCode}
}

buildSupernode() {
    field="supernode"
    echo "====================================================================="
    info "${field}" "compiling source and building image..."
    ${BUILD_DIR}/supernode/supernode-build.sh all
    retCode=$?
    printResult "${field}" ${retCode}
    return ${retCode}
}

main() {
    case "$1" in
        client)
            buildClient
        ;;
        supernode)
            buildSupernode
        ;;
        *)
            buildSupernode && buildClient
        ;;
    esac
}

main "$@"

