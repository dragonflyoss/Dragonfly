
# build client


.PHONY : build
build:
	rm -rf temp
	mkdir temp

	cp -r ${df_home}/src/getter/* ./temp
	export GOPATH=${df_home}/build/client;cd ${build_daemon_home}/src/df-daemon;go build
	export GOPATH=${go_path}
	mv ${build_daemon_home}/src/df-daemon/df-daemon ./temp

	chmod a+x ./temp/df-daemon
	chmod a+x ./temp/dfget


.PHONY : package
package:
	rm -rf ./temp1/df-client
	mkdir -p ./temp1/df-client
	cp -r ./temp/* ./temp1/df-client
	cd ./temp1;tar czf ${df_install_home}/df-client.tar.gz ./df-client
	rm -rf ./temp1


.PHONY : install
install:
	mkdir -p ${df_install_home}/df-client
	rm -rf ${df_install_home}/df-client/*
	cp -r ./temp/* ${df_install_home}/df-client


.PHONY : clean
clean:
	rm -rf temp
	rm -f Makefile
	rm -rf ${build_daemon_home%/github.com*}


