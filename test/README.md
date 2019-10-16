# Introduction

In order to provide Dragonfly's service with high quality, we must take testing of dragonfly seriously.

This doc will illustrate the following three parts:

* the organization of Dragonfly test
* the usage of Dragonfly test
* the development of Dragonfly test

## Organization of Test

Test in Dragonfly could be divided into following parts:

* [unit testing](https://en.wikipedia.org/wiki/Unit_testing#Description)
* [integration testing](https://en.wikipedia.org/wiki/Integration_testing)

*Unit testing* uses [go testing](https://golang.org/pkg/testing/) package, named with `_test.go` suffix and always locates in the same directory with the code tested. [client/client_test.go](../client/client_test.go) is a good example of unit test.

*Integration test* is in `dragonfly/test`, programmed with `go language`. There are two kinds of integration test:

* API test named as `api_xxx_test.go`;
* command line test named as `cli_xxx_test.go` ("xxx" represents the test point).

It uses [go-check](https://labix.org/gocheck) package, a rich testing framework for go language. It provides many useful functions, such as:

* SetUpTest: Run before each test to do some common work.
* TearDownTest: Run after each test to do some cleanup work.
* SetUpSuite: Run before each suite to do common work for the whole suite.
* TearDownSuite: Run after each suite to do cleanup work for the whole suite.

For other files, they are:

* `main_test.go` : the entrypoint of integration test.
* `utils.go`: common lib functions.
* `environment/*.go`: directory environment is used to hold environment variables.
* `command package`: package command is used to encapsulate CLI lib functions.
* `request package`: package request is used to encapsulate http request lib functions.

For Dragonfly's developer, if your code is only used in a single module, then the unit test is enough. While if your code is called by multiple modules, integration tests are required. In Dragonfly, both of them are developed with go language. More details can be gotten in [Unit Testing](#unit-testing) and [Integration Testing](#integration-testing).

## Run Test Cases Automatically

Test cases can be run via `Makefile` of this repo, or just manually.

To run the test automatically, the following prerequisites are needed:

* golang is installed and GOPATH and GOROOT is set correctly

Then you could just clone the Dragonfly source to GOPATH and run tests as following:

``` shell
# env |grep GO
GOROOT=/usr/local/go
GOPATH=/go
# cd /go/src/github.com/dragonflyoss/Dragonfly
# make test
```

Using `make -n test`, let us take a look at what `make test` has done.

```
# make -n test
bash -c "env PATH=/sbin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/usr/X11R6/bin:/usr/local/go/bin:/opt/satools:/root/bin hack/make.sh \
check build unit-test integration-test cri-test"
```

`make test` calls the hack/make.sh script to check code format, build Dragonfly daemon and client, run unit test, run integration test and run cri test.

`hack/make.sh` needs `docker` installed on test machine, as it uses `docker build` to build a test image including tools needed to run `make test`. `go` is also needed to be installed and set `GOPATH` `GOROOT` `PATH` correctly. For more information, you could check the `hack/make.sh` script.

## Run Test Cases Manually

As a Dragonfly developer, sometimes you need to run test cases manually.

In order to run unit-test or integration test, install go and configure go environment first.

``` shell
# go version
go version go1.12.10 linux/amd64
# which go
/usr/local/go/bin/go
# env |grep GO
GOROOT=/usr/local/go
GOPATH=/go
```

Then copy or clone Dragonfly source code to the GOPATH:

``` shell
# pwd
/go/src/github.com/dragonflyoss/Dragonfly
```

Make a build folder to use later:

``` shell
BUILDPATH=/tmp/dragonfly
export GOPATH=$GOPATH:$BUILDPATH
```

And please notice that files in `/tmp` directory may be deleted after reboot.

Now you could run unit test as following:

``` shell
# make unit-test
```

Or using go test $testdir to run unit test in a specified directory.

``` shell
# go test ./client
ok      github.com/dragonflyoss/Dragonfly/client    0.094s
```

There are more works to do for integration test compared with unit test.

First you need to make sure Dragonfly `supernode` binary is installed or built.

Next you need to start Dragonfly daemon and

``` shell
# supernode -D &
```

Then integration test could be run as following:

* run entire test:

``` shell
# cd test
# go test
```

* run a single test suite(all the test function will be run):

``` shell
# go test -check.f APIPingSuite
OK: 3 passed
PASS
ok      github.com/dragonflyoss/Dragonfly/test    3.081s
```

* run a single test case(only specified test function will be run):

``` shell
# go test -check.f APIPingSuite.TestPing
OK: 1 passed
PASS
ok      github.com/dragonflyoss/Dragonfly/test    0.488s
```

* run with more information:

``` shell
# go test -check.vv
```
