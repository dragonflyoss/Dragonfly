#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

if ! git diff --name-only "$COMMIT_RANGE" | grep -qvE '(\.md)||^(docs/)||^(CONTRIBUTORS)|^(LICENSE)'; then
  echo "Only doc files were updated, not running the CI."
  circleci-agent step halt
fi