#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

# golangci-lint binary version v1.17.1

PKG=github.com/dragonflyoss/Dragonfly
LINT_IMAGE=pouchcontainer/pouchlinter:v0.2.3
USE_DOCKER=${USE_DOCKER:-"0"}

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return
LINT_SOURCE_HOME=$(cd ".." && pwd)

. ./env.sh

golangci-lint-docker() {
    cd "${LINT_SOURCE_HOME}" || return
    docker run                                                            \
        --rm                                                              \
        -ti                                                               \
        -v "$(pwd)"/.go/pkg:/go/pkg                                       \
        -v "$(pwd)":/go/src/${PKG}                                        \
        -e GO111MODULE=on                                                 \
        -e GOPROXY="${GOPROXY}"                                           \
        -w /go/src/${PKG}                                                 \
        ${LINT_IMAGE}                                                     \
        golangci-lint run
}

check-golangci-lint() {
  has_installed="$(command -v golangci-lint || echo false)"
  if [[ "${has_installed}" = "false" ]]; then
    echo false
    return
  fi
  echo true
}

golangci-lint-local() {
    has_installed="$(check-golangci-lint)"
    if [[ "${has_installed}" = "true" ]]; then
        echo "Detected that golangci-lint has already been installed. Start linting..."
        cd "${LINT_SOURCE_HOME}" || return
        golangci-lint run
        return
    else
        echo "Detected that golangci-lint has not been installed. You have to install golangci-lint first."
        return 1
    fi
}

main() {
    if [[ "1" == "${USE_DOCKER}" ]]
    then
        echo "Begin to check gocode with golangci-lint in docker."
        golangci-lint-docker
    else
        echo "Begin to check gocode with golangci-lint locally"
        golangci-lint-local
    fi
    echo "Check gocode with golangci-lint successfully "
}

main "$@"
