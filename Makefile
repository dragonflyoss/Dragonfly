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
	@rm -rf ./release
.PHONY: clean

build-dirs:
	@mkdir -p ./bin
.PHONY: build-dirs

build: build-dirs
	@echo "Begin to build dfget and dfdaemon and supernode." 
	./hack/build.sh
.PHONY: build

build-java: build-client
	@echo "Begin to build dfget and dfdaemon and java version supernode." 
	./hack/build-supernode.sh
.PHONY: build-java

build-client: build-dirs
	@echo "Begin to build dfget and dfdaemon."
	./hack/build.sh dfget
	./hack/build.sh dfdaemon
.PHONY: build-client

build-supernode: build-dirs
	@echo "Begin to build supernode."
	./hack/build.sh supernode
.PHONY: build-supernode

build-supernode-java: build-dirs
	@echo "Begin to build java version supernode."
	./hack/build-supernode.sh
.PHONY: build-supernode-java

install:
	@echo "Begin to install dfget and dfdaemon and supernode."
	./hack/install.sh install
.PHONY: install

uninstall:
	@echo "Begin to uninstall dfget and dfdaemon and supernode."
	./hack/install.sh uninstall
.PHONY: uninstall

unit-test: build-dirs
	./hack/unit-test.sh
.PHONY: unit-test

check:
	@echo "Begin to check code formats."
	./hack/check.sh	
	@echo "Begin to check java version supernode code formats."
	./hack/check-supernode.sh
.PHONY: check

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
