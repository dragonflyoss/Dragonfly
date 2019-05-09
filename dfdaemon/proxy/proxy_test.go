package proxy

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/stretchr/testify/assert"
)

type testItem struct {
	URL      string
	Direct   bool
	UseHTTPS bool
}

type testCase struct {
	Error error
	Rules []*config.Proxy
	Items []testItem
}

func newTestCase() *testCase {
	return &testCase{}
}

func (tc *testCase) WithRule(regx string, direct bool, useHTTPS bool) *testCase {
	if tc.Error != nil {
		return tc
	}

	var r *config.Proxy
	r, tc.Error = config.NewProxy(regx, useHTTPS, direct)
	tc.Rules = append(tc.Rules, r)
	return tc
}

func (tc *testCase) WithTest(url string, direct bool, useHTTPS bool) *testCase {
	tc.Items = append(tc.Items, testItem{url, direct, useHTTPS})
	return tc
}

func (tc *testCase) Test(t *testing.T) {
	a := assert.New(t)
	if !a.Nil(tc.Error) {
		return
	}
	tp, err := New(WithRules(tc.Rules))
	if !a.Nil(err) {
		return
	}
	for _, item := range tc.Items {
		req, err := http.NewRequest("GET", item.URL, nil)
		if !a.Nil(err) {
			continue
		}
		if !a.Equal(tp.shouldUseDfget(req), !item.Direct) {
			fmt.Println(item.URL)
		}
		if item.UseHTTPS {
			a.Equal(req.URL.Scheme, "https")
		} else {
			a.Equal(req.URL.Scheme, "http")
		}
	}
}

func TestMatch(t *testing.T) {
	newTestCase().
		WithRule("/blobs/sha256/", false, false).
		WithTest("http://index.docker.io/v2/blobs/sha256/xxx", false, false).
		WithTest("http://index.docker.io/v2/auth", true, false).
		Test(t)

	newTestCase().
		WithRule("/a/d", false, true).
		WithRule("/a/b", true, false).
		WithRule("/a", false, false).
		WithRule("/a/c", true, false).
		WithRule("/a/e", false, true).
		WithTest("http://h/a", false, false).   // should match /a
		WithTest("http://h/a/b", true, false).  // should match /a/b
		WithTest("http://h/a/c", false, false). // should match /a, not /a/c
		WithTest("http://h/a/d", false, true).  // should match /a/d and use https
		WithTest("http://h/a/e", false, false). // should match /a, not /a/e
		Test(t)
}
