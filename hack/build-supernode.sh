#!/bin/bash
SUPERNODE_BINARY_NAME="supernode.jar"
curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return
SUPERNODE_SOURCE_HOME="${curDir}/../src/supernode"
SUPERNODE_BIN="${SUPERNODE_SOURCE_HOME}/target"
BUILD_SOURCE_HOME=$(cd ".." && pwd)
BIN_DIR="${BUILD_SOURCE_HOME}/bin"
USE_DOCKER=${USE_DOCKER:-"0"}

build-supernode-docker() {
    cd "${SUPERNODE_SOURCE_HOME}" || return
    mvn clean package -DskipTests docker:build -e
    cp "${SUPERNODE_BIN}/supernode.jar"  "${BIN_DIR}/${SUPERNODE_BINARY_NAME}"
}

build-supernode-local(){
   cd "${SUPERNODE_SOURCE_HOME}" || return
   mvn clean package -DskipTests
   cp "${SUPERNODE_BIN}/supernode.jar"  "${BIN_DIR}/${SUPERNODE_BINARY_NAME}"
}

create-dirs() {
    test -e "$1"
    mkdir -p "$1"
}

main() {
    create-dirs "${BIN_DIR}"
    case "${USE_DOCKER}" in
        "1")
            echo "Begin to build supernode with docker."
            build-supernode-docker
        ;;
        *)
            echo "Begin to build supernode in the local environment."
            build-supernode-local
        ;;
    esac
    echo "BUILD: supernode in ${BIN_DIR}/${SUPERNODE_BINARY_NAME}"
}

main "$@"
