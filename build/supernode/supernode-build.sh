#!/usr/bin/env bash

curDir=`cd $(dirname $0) && pwd`
SUPERNODE_SOURCE_HOME="${curDir}/../../src/supernode"

ACTION=$1

. ${curDir}/../log.sh

usage() {
    echo "Usage: $0 {source|image|all}"
    echo "  source      compile supernode's source"
    echo "  image       build docker image of supernode"
    echo "  all         compile source and build image"
    exit 2; # bad usage
}

if [ $# -lt 1 ]; then
    usage
fi

compileSupernode() {
    echo "====================================================================="
    info "supernode:source" "compiling source..."
    mvn clean package cobertura:cobertura
}

buildDockerImage() {
    echo "====================================================================="
    info "supernode:image" "building image..."
    mvn clean package -DskipTests docker:build
}

main() {
    cd ${SUPERNODE_SOURCE_HOME}
    case "${ACTION}" in
        source)
            compileSupernode
        ;;
        image)
            buildDockerImage
        ;;
        all)
            compileSupernode && buildDockerImage
        ;;
        *)
            usage
        ;;
    esac
}

main

