#!/usr/bin/env bash

# color: normal, red, yellow, green
CN="\033[0;;m"
CR="\033[1;31;m"
CY="\033[1;33;m"
CG="\033[1;32;m"

log() {
    filed=$1
    msg=$2
    echo -e "${CY}BUILD(${filed})${CN}: ${msg}"
}

info() {
    log "$1" "${CG}$2${CN}"
}

error() {
    log "$1" "${CR}$2${CN}"
}

