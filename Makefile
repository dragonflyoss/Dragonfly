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

clean:
	@echo "Begin to clean redundant files."
	@rm -rf ./bin
.PHONY: clean

build-dirs:
	@mkdir -p ./bin
.PHONY: build-dirs

build: build-dirs
	@echo "Begin to build dfget and dfdaemon and supernode."
	./hack/build-supernode.sh 
	./hack/build-client.sh
.PHONY: build

build-client: build-dirs
	@echo "Begin to build dfget and dfdaemon."
	./hack/build-client.sh
.PHONY: build-client

build-supernode: build-dirs
	@echo "Begin to build supernode."
	./hack/build-supernode.sh 
.PHONY: build-supernode

install:
	@echo "Begin to install dfget and dfdaemon."
	./hack/install-client.sh install
.PHONY: install

uninstall:
	@echo "Begin to uninstall dfget and dfdaemon."
	./hack/install-client.sh uninstall
.PHONY: uninstall

unit-test: build-dirs
	./hack/unit-test.sh
.PHONY: unit-test

check:
	@echo "Begin to check client code formats."
	./hack/check-client.sh	
	@echo "Begin to check supernode code formats."
	./hack/check-supernode.sh
.PHONY: check

docs:
	@echo "Begin to generate docs of API/CLI"
	./hack/generate-docs.sh
.PHONY: docs
