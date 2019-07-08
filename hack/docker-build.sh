#!/bin/bash

DF_VERSION=${DF_VERSION:-"latest"}
curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}/../" || return

docker-build::build-dfclient(){
    docker build -t dfclient:"${DF_VERSION}" -f Dockerfile .
}

docker-build::build-supernode(){
    docker build -t supernode:"${DF_VERSION}" -f Dockerfile.supernode .
}

main() {
    case "$1" in
        dfclient)
            docker-build::build-dfclient
        ;;
        supernode)
            docker-build::build-supernode
        ;;
        *)
            docker-build::build-dfclient
            docker-build::build-supernode
        ;;
    esac
}

main "$@"
