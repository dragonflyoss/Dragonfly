#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

BIN_DIR="../bin"
DFDAEMON_BINARY_NAME=dfdaemon
DFGET_BINARY_NAME=dfget
SUPERNODE_BINARY_NAME=supernode

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return

. ./env.sh

install() {
    case "${1-}" in
        dfclient)
            install-client
        ;;
        supernode)
            install-supernode
        ;;
        *)
            install-client
            install-supernode
        ;;
    esac
}

install-client(){
    local installClientDir="${INSTALL_HOME}/${INSTALL_CLIENT_PATH}"
    echo "install: ${installClientDir}"
    createDir "${installClientDir}"

    cp "${BIN_DIR}/${GOOS}_${GOARCH}/${DFDAEMON_BINARY_NAME}"   "${installClientDir}"
    cp "${BIN_DIR}/${GOOS}_${GOARCH}/${DFGET_BINARY_NAME}"      "${installClientDir}"

    createLink "${installClientDir}/${DFDAEMON_BINARY_NAME}" /usr/local/bin/dfdaemon
    createLink "${installClientDir}/${DFGET_BINARY_NAME}" /usr/local/bin/dfget
}

install-supernode(){
    local installSuperDir="${INSTALL_HOME}/${INSTALL_SUPERNODE_PATH}"
    echo "install: ${installSuperDir}"
    createDir "${installSuperDir}"

    cp "${BIN_DIR}/${GOOS}_${GOARCH}/${SUPERNODE_BINARY_NAME}"  "${installSuperDir}"

    createLink "${installSuperDir}/${SUPERNODE_BINARY_NAME}" /usr/local/bin/supernode
}


uninstall() {
    case "${1-}" in
        dfclient)
            uninstall-client
        ;;
        supernode)
            uninstall-supernode
        ;;
        *)
            uninstall-all
        ;;
    esac
}

uninstall-client() {
    echo "unlink /usr/local/bin/dfdaemon"
    test -e /usr/local/bin/dfdaemon && unlink /usr/local/bin/dfdaemon
    echo "unlink /usr/local/bin/dfget"
    test -e /usr/local/bin/dfget && unlink /usr/local/bin/dfget
}

uninstall-supernode() {
    echo "unlink /usr/local/bin/supernode"
    test -e /usr/local/bin/supernode && unlink /usr/local/bin/supernode
}

uninstall-all(){
    uninstall-client
    uninstall-supernode

    echo "uninstall dragonfly: ${INSTALL_HOME}"
    test -d "${INSTALL_HOME}" && rm -rf "${INSTALL_HOME}"
}

createLink() {
    srcPath="$1"
    linkPath="$2"

    echo "create link ${linkPath} to ${srcPath}"
    test -e "${linkPath}" && unlink "${linkPath}"
    ln -s "${srcPath}" "${linkPath}"
}

createDir() {
    test -e "$1" && rm -rf "$1"
    mkdir -p "$1"
}

main() {
    case "${1-}" in
        install)
            install "${2-}"
        ;;
        uninstall)
            uninstall "${2-}"
        ;;
        *)
            echo "You must specify the subcommand 'install' or 'uninstall'."
        ;;
    esac
}

main "$@"
