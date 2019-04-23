package gostub

import (
	"fmt"
	"reflect"
)

// Stub replaces the value stored at varToStub with stubVal.
// varToStub must be a pointer to the variable. stubVal should have a type
// that is assignable to the variable.
func Stub(varToStub interface{}, stubVal interface{}) *Stubs {
	return New().Stub(varToStub, stubVal)
}

// StubFunc replaces a function variable with a function that returns stubVal.
// funcVarToStub must be a pointer to a function variable. If the function
// returns multiple values, then multiple values should be passed to stubFunc.
// The values must match be assignable to the return values' types.
func StubFunc(funcVarToStub interface{}, stubVal ...interface{}) *Stubs {
	return New().StubFunc(funcVarToStub, stubVal...)
}

type envVal struct {
	val string
	ok  bool
}

// Stubs represents a set of stubbed variables that can be reset.
type Stubs struct {
	// stubs is a map from the variable pointer (being stubbed) to the original value.
	stubs   map[reflect.Value]reflect.Value
	origEnv map[string]envVal
}

// New returns Stubs that can be used to stub out variables.
func New() *Stubs {
	return &Stubs{
		stubs:   make(map[reflect.Value]reflect.Value),
		origEnv: make(map[string]envVal),
	}
}

// Stub replaces the value stored at varToStub with stubVal.
// varToStub must be a pointer to the variable. stubVal should have a type
// that is assignable to the variable.
func (s *Stubs) Stub(varToStub interface{}, stubVal interface{}) *Stubs {
	v := reflect.ValueOf(varToStub)
	stub := reflect.ValueOf(stubVal)

	// Ensure varToStub is a pointer to the variable.
	if v.Type().Kind() != reflect.Ptr {
		panic("variable to stub is expected to be a pointer")
	}

	if _, ok := s.stubs[v]; !ok {
		// Store the original value if this is the first time varPtr is being stubbed.
		s.stubs[v] = reflect.ValueOf(v.Elem().Interface())
	}

	// *varToStub = stubVal
	v.Elem().Set(stub)
	return s
}

// StubFunc replaces a function variable with a function that returns stubVal.
// funcVarToStub must be a pointer to a function variable. If the function
// returns multiple values, then multiple values should be passed to stubFunc.
// The values must match be assignable to the return values' types.
func (s *Stubs) StubFunc(funcVarToStub interface{}, stubVal ...interface{}) *Stubs {
	funcPtrType := reflect.TypeOf(funcVarToStub)
	if funcPtrType.Kind() != reflect.Ptr ||
		funcPtrType.Elem().Kind() != reflect.Func {
		panic("func variable to stub must be a pointer to a function")
	}
	funcType := funcPtrType.Elem()
	if funcType.NumOut() != len(stubVal) {
		panic(fmt.Sprintf("func type has %v return values, but only %v stub values provided",
			funcType.NumOut(), len(stubVal)))
	}

	return s.Stub(funcVarToStub, FuncReturning(funcPtrType.Elem(), stubVal...).Interface())
}

// FuncReturning creates a new function with type funcType that returns results.
func FuncReturning(funcType reflect.Type, results ...interface{}) reflect.Value {
	var resultValues []reflect.Value
	for i, r := range results {
		var retValue reflect.Value
		if r == nil {
			// We can't use reflect.ValueOf(nil), so we need to create the zero value.
			retValue = reflect.Zero(funcType.Out(i))
		} else {
			// We cannot simply use reflect.ValueOf(r) as that does not work for
			// interface types, as reflect.ValueOf receives the dynamic type, which
			// is the underlying type. e.g. for an error, it may *errors.errorString.
			// Instead, we make the return type's expected interface value using
			// reflect.New, and set the data to the passed in value.
			tempV := reflect.New(funcType.Out(i))
			tempV.Elem().Set(reflect.ValueOf(r))
			retValue = tempV.Elem()
		}
		resultValues = append(resultValues, retValue)
	}
	return reflect.MakeFunc(funcType, func(_ []reflect.Value) []reflect.Value {
		return resultValues
	})
}

// Reset resets all stubbed variables back to their original values.
func (s *Stubs) Reset() {
	for v, originalVal := range s.stubs {
		v.Elem().Set(originalVal)
	}
	s.resetEnv()
}

// ResetSingle resets a single stubbed variable back to its original value.
func (s *Stubs) ResetSingle(varToStub interface{}) {
	v := reflect.ValueOf(varToStub)
	originalVal, ok := s.stubs[v]
	if !ok {
		panic("cannot reset variable as it has not been stubbed yet")
	}

	v.Elem().Set(originalVal)
}
