# build client

BUILD_CMD=./run.sh

.PHONY: build
build: pre dfdaemon dfget


.PHONY: pre
pre:
	@${BUILD_CMD} pre


.PHONY: dfget
dfget:
	@${BUILD_CMD} dfget


.PHONY: dfdaemon
dfdaemon:
	@${BUILD_CMD} dfdaemon


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

