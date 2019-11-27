#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return

. ./env.sh

generate-cli-docs(){
    BUILD_PATH=bin/${GOOS}_${GOARCH}
    CLI_DOCS_DIR=$(cd "../docs/cli_reference" && pwd)
    DFGET_BIN_PATH=../"${BUILD_PATH}"/dfget
    DFDAEMON_BIN_PATH=../"${BUILD_PATH}"/dfdaemon
    SUPERNODE_BIN_PATH=../"${BUILD_PATH}"/supernode

    ${DFGET_BIN_PATH} gen-doc -p "${CLI_DOCS_DIR}" || return
    ${DFDAEMON_BIN_PATH} gen-doc -p "${CLI_DOCS_DIR}" || return
    ${SUPERNODE_BIN_PATH} gen-doc -p "${CLI_DOCS_DIR}" || return
    echo "Generate: CLI docs in ${CLI_DOCS_DIR}" 
}

main () {
    case "${1-}" in
        cli)
            generate-cli-docs
        ;;
        *)
            generate-cli-docs
        ;;  
    esac
}

main "$@"
