#!/bin/bash

# Here is details of the swagger binary we use.
# $ swagger version
# version: 0.19.0

# Get the absolute path of this file
DIR="$( cd "$( dirname "$0"  )" && pwd  )"

swagger generate model -f "$DIR/../apis/swagger.yml" -t "$DIR/../apis" -m types
