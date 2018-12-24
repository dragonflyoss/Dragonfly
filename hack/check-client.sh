#!/bin/bash
curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}/../" || return

check() {
    exclude="vendor/"

    # gofmt
    echo "CHECK: gofmt, check code formats"
    result=$(find . -name '*.go' | grep -vE "${exclude}" | xargs gofmt -s -l -d 2>/dev/null)
    [ ${#result} -gt 0 ] && (echo "${result}" \
        && echo "CHECK: please format Go code with 'gofmt -s -w .'" && false)

    # golint
    which golint > /dev/null || (echo "CHECK: install golint" \
        && go get -u golang.org/x/lint/golint )

    echo "CHECK: golint, check code style"
    result=$(go list ./... | grep -vE "${exclude}" | sed 's/^_//' | xargs golint)
    [ ${#result} -gt 0 ] && (echo "${result}" && false)

    # go vet check
    echo "CHECK: go vet, check code syntax"
    packages=$(go list ./... | grep -vE "${exclude}" | sed 's/^_//')
    echo "${packages}" | xargs go vet 2>&1
}

check "$@"
