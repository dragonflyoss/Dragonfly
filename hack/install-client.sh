#!/bin/bash
BIN_DIR="../release"
DFDAEMON_BINARY_NAME=dfdaemon
DFGET_BINARY_NAME=dfget

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return

. ./env.sh

install() {
    installDir="${INSTALL_HOME}/${INSTALL_CLIENT_PATH}"
    echo "install: ${installDir}"
    createDir "${installDir}"
    cp "${BIN_DIR}/${GOOS}_${GOARCH}/${DFDAEMON_BINARY_NAME}"   "${installDir}"
    cp "${BIN_DIR}/${GOOS}_${GOARCH}/${DFGET_BINARY_NAME}"      "${installDir}"

    createLink "${installDir}/${DFDAEMON_BINARY_NAME}" /usr/local/bin/dfdaemon
    createLink "${installDir}/${DFGET_BINARY_NAME}" /usr/local/bin/dfget
}

uninstall() {
    echo "unlink /usr/local/bin/dfdaemon"
    test -e /usr/local/bin/dfdaemon && unlink /usr/local/bin/dfdaemon
    echo "unlink /usr/local/bin/dfget"
    test -e /usr/local/bin/dfget && unlink /usr/local/bin/dfget

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
    case "$1" in
        install)
            install
        ;;
        uninstall)
            uninstall
        ;;
        *)
            echo "You must specify the subcommand 'install' or 'uninstall'."
        ;;
    esac
}

main "$@"
