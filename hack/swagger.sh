#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

# Get the absolute path of this file
DIR="$( cd "$( dirname "$0"  )" && pwd  )"
cd "${DIR}" || return

# Here is details of the swagger binary we use.
# $ swagger version
# version: 0.19.0
SWAGGER_VERSION=v0.19.0

. ./env.sh

BUILD_PATH=../bin/${GOOS}_${GOARCH}

SWAGGER_BIN="swagger"

# swagger::check_version checks the command and the version.
swagger::check_version() {
  local has_installed version

  has_installed="$(command -v swagger || echo false)"
  if [[ "${has_installed}" = "false" ]]; then
    echo false
    return
  fi

  version="$(swagger version | head -n 1 | cut -d " " -f 2)"

  if [ "${SWAGGER_VERSION}" == "${version}" ]
  then
    echo true
  else
    echo false
  fi
}

# swagger::install installs the swagger binary.
swagger::install() {
    has_installed="$(swagger::check_version)"
    if [[ "${has_installed}" == "true" ]]; then
        echo ">>>> Detected that swagger-${SWAGGER_VERSION} has already installed. Skip installation. <<<<"
        return
    fi

    echo ">>>> Detected that swagger-${SWAGGER_VERSION} hasn't already installed and start to install it. <<<<"
    local url
    url="https://github.com/go-swagger/go-swagger/releases/download/${SWAGGER_VERSION}/swagger_${GOOS}_${GOARCH}"
    test -d "${BUILD_PATH}" || mkdir -p "${BUILD_PATH}"
    wget -O "${BUILD_PATH}"/swagger "${url}"
    chmod +x "${BUILD_PATH}"/swagger
    SWAGGER_BIN="${BUILD_PATH}/swagger"
}

# swagger::generate generate the code by go-swagger.
swagger::generate() {
    "${SWAGGER_BIN}" generate model -f "$DIR/../apis/swagger.yml" -t "$DIR/../apis" -m types
}

main() {
    swagger::install
    swagger::generate
}

main "$@"
