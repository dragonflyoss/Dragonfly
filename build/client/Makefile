# Copyright 1999-2018 Alibaba Group.
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

# build client
# NOTE: All actions of every rule must be started with tabs(not 4 spaces).
# Using command 'cat -e -t Makefile' to show the presence of tabs with `^I`.

BUILD_CMD=./run.sh

.PHONY: build
build: pre check dfdaemon dfget


.PHONY: pre
pre:
	@${BUILD_CMD} pre

.PHONY: check
check:
	@${BUILD_CMD} check

.PHONY: dfget
dfget:
	@${BUILD_CMD} dfget-go


.PHONY: dfdaemon
dfdaemon:
	@${BUILD_CMD} dfdaemon

.PHONY: unit-test
unit-test:
	@${BUILD_CMD} unit-test

.PHONY: package
package: 
	@${BUILD_CMD} package


.PHONY: install
install:
	@${BUILD_CMD} install


.PHONY: uninstall
uninstall:
	@${BUILD_CMD} uninstall


.PHONY: clean
clean:
	@${BUILD_CMD} clean

