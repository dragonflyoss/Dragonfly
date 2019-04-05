package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/handler"
	"github.com/sirupsen/logrus"
)

func New(rules []Rule) *TransparentProxy {
	proxy := &TransparentProxy{}
	proxy.SetRules(rules)
	return proxy
}

type Rule struct {
	Match    string
	match    *regexp.Regexp
	UseHTTPS bool
	Direct   bool
}

type TransparentProxy struct {
	rules []Rule
}

func (proxy *TransparentProxy) SetRules(rules []Rule) error {
	compiled := []Rule{}
	for i, r := range rules {
		reg, err := regexp.Compile(r.Match)
		if err != nil {
			return fmt.Errorf("rule %d is invalid: %s", i, err.Error())
		}
		compiled = append(compiled, Rule{
			Match:    r.Match,
			match:    reg,
			UseHTTPS: r.UseHTTPS,
			Direct:   r.Direct,
		})
	}
	proxy.rules = compiled
	return nil
}

func (proxy *TransparentProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		handleHTTPS(w, r)
	} else {
		proxy.handleHTTP(w, r)
	}
}

func (proxy *TransparentProxy) handleHTTP(w http.ResponseWriter, req *http.Request) {
	rt := handler.NewDFRoundTripper(nil)
	rt.ShouldUseDfget = proxy.ShouldUseDfget
	if proxy.ShouldUseDfget(req) {
		logrus.Debugf("Dfget proxy: %s", req.URL.String())
	} else {
		logrus.Debugf("Direct proxy: %s %s", req.Method, req.URL.String())
	}
	req.Header.Del("Accept-Encoding")
	resp, err := rt.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (proxy *TransparentProxy) ShouldUseDfget(req *http.Request) bool {
	if req.Method != http.MethodGet {
		return false
	}

	useDfget := false
	for _, rule := range proxy.rules {
		if rule.match.MatchString(req.URL.String()) {
			if rule.Direct {
				return false
			}
			useDfget = true
			if rule.UseHTTPS {
				req.URL.Scheme = "https"
			}
		}
	}
	return useDfget
}

func handleHTTPS(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("Tunneling https request %s", r.URL.String())
	dst, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	go copyAndClose(dst, client_conn)
	go copyAndClose(client_conn, dst)
}

func copyAndClose(dst io.WriteCloser, src io.ReadCloser) {
	io.Copy(dst, src)
	dst.Close()
	src.Close()
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
