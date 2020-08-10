#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

DFDAEMON_BINARY_NAME=dfdaemon
DFGET_BINARY_NAME=dfget
SUPERNODE_BINARY_NAME=supernode
PKG=github.com/dragonflyoss/Dragonfly
BUILD_IMAGE=golang:1.13.15
VERSION=$(git describe --tags "$(git rev-list --tags --max-count=1)")
REVISION=$(git rev-parse --short HEAD)
DATE=$(date "+%Y%m%d-%H:%M:%S")
LDFLAGS="-X ${PKG}/version.version=${VERSION:1} -X ${PKG}/version.revision=${REVISION} -X ${PKG}/version.buildDate=${DATE}"

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return
BUILD_SOURCE_HOME=$(cd ".." && pwd)

. ./env.sh

BUILD_PATH=bin/${GOOS}_${GOARCH}
USE_DOCKER=${USE_DOCKER:-"0"}

create-dirs() {
    cd "${BUILD_SOURCE_HOME}" || return
    mkdir -p .go/src/${PKG} .go/bin .cache
    mkdir -p "${BUILD_PATH}"
}

build-local() {
    test -f "${BUILD_SOURCE_HOME}/${BUILD_PATH}/$1" && rm -f "${BUILD_SOURCE_HOME}/${BUILD_PATH}/$1"
    cd "${BUILD_SOURCE_HOME}/cmd/$2" || return
    go build -o "${BUILD_SOURCE_HOME}/${BUILD_PATH}/$1" -ldflags "${LDFLAGS}"
    chmod a+x "${BUILD_SOURCE_HOME}/${BUILD_PATH}/$1"
    echo "BUILD: $2 in ${BUILD_SOURCE_HOME}/${BUILD_PATH}/$1"
}

build-dfdaemon-local(){
    build-local ${DFDAEMON_BINARY_NAME} dfdaemon
}

build-dfget-local() {
    build-local ${DFGET_BINARY_NAME} dfget
}

build-supernode-local() {
    build-local ${SUPERNODE_BINARY_NAME} supernode
}

build-docker() {
    cd "${BUILD_SOURCE_HOME}" || return
    docker run                                                            \
        --rm                                                              \
        -ti                                                               \
        -u "$(id -u)":"$(id -g)"                                          \
        -v "$(pwd)"/.go:/go                                               \
        -v "$(pwd)":/go/src/${PKG}                                        \
        -v "$(pwd)"/"${BUILD_PATH}":/go/bin                               \
        -v "$(pwd)"/.cache:/.cache                                        \
        -e GOOS="${GOOS}"                                                 \
        -e GOARCH="${GOARCH}"                                             \
        -e CGO_ENABLED=0                                                  \
        -e GO111MODULE=on                                                 \
        -e GOPROXY="${GOPROXY}"                                           \
        -w /go/src/${PKG}                                                 \
        ${BUILD_IMAGE}                                                    \
        go build -o "/go/bin/$1" -ldflags "${LDFLAGS}" ./cmd/"$2" 
    echo "BUILD: $1 in ${BUILD_SOURCE_HOME}/${BUILD_PATH}/$1"
}

build-dfdaemon-docker(){
    build-docker ${DFDAEMON_BINARY_NAME} dfdaemon
}

build-dfget-docker() {
    build-docker ${DFGET_BINARY_NAME} dfget
}

build-supernode-docker() {
    build-docker ${SUPERNODE_BINARY_NAME} supernode
}

main() {
    create-dirs
    if [[ "1" == "${USE_DOCKER}" ]]
    then
        echo "Begin to build with docker."
        case "${1-}" in
            dfdaemon)
                build-dfdaemon-docker
            ;;
            dfget)
                build-dfget-docker
            ;;
            supernode)
                build-supernode-docker
            ;;
            *)
                build-dfget-docker
                build-dfdaemon-docker
                build-supernode-docker
            ;;
        esac
    else
        echo "Begin to build in the local environment."
        case "${1-}" in
            dfdaemon)
                build-dfdaemon-local
            ;;
            dfget)
                build-dfget-local
            ;;
            supernode)
                build-supernode-local
            ;;
            *)
                build-dfget-local
                build-dfdaemon-local
                build-supernode-local
            ;;
        esac
    fi
}

main "$@"
