#!/bin/bash
set -ex

# Build fuzzers
go-fuzz-build -libfuzzer -o uploader-fuzzer.a ./dfget/core/uploader
clang-9 -fsanitize=fuzzer uploader-fuzzer.a -o uploader-fuzzer

go-fuzz-build -libfuzzer -o cdn-fuzzer.a ./supernode/daemon/mgr/cdn
clang-9 -fsanitize=fuzzer cdn-fuzzer.a -o cdn-fuzzer

# Run regression or upload to Fuzzit
wget -q -O fuzzit https://github.com/fuzzitdev/fuzzit/releases/download/v2.4.29/fuzzit_Linux_x86_64
chmod a+x fuzzit
./fuzzit create job --type "${1}" dragonfly/uploader uploader-fuzzer
./fuzzit create job --type "${1}" dragonfly/cdn cdn-fuzzer
