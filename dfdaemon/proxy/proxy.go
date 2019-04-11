package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/handler"
	"github.com/sirupsen/logrus"
)

// New returns a new transparent proxy with the given rules
func New(rules []*config.Proxy) (*TransparentProxy, error) {
	proxy := &TransparentProxy{}
	if err := proxy.SetRules(rules); err != nil {
		return nil, fmt.Errorf("invalid rules: %v", err)
	}
	return proxy, nil
}

type TransparentProxy struct {
	rules []*config.Proxy
}

// SetRules change the rule lists of the proxy to the given rules
func (proxy *TransparentProxy) SetRules(rules []*config.Proxy) error {
	proxy.rules = rules
	return nil
}

// ServeHTTP implements http.Handler.ServeHTTP
func (proxy *TransparentProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		HandleHTTPS(w, r)
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

// ShouldUseDfget returns whether we should use dfget to proxy a request
func (proxy *TransparentProxy) ShouldUseDfget(req *http.Request) bool {
	if req.Method != http.MethodGet {
		return false
	}

	useDfget := false
	for _, rule := range proxy.rules {
		if rule.Match(req.URL.String()) {
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

// HandleHTTPS handles a CONNECT request and proxy an https request through an
// http tunnel.
func HandleHTTPS(w http.ResponseWriter, r *http.Request) {
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
