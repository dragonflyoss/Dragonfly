/*
Package gostub is used for stubbing variables in tests, and resetting the
original value once the test has been run.

This can be used to stub static variables as well as static functions. To
stub a static variable, use the Stub function:
  var configFile = "config.json"

  func GetConfig() ([]byte, error) {
    return ioutil.ReadFile(configFile)
  }

  // Test code
  stubs := gostub.Stub(&configFile, "/tmp/test.config")

  data, err := GetConfig()
  // data will now return contents of the /tmp/test.config file

gostub can also stub static functions in a test by using a variable
to reference the static function, and using that local variable to call
the static function:
  var timeNow = time.Now

  func GetDate() int {
  	return timeNow().Day()
  }

You can test this by using gostub to stub the timeNow variable:
  stubs := gostub.Stub(&timeNow, func() time.Time {
    return time.Date(2015, 6, 1, 0, 0, 0, 0, time.UTC)
  })
  defer stubs.Reset()

  // Test can check that GetDate returns 6

If you are stubbing a function to return a constant value like in
the above test, you can use StubFunc instead:
  stubs := gostub.StubFunc(&timeNow, time.Date(2015, 6, 1, 0, 0, 0, 0, time.UTC))
  defer stubs.Reset()

StubFunc can also be used to stub functions that return multiple values:
  var osHostname = osHostname
  // [...] production code using osHostname to call it.

  // Test code:
  stubs := gostub.StubFunc(&osHostname, "fakehost", nil)
  defer stubs.Reset()

The Reset method should be deferred to run at the end of the test to reset
all stubbed variables back to their original values.

You can set up multiple stubs by calling Stub again:
  stubs := gostub.Stub(&v1, 1)
  stubs.Stub(&v2, 2)
  defer stubs.Reset()

For simple cases where you are only setting up simple stubs, you can condense
the setup and cleanup into a single line:
  defer gostub.Stub(&v1, 1).Stub(&v2, 2).Reset()
This sets up the stubs and then defers the Reset call.

You should keep the return argument from the Stub call if you need to change
stubs or add more stubs during test execution:
  stubs := gostub.Stub(&v1, 1)
  defer stubs.Reset()

  // Do some testing
  stubs.Stub(&v1, 5)

  // More testing
  stubs.Stub(&b2, 6)

The Stub call must be passed a pointer to the variable that should be stubbed,
and a value which can be assigned to the variable.
*/
package gostub

//go:generate godocdown -template README.md.tmpl --output=README.md
