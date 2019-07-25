#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

ln -s /opt/dragonfly/df-client/dfget /usr/local/bin/dfget
ln -s /opt/dragonfly/df-client/dfdaemon /usr/local/bin/dfdaemon
