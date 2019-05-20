package proxy

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/handler"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Option is a functional option for configuring the proxy
type Option func(p *TransparentProxy) error

// WithCert sets the certificate to
func WithCert(cert *tls.Certificate) Option {
	return func(p *TransparentProxy) error {
		p.cert = cert
		return nil
	}
}

// WithHTTPSHosts sets the rules for hijacking https requests
func WithHTTPSHosts(hosts ...*config.HijackHost) Option {
	return func(p *TransparentProxy) error {
		p.httpsHosts = hosts
		return nil
	}
}

// WithCertFromFile is a convenient wrapper for WithCert, to read certificate from
// the given file
func WithCertFromFile(certFile, keyFile string) Option {
	return func(p *TransparentProxy) error {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return errors.Wrap(err, "load cert")
		}
		p.cert = &cert
		return nil
	}
}

// WithRules sets the proxy rules
func WithRules(rules []*config.Proxy) Option {
	return func(p *TransparentProxy) error { return p.SetRules(rules) }
}

// New returns a new transparent proxy with the given rules
func New(opts ...Option) (*TransparentProxy, error) {
	proxy := &TransparentProxy{}

	for _, opt := range opts {
		if err := opt(proxy); err != nil {
			return nil, err
		}
	}

	return proxy, nil
}

// TransparentProxy is an http proxy handler. It proxies requests with dfget
// if any defined proxy rules is matched
type TransparentProxy struct {
	rules []*config.Proxy

	// httpsHosts is the list of hosts whose https requests will be hijacked
	httpsHosts []*config.HijackHost

	cert *tls.Certificate
}

// remoteConfig returns the tls.Config used to connect to the given remote host.
// If the host should not be hijacked, nil will be returned.
func (proxy *TransparentProxy) remoteConfig(host string) *tls.Config {
	for _, h := range proxy.httpsHosts {
		if h.Regx.MatchString(host) {
			config := &tls.Config{}
			if h.Certs != nil {
				config.RootCAs = h.Certs.CertPool
			}
			return config
		}
	}
	return nil
}

// SetRules change the rule lists of the proxy to the given rules
func (proxy *TransparentProxy) SetRules(rules []*config.Proxy) error {
	proxy.rules = rules
	return nil
}

// ServeHTTP implements http.Handler.ServeHTTP
func (proxy *TransparentProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		proxy.handleHTTPS(w, r)
	} else {
		proxy.handleHTTP(w, r)
	}
}

func (proxy *TransparentProxy) handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := proxy.roundTripper(nil).RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (proxy *TransparentProxy) roundTripper(tlsConfig *tls.Config) http.RoundTripper {
	rt := handler.NewDFRoundTripper(tlsConfig)
	rt.ShouldUseDfget = proxy.shouldUseDfget
	return rt
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

// tunnelHTTPS handles a CONNECT request and proxy an https request through an
// http tunnel.
func tunnelHTTPS(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("Tunneling https request for %s", r.Host)
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

func (proxy *TransparentProxy) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	if proxy.cert == nil {
		tunnelHTTPS(w, r)
		return
	}

	cConfig := proxy.remoteConfig(r.Host)
	if cConfig == nil {
		tunnelHTTPS(w, r)
		return
	}

	logrus.Debugln("hijack https request to", r.Host)

	sConfig := new(tls.Config)
	sConfig.Certificates = []tls.Certificate{*proxy.cert}

	sConn, err := handshake(w, sConfig)
	if err != nil {
		logrus.Errorf("handshake failed for %s: %v", r.Host, err)
		return
	}
	defer sConn.Close()

	cConn, err := tls.Dial("tcp", r.Host, cConfig)
	if err != nil {
		logrus.Errorf("dial failed for %s: %v", r.Host, err)
		return
	}
	defer cConn.Close()

	rp := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Host = r.Host
			r.URL.Scheme = "https"
		},
		Transport: proxy.roundTripper(cConfig),
	}

	// We have to wait until the connection is closed
	wg := sync.WaitGroup{}
	wg.Add(1)
	http.Serve(&singleUseListener{&customCloseConn{sConn, wg.Done}}, rp)
	wg.Wait()
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

var okHeader = []byte("HTTP/1.1 200 OK\r\n\r\n")

// handshake hijacks w's underlying net.Conn, responds to the CONNECT request
// and manually performs the TLS handshake.
func handshake(w http.ResponseWriter, config *tls.Config) (net.Conn, error) {
	raw, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(w, "no upstream", http.StatusServiceUnavailable)
		return nil, err
	}
	if _, err = raw.Write(okHeader); err != nil {
		raw.Close()
		return nil, err
	}
	conn := tls.Server(raw, config)
	if err = conn.Handshake(); err != nil {
		conn.Close()
		raw.Close()
		return nil, err
	}
	return conn, nil
}

// A singleUseListener implements a net.Listener that returns the net.Conn specified
// in c for the first Accept call, and return errors for the subsequent calls.
type singleUseListener struct {
	c net.Conn
}

func (l *singleUseListener) Accept() (net.Conn, error) {
	if l.c == nil {
		return nil, errors.New("closed")
	}
	c := l.c
	l.c = nil
	return c, nil
}

func (l *singleUseListener) Close() error { return nil }

func (l *singleUseListener) Addr() net.Addr { return l.c.LocalAddr() }

// A customCloseConn implements net.Conn and calls f before closing the underlying net.Conn
type customCloseConn struct {
	net.Conn
	f func()
}

func (c *customCloseConn) Close() error {
	if c.f != nil {
		c.f()
		c.f = nil
	}
	return c.Conn.Close()
}
