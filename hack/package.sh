#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

USE_DOCKER=${USE_DOCKER:-"0"}
VERSION=${VERSION:-"0.0.$(date +%s)"}

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return
BUILD_SOURCE_HOME=$(cd ".." && pwd)

. ./env.sh

BUILD_PATH=bin/${GOOS}_${GOARCH}
DFDAEMON_BINARY_NAME=dfdaemon
DFGET_BINARY_NAME=dfget

main() {
    cd "${BUILD_SOURCE_HOME}" || return
    # Maybe we should use a variable to set the directory for release,
    # however using a variable after `rm -rf` seems risky.
    mkdir -p release
    rm -rf release/*

    if [ "1" == "${USE_DOCKER}" ]
    then
        echo "Begin to package with docker."
        FPM="docker run --rm -it -v $(pwd):$(pwd) -w $(pwd) inoc603/fpm:alpine"
    else
        echo "Begin to package in local environment."
        FPM="fpm"
    fi

    case "${1-}" in
        rpm )
            build_rpm
            ;;
        deb )
            build_deb
            ;;
        * )
            build_rpm
            build_deb
            ;;
    esac
}

# TODO: Add description
DFCLIENT_DESCRIPTION="df-client"
# TODO: Add maintainer
MAINTAINER="dragonflyoss"

build_rpm() {
    ${FPM} -s dir -t rpm -f -p release --rpm-os=linux \
        --description "${DFCLIENT_DESCRIPTION}" \
        --maintainer "${MAINTAINER}" \
        --after-install ./hack/after-install.sh \
        --before-remove ./hack/before-remove.sh \
        -n df-client -v "${VERSION}" \
	"${BUILD_PATH}/${DFGET_BINARY_NAME}=${INSTALL_HOME}/${INSTALL_CLIENT_PATH}/${DFGET_BINARY_NAME}" \
	"${BUILD_PATH}/${DFDAEMON_BINARY_NAME}=${INSTALL_HOME}/${INSTALL_CLIENT_PATH}/${DFDAEMON_BINARY_NAME}"
}

build_deb() {
    ${FPM} -s dir -t deb -f -p release \
        --description "${DFCLIENT_DESCRIPTION}" \
        --maintainer "${MAINTAINER}" \
        --after-install ./hack/after-install.sh \
        --before-remove ./hack/before-remove.sh \
        -n df-client -v "${VERSION}" \
	"${BUILD_PATH}/${DFGET_BINARY_NAME}=${INSTALL_HOME}/${INSTALL_CLIENT_PATH}/${DFGET_BINARY_NAME}" \
	"${BUILD_PATH}/${DFDAEMON_BINARY_NAME}=${INSTALL_HOME}/${INSTALL_CLIENT_PATH}/${DFDAEMON_BINARY_NAME}"
}

main "$@"
