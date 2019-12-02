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

.DEFAULT_GOAL:=help

# You can assign 1 to USE_DOCKER so that you can build components of Dragonfly
# with docker.
USE_DOCKER ?= 0 # Default: build components in the local environment.

# Assign the Dragonfly version to DF_VERSION as the image tag.
DF_VERSION ?= latest # Default: use latest as the image tag which built by docker.

# Default: in order to use go mod we have to export GO111MODULE=on.
export GO111MODULE := on

# Use GOPROXY environment variable if set
GOPROXY := $(shell go env GOPROXY)
ifeq ($(GOPROXY),)
    GOPROXY := https://goproxy.io
endif
export GOPROXY

help:  ## Display this help information
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n"} /^[a-zA-Z_-]+:.*?##/ { printf " \033[36m make %-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

clean:  ## Clean up redundant files
	@echo "Begin to clean redundant files."
	@rm -rf ./bin
	@rm -rf ./release
.PHONY: clean

build-dirs: ## Prepare required folders for build
	@mkdir -p ./bin
.PHONY: build-dirs

build: build-dirs  ## Build dfget, dfdaemon and supernode
	@echo "Begin to build dfget, dfdaemon and supernode."
	./hack/build.sh
.PHONY: build

build-client: build-dirs  ## Build dfget and dfdaemon
	@echo "Begin to build dfget and dfdaemon."
	./hack/build.sh dfget
	./hack/build.sh dfdaemon
.PHONY: build-client

build-supernode: build-dirs  ## Build supernode
	@echo "Begin to build supernode."
	./hack/build.sh supernode
.PHONY: build-supernode

install:  ## Install dfget, dfdaemon and supernode
	@echo "Begin to install dfget, dfdaemon and supernode."
	./hack/install.sh install
.PHONY: install

install-client:  ## Install dfget and dfdaemon
	@echo "Begin to install dfget and dfdaemon."
	./hack/install.sh install dfclient
.PHONY: install-client

install-supernode:  ## Install supernode
	@echo "Begin to install supernode."
	./hack/install.sh install supernode
.PHONY: install-supernode

uninstall:  ## Uninstall dfget, dfdaemon and supernode
	@echo "Begin to uninstall dfget, dfdaemon and supernode."
	./hack/install.sh uninstall
.PHONY: uninstall

uninstall-client:  ## Uninstall dfget and dfdaemon
	@echo "Begin to uninstall dfget and dfdaemon."
	./hack/install.sh uninstall-dfclient
.PHONY: uninstall-client

uninstall-supernode:  ## Uninstall supernode
	@echo "Begin to uninstall supernode."
	./hack/install.sh uninstall-supernode
.PHONY: uninstall-supernode

docker-build:  ## Build dfclient and supernode images
	@echo "Begin to use docker build dfclient and supernode images."
	./hack/docker-build.sh
.PHONY: docker-build

docker-build-client:  ## Build dfclient image
	@echo "Begin to use docker build dfclient image."
	./hack/docker-build.sh dfclient
.PHONY: docker-build-client

docker-build-supernode:  ## Build supernode image
	@echo "Begin to use docker build supernode image."
	./hack/docker-build.sh supernode
.PHONY: docker-build-supernode

unit-test: build-dirs  ## Run unit test
	./hack/unit-test.sh
.PHONY: unit-test

# TODO: output the log file when the test is failed
integration-test:  ## Run integration test
	@go test ./test
.PHONY: integration-test

boilerplate-check:  ## Check code boilerplate
	@echo "Begin to check code boilerplate."
	./hack/boilerplate-check.sh
.PHONY: boilerplate-check

go-mod-tidy:  ## Tidy up go.mod and go.sum
	@echo "Begin to tidy up go.mod and go.sum"
	@go mod tidy
.PHONY: go-mod-tidy

check-go-mod: go-mod-tidy  ## Check for unused/missing packages in go.mod
	@echo "Begin to check for unused/missing packages in go.mod"
	@git diff --exit-code -- go.sum go.mod
.PHONY: check-go-mod

docs:  ## Generate docs of API/CLI
	@echo "Begin to generate docs of API/CLI"
	./hack/generate-docs.sh
.PHONY: docs

rpm:  ## Build rpm package
	./hack/package.sh rpm
.PHONY: rpm

deb:  ## Build deb package
	./hack/package.sh deb
.PHONY: deb

release:  ## Build a release
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

golangci-lint:  ## Run golangci-lint
	./hack/golangci-lint.sh
.PHONY: golangci-lint
