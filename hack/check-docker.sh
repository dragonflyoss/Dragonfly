#!/bin/bash
check() {
    which docker > /dev/null && docker ps > /dev/null 2>&1
    if [[ $? != 0 ]]; then
        echo "Please install docker and start docker daemon first." && exit 3
    fi
}

check "$@"
