#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

# Extract commit range (or single commit)
COMMIT_RANGE=$(echo "${CIRCLE_COMPARE_URL}" | cut -d/ -f7)

# Fix single commit, unfortunately we don't always get a commit range from Circle CI
if [[ $COMMIT_RANGE != *"..."* ]]; then
  COMMIT_RANGE="${COMMIT_RANGE}...${COMMIT_RANGE}"
fi

if ! git diff --name-only "$COMMIT_RANGE" | grep -qvE '(\.md)||^(docs/)||^(CONTRIBUTORS)|^(LICENSE)'; then
  echo "Only doc files were updated, not running the CI."
  circleci-agent step halt
fi