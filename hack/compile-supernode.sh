#!/bin/bash
curDir=$(cd "$(dirname "$0")" && pwd)
SUPERNODE_SOURCE_HOME="${curDir}/../src/supernode"

compileSupernode() {
    cd "${SUPERNODE_SOURCE_HOME}"|| return
    mvn clean package
}

compileSupernode "$@"
