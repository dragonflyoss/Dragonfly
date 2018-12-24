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

PKG := github.com/dragonflyoss/Dragonfly
SUPERNODE_SOURCE_HOME="${curDir}/../../src/supernode"
BUILD_IMAGE ?= golang:1.10.4
GOARCH := $(shell go env GOARCH)
GOOS := $(shell go env GOOS)
BUILD := $(shell git rev-parse HEAD)
BUILD_PATH := release/${GOOS}_${GOARCH}

LDFLAGS_DFGET = -ldflags "-X ${PKG}/cmd/dfget/app.Build=${BUILD}"
LDFLAGS_DFDAEMON = -ldflags "-X ${PKG}/cmd/dfdaemon/app.Build=${BUILD}"

ifeq ($(GOOS),darwin)
    BUILD_PATH := release
endif

clean:
	@echo "Begin to clean redundant files."
	@rm -rf ./release
.PHONY: clean

check-client:
	@echo "Begin to check client code formats."
	./hack/check-client.sh
.PHONY: check

check-supernode:
	@echo "Begin to check supernode code formats."
	./hack/check-supernode.sh

build-dirs:
	@mkdir -p .go/src/$(PKG) .go/bin .cache
	@mkdir -p $(BUILD_PATH)
.PHONY: build-dirs

build-dfget: build-dirs
	@echo "Begin to build dfget."
	@docker run                                                            \
	    --rm                                                               \
	    -ti                                                                \
	    -u $$(id -u):$$(id -g)                                             \
	    -v $$(pwd)/.go:/go                                                 \
	    -v $$(pwd):/go/src/$(PKG)                                          \
	    -v $$(pwd)/$(BUILD_PATH):/go/bin                                   \
	    -v $$(pwd)/.cache:/.cache                                          \
	    -e GOOS=$(GOOS)                                                    \
	    -e GOARCH=$(GOARCH)                                                \
	    -e CGO_ENABLED=0                                                   \
	    -w /go/src/$(PKG)                                                  \
	    $(BUILD_IMAGE)                                                     \
	    go install -v -pkgdir /go/pkg $(LDFLAGS_DFGET) ./cmd/dfget
.PHONY: build-dfget	

build-dfdaemon: build-dirs
	@echo "Begin to build dfdaemon."
	@docker run                                                            \
	    --rm                                                               \
	    -ti                                                                \
	    -u $$(id -u):$$(id -g)                                             \
	    -v $$(pwd)/.go:/go                                                 \
	    -v $$(pwd):/go/src/$(PKG)                                          \
	    -v $$(pwd)/$(BUILD_PATH):/go/bin                                         \
	    -v $$(pwd)/.cache:/.cache                                          \
	    -e GOOS=$(GOOS)                                                    \
	    -e GOARCH=$(GOARCH)                                                \
	    -e CGO_ENABLED=0                                                   \
	    -w /go/src/$(PKG)                                                  \
	    $(BUILD_IMAGE)                                                     \
	    go install -v -pkgdir /go/pkg $(LDFLAGS_DFDAEMON) ./cmd/dfdaemon
.PHONY: build-dfdaemon	

build-supernode:
	./hack/compile-supernode.sh
	./hack/build-supernode-image.sh
.PHONY: build-supernode

unit-test: build-dirs
	./hack/unit-test.sh
.PHONY: unit-test

install:
	@echo "Begin to install dfget and dfdaemon."
	./hack/install-client.sh install
.PHONY: install

uninstall:
	@echo "Begin to uninstall dfget and dfdaemon."
	./hack/install-client.sh uninstall
.PHONY: uninstall 

build-client: check-client build-dfget build-dfdaemon
.PHONY: build-client

build: check-client build-dfget build-dfdaemon check-supernode build-supernode