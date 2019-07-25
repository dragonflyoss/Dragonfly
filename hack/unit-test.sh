#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return

. ./env.sh

unit_test() {
    echo "TEST: unit test"
    cd ../ || return
    go test -i ./...

    # folder /test contains test cases for integration test.
    # then we exclude them in unit test.
    cmd="go list ./... | grep 'github.com/dragonflyoss/Dragonfly/'"
    excludes="${GO_SOURCE_EXCLUDES[*]}"
    excludes="${excludes// /|}"
    retCode=0
    test -n "${excludes}" && cmd+=" | grep -vE '${excludes}'"

    for d in $(eval "${cmd}")
    do
        go test -race -coverprofile=profile.out -covermode=atomic "${d}"
        if [ "$?" != "0" ]; then
            retCode=1
        fi
        if [ -f profile.out ] ; then
            cat profile.out >> coverage.txt
            rm profile.out > /dev/null 2>&1
        fi
    done
    return ${retCode}
}

unit_test "$@"
