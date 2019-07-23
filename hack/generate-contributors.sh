#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

# see also ".mailmap" for how email addresses and names are deduplicated

{
	cat <<-'EOH'
	# This file lists all contributors having contributed to dragonfly.
	# For how it is generated, see `hack/generate-contributors.sh`.
	EOH
	echo
	git log --format='%aN <%aE>' | LC_ALL=C.UTF-8 sort -uf
} > CONTRIBUTORS
