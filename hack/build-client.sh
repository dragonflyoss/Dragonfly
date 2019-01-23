#!/bin/bash
DFDAEMON_BINARY_NAME=dfdaemon
DFGET_BINARY_NAME=dfget
PKG=github.com/dragonflyoss/Dragonfly
BUILD_IMAGE=golang:1.10.4
GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
BUILD=$(git rev-parse HEAD)
BUILD_PATH=bin/${GOOS}_${GOARCH}
USE_DOCKER=${USE_DOCKER:-"0"}

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return
BUILD_SOURCE_HOME=$(cd ".." && pwd)

. ./env.sh

create-dirs() {
    cd "${BUILD_SOURCE_HOME}" || return
    if [ "${GOOS}" == "darwin" ] && [ "1" == "${USE_DOCKER}" ]
    then
        BUILD_PATH=bin
    fi
    mkdir -p .go/src/${PKG} .go/bin .cache
    mkdir -p "${BUILD_PATH}"
}

build-dfdaemon-local() {	
    test -f "${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFDAEMON_BINARY_NAME}" && rm -f "${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFDAEMON_BINARY_NAME}"	
    cd "${BUILD_SOURCE_HOME}/cmd/dfdaemon" || return	
    go build -o "${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFDAEMON_BINARY_NAME}"	
    chmod a+x "${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFDAEMON_BINARY_NAME}"
    echo "BUILD: dfdaemon in ${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFDAEMON_BINARY_NAME}" 
}

build-dfget-local() {
    test -f "${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFGET_BINARY_NAME}" && rm -f "${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFGET_BINARY_NAME}"	
    cd "${BUILD_SOURCE_HOME}/cmd/dfget"	|| return
    go build -o "${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFGET_BINARY_NAME}"	
    chmod a+x "${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFGET_BINARY_NAME}"	
    echo "BUILD: dfget in ${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFGET_BINARY_NAME}"	
}

build-dfdaemon-docker() {
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
        -w /go/src/${PKG}                                                 \
        ${BUILD_IMAGE}                                                    \
        go install -v -pkgdir /go/pkg -ldflags "-X ${PKG}/cmd/dfdaemon/app.Build=${BUILD}" ./cmd/dfdaemon
    echo "BUILD: dfdaemon in ${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFDAEMON_BINARY_NAME}" 
}

build-dfget-docker() {
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
        -w /go/src/${PKG}                                                 \
        ${BUILD_IMAGE}                                                    \
        go install -v -pkgdir /go/pkg -ldflags "-X ${PKG}/cmd/dfget/app.Build=${BUILD}" ./cmd/dfget
echo "BUILD: dfget in ${BUILD_SOURCE_HOME}/${BUILD_PATH}/${DFGET_BINARY_NAME}"	
}

main() {
    create-dirs
    if [ "1" == "${USE_DOCKER}" ]
    then
        echo "Begin to build clients with docker."
        case "$1" in
            dfdaemon)
                build-dfdaemon-docker
            ;;
            dfget)
                build-dfget-docker
            ;;
            *)
                build-dfget-docker
                build-dfdaemon-docker
            ;;
        esac
    else
        echo "Begin to build clients in the local environment."
        case "$1" in
            dfdaemon)
                build-dfdaemon-local
            ;;
            dfget)
                build-dfget-local
            ;;
            *)
                build-dfget-local
                build-dfdaemon-local
            ;;
        esac
    fi
}

main "$@"
