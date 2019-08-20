#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}/../" || return

check() {
    result=$(git ls-files | xargs go run ./hack/boilerplate/check-boilerplate.go 2>&1)
    if [[ ${#result} -gt 0 ]]; then
        echo "${result}"
        return 1
    fi
}

check
