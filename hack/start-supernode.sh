#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail

nginx

/opt/dragonfly/df-supernode/supernode "$@"
