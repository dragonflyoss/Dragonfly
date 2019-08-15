# Copyright The Dragonfly Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# You can assign 1 to USE_DOCKER so that you can build components of Dragonfly
# with docker.
USE_DOCKER?=0 # Default: build components in the local environment.

# Assign the Dragonfly version to DF_VERSION as the image tag.
DF_VERSION?=latest # Default: use latest as the image tag which built by docker.

# Default: in order to use go mod we have to export GO111MODULE=on.
export GO111MODULE=on

# Default: use GOPROXY to speed up downloading go mod dependency.
export GOPROXY=https://goproxy.io

clean:
	@echo "Begin to clean redundant files."
	@rm -rf ./bin
	@rm -rf ./release
.PHONY: clean

build-dirs:
	@mkdir -p ./bin
.PHONY: build-dirs

build: build-dirs
	@echo "Begin to build dfget and dfdaemon and supernode."
	./hack/build.sh
.PHONY: build

build-client: build-dirs
	@echo "Begin to build dfget and dfdaemon."
	./hack/build.sh dfget
	./hack/build.sh dfdaemon
.PHONY: build-client

build-supernode: build-dirs
	@echo "Begin to build supernode."
	./hack/build.sh supernode
.PHONY: build-supernode

install:
	@echo "Begin to install dfget and dfdaemon and supernode."
	./hack/install.sh install
.PHONY: install

install-client:
	@echo "Begin to install dfget and dfdaemon."
	./hack/install.sh install dfclient
.PHONY: install-client

install-supernode:
	@echo "Begin to install supernode."
	./hack/install.sh install supernode
.PHONY: install-supernode

uninstall:
	@echo "Begin to uninstall dfget and dfdaemon and supernode."
	./hack/install.sh uninstall
.PHONY: uninstall

uninstall-client:
	@echo "Begin to uninstall dfget and dfdaemon."
	./hack/install.sh uninstall-dfclient
.PHONY: uninstall-client

uninstall-supernode:
	@echo "Begin to uninstall supernode."
	./hack/install.sh uninstall-supernode
.PHONY: uninstall-supernode

docker-build:
	@echo "Begin to use docker build dfclient and supernode images."
	./hack/docker-build.sh
.PHONY: docker-build

docker-build-client:
	@echo "Begin to use docker build dfclient image."
	./hack/docker-build.sh dfclient
.PHONY: docker-build-client

docker-build-supernode:
	@echo "Begin to use docker build supernode image."
	./hack/docker-build.sh supernode
.PHONY: docker-build-supernode

unit-test: build-dirs
	./hack/unit-test.sh
.PHONY: unit-test

# TODO: output the log file when the test is failed
integration-test:
	@go test ./test
.PHONY: integration-test

check:
	@echo "Begin to check code formats."
	./hack/check.sh
	@echo "Begin to check dockerd whether is startup"
	./hack/check-docker.sh
.PHONY: check

go-mod-tidy:
	@echo "Begin to tidy up go.mod and go.sum"
	@go mod tidy
.PHONY: go-mod-tidy

go-mod-vendor:
	@echo "Begin to vendor go mod dependency"
	@go mod vendor
.PHONY: go-mod-vendor

check-go-mod: go-mod-tidy
	@echo "Begin to check for unused/missing packages in go.mod"
	@git diff --exit-code -- go.sum go.mod
.PHONY: check-go-mod

docs:
	@echo "Begin to generate docs of API/CLI"
	./hack/generate-docs.sh
.PHONY: docs

rpm:
	./hack/package.sh rpm
.PHONY: rpm

deb:
	./hack/package.sh deb
.PHONY: deb

release:
	./hack/package.sh
.PHONY: release

df.key:
	openssl genrsa -des3 -passout pass:x -out df.pass.key 2048
	openssl rsa -passin pass:x -in df.pass.key -out df.key
	rm df.pass.key

df.crt: df.key
	openssl req -new -key df.key -out df.csr
	openssl x509 -req -sha256 -days 365 -in df.csr -signkey df.key -out df.crt
	rm df.csr

golangci-lint:
	./hack/golangci-lint.sh
.PHONY: golangci-lint
