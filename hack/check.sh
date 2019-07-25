#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}/../" || return

check() {
    # gofmt
    echo "CHECK: gofmt, check code formats"
    result=$(find . -name '*.go' -print0 | xargs -0 gofmt -s -l -d 2>/dev/null)
    if [[ ${#result} -gt 0 ]]; then
        echo "${result}"
        echo "CHECK: please format Go code with 'gofmt -s -w .'"
        return 1
    fi

    # golint
    which golint > /dev/null || (echo "CHECK: install golint" \
        && go get -u golang.org/x/lint/golint )

    echo "CHECK: golint, check code style"
    result=$(go list ./... | sed 's/^_//' | xargs golint)
    if [[ ${#result} -gt 0 ]]; then
        echo "${result}"
        return 1
    fi

    # go vet check
    echo "CHECK: go vet, check code syntax"
    packages=$(go list ./... | sed 's/^_//')
    echo "${packages}" | xargs go vet 2>&1
}

check
