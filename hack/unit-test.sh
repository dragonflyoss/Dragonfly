#!/bin/bash
curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}" || return

. ./env.sh

unit_test() {
    echo "TEST: unit test"
    cd ../ || return
    go test -i ./...

    cmd="go list ./... | grep 'github.com/dragonflyoss/Dragonfly/'"
    sources="${GO_SOURCE_DIRECTORIES[*]}"
    sources="${sources// /|}"
    test -n "${sources}" && cmd+=" | grep -E '${sources}'"

    for d in $(eval "${cmd}")
    do
        go test -race -coverprofile=profile.out -covermode=atomic "${d}"
        if [ -f profile.out ] ; then
            cat profile.out >> coverage.txt
            rm profile.out > /dev/null 2>&1
        fi
    done
}

unit_test "$@"
