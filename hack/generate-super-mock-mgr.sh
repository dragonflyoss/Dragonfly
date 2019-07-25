#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

# This script is used to generate mock files for interfaces.

# Get the absolute path of this file
DIR="$( cd "$( dirname "$0"  )" && pwd  )"/..
cd "$DIR"

# install mockgen if it not exists
mockgen -h >/dev/null 2>&1 || go get github.com/golang/mock/mockgen
# install goimports if it not exists
goimports -h >/dev/null 2>&1 || go get golang.org/x/tools/cmd/goimports


# generate mock files for supernode mgr interfaces.
MGR_ARRAY=("cdn_mgr" "dfget_task_mgr" "peer_mgr" "progress_mgr" "scheduler_mgr")
for name in "${MGR_ARRAY[@]}"
do
	mockgen -destination "./supernode/daemon/mgr/mock/mock_$name.go" -source "supernode/daemon/mgr/$name.go" -package mock
	goimports -w --local "github.com/dragonflyoss/Dragonfly" "./supernode/daemon/mgr/mock"
	echo "Mock file ./supernode/daemon/mgr/mock/mock_$name.go generated successfully for $name"
done


