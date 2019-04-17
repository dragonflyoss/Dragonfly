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

// TransparentProxy is an http proxy handler. It proxies requests with dfget
// if any defined proxy rules is matched
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
		handleHTTPS(w, r)
	} else {
		proxy.handleHTTP(w, r)
	}
}

func (proxy *TransparentProxy) handleHTTP(w http.ResponseWriter, req *http.Request) {
	rt := handler.NewDFRoundTripper(nil)
	rt.ShouldUseDfget = proxy.shouldUseDfget
	// delete the Accept-Encoding header to avoid returning the same cached
	// result for different requests
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

// shouldUseDfget returns whether we should use dfget to proxy a request. It
// also change the scheme of the given request if the matched rule has
// UseHTTPS = true
func (proxy *TransparentProxy) shouldUseDfget(req *http.Request) bool {
	if req.Method != http.MethodGet {
		return false
	}

	for _, rule := range proxy.rules {
		if rule.Match(req.URL.String()) {
			if rule.UseHTTPS {
				req.URL.Scheme = "https"
			}
			return !rule.Direct
		}
	}
	return false
}

// handleHTTPS handles a CONNECT request and proxy an https request through an
// http tunnel.
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
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	go copyAndClose(dst, clientConn)
	go copyAndClose(clientConn, dst)
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
