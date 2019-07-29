package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/downloader"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/downloader/dfget"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/transport"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Option is a functional option for configuring the proxy
type Option func(p *Proxy) error

// WithCert sets the certificate to
func WithCert(cert *tls.Certificate) Option {
	return func(p *Proxy) error {
		p.cert = cert
		return nil
	}
}

// WithHTTPSHosts sets the rules for hijacking https requests
func WithHTTPSHosts(hosts ...*config.HijackHost) Option {
	return func(p *Proxy) error {
		p.httpsHosts = hosts
		for _, host := range p.httpsHosts {
			if host.ServerCertFile != "" && host.ServerKeyFile != "" {
				cert, err := tls.LoadX509KeyPair(host.ServerCertFile, host.ServerKeyFile)
				if err != nil {
					logrus.Errorf("failed to load self-signed certificate (%s, %s) for https hijacking, host: %s, err: %v",
						host.ServerCertFile, host.ServerKeyFile, host.Regx.String(), err)
					return err
				}
				logrus.Infof("use self-signed certificate (%s, %s) for https hijacking, host: %s", host.ServerCertFile, host.ServerKeyFile, host.Regx.String())
				host.ServerCert = &cert
			}
		}
		return nil
	}
}

// WithRegistryMirror sets the registry mirror for the proxy
func WithRegistryMirror(r *config.RegistryMirror) Option {
	return func(p *Proxy) error {
		p.registry = r
		return nil
	}
}

// WithCertFromFile is a convenient wrapper for WithCert, to read certificate from
// the given file
func WithCertFromFile(certFile, keyFile string) Option {
	return func(p *Proxy) error {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return errors.Wrap(err, "load cert")
		}
		logrus.Infof("use self-signed certificate (%s, %s) for https hijacking", certFile, keyFile)
		p.cert = &cert
		return nil
	}
}

// WithDirectHandler sets the handler for non-proxy requests
func WithDirectHandler(h *http.ServeMux) Option {
	return func(p *Proxy) error {
		// Make sure the root handler of the given server mux is the
		// registry mirror reverse proxy
		h.HandleFunc("/", p.mirrorRegistry)
		p.directHandler = h
		return nil
	}
}

// WithRules sets the proxy rules
func WithRules(rules []*config.Proxy) Option {
	return func(p *Proxy) error { return p.SetRules(rules) }
}

// WithDownloaderFactory sets the factory function to get a downloader
func WithDownloaderFactory(f downloader.Factory) Option {
	return func(p *Proxy) error {
		p.downloadFactory = f
		return nil
	}
}

// New returns a new transparent proxy with the given rules
func New(opts ...Option) (*Proxy, error) {
	proxy := &Proxy{
		directHandler: http.NewServeMux(),
	}

	for _, opt := range opts {
		if err := opt(proxy); err != nil {
			return nil, err
		}
	}

	return proxy, nil
}

// NewFromConfig returns a new transparent proxy from the given properties
func NewFromConfig(c config.Properties) (*Proxy, error) {
	opts := []Option{
		WithRules(c.Proxies),
		WithRegistryMirror(c.RegistryMirror),
		WithDownloaderFactory(func() downloader.Interface {
			return dfget.NewGetter(c.DFGetConfig())
		}),
	}

	logrus.Infof("registry mirror: %s", c.RegistryMirror.Remote)

	if len(c.Proxies) > 0 {
		logrus.Infof("%d proxy rules loaded", len(c.Proxies))
		for i, r := range c.Proxies {
			method := "with dfget"
			if r.Direct {
				method = "directly"
			}
			scheme := ""
			if r.UseHTTPS {
				scheme = "and force https"
			}
			logrus.Infof("[%d] proxy %s %s %s", i+1, r.Regx, method, scheme)
		}
	}

	if len(c.SuperNodes) > 0 {
		logrus.Infof("use supernodes: %s", strings.Join(c.SuperNodes, ","))
	}
	logrus.Infof("rate limit set to %s", c.RateLimit)

	if c.HijackHTTPS != nil {
		opts = append(opts, WithHTTPSHosts(c.HijackHTTPS.Hosts...))
		if c.HijackHTTPS.Cert != "" && c.HijackHTTPS.Key != "" {
			opts = append(opts, WithCertFromFile(c.HijackHTTPS.Cert, c.HijackHTTPS.Key))
		}
	}
	return New(opts...)
}

// Proxy is an http proxy handler. It proxies requests with dfget
// if any defined proxy rules is matched
type Proxy struct {
	// reverse proxy upstream url for the default registry
	registry *config.RegistryMirror
	// proxy rules
	rules []*config.Proxy
	// httpsHosts is the list of hosts whose https requests will be hijacked
	httpsHosts []*config.HijackHost
	// cert is the certificate used to hijack https proxy requests
	cert *tls.Certificate
	// directHandler are used to handle non proxy requests
	directHandler http.Handler
	// downloadFactory returns the downloader used for p2p downloading
	downloadFactory downloader.Factory
}

func (proxy *Proxy) mirrorRegistry(w http.ResponseWriter, r *http.Request) {
	reverseProxy := httputil.NewSingleHostReverseProxy(proxy.registry.Remote.URL)
	t, err := transport.New(
		transport.WithDownloader(proxy.downloadFactory()),
		transport.WithTLS(proxy.registry.TLSConfig()),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get transport: %v", err), http.StatusInternalServerError)
	}
	reverseProxy.Transport = t
	reverseProxy.ServeHTTP(w, r)
}

// remoteConfig returns the tls.Config used to connect to the given remote host.
// If the host should not be hijacked, nil will be returned.
func (proxy *Proxy) remoteConfig(host string) (*tls.Config, *tls.Certificate) {
	for _, h := range proxy.httpsHosts {
		if h.Regx.MatchString(host) {
			config := &tls.Config{InsecureSkipVerify: h.Insecure}
			if h.Certs != nil {
				config.RootCAs = h.Certs.CertPool
			}
			return config, h.ServerCert
		}
	}
	return nil, nil
}

// SetRules change the rule lists of the proxy to the given rules
func (proxy *Proxy) SetRules(rules []*config.Proxy) error {
	proxy.rules = rules
	return nil
}

// ServeHTTP implements http.Handler.ServeHTTP
func (proxy *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		// handle https proxy requests
		proxy.handleHTTPS(w, r)
	} else if r.URL.Scheme == "" {
		// handle direct requests
		proxy.directHandler.ServeHTTP(w, r)
	} else {
		// handle http proxy requests
		proxy.handleHTTP(w, r)
	}
}

func (proxy *Proxy) handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := proxy.roundTripper(nil).RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		logrus.Errorf("failed to write http body: %v", err)
	}
}

func (proxy *Proxy) roundTripper(tlsConfig *tls.Config) http.RoundTripper {
	rt, _ := transport.New(
		transport.WithDownloader(proxy.downloadFactory()),
		transport.WithTLS(tlsConfig),
		transport.WithCondition(proxy.shouldUseDfget),
	)
	return rt
}

// shouldUseDfget returns whether we should use dfget to proxy a request. It
// also change the scheme of the given request if the matched rule has
// UseHTTPS = true
func (proxy *Proxy) shouldUseDfget(req *http.Request) bool {
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

func (proxy *Proxy) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	cConfig, serverCert := proxy.remoteConfig(r.Host)
	if cConfig == nil || serverCert == nil && proxy.cert == nil {
		tunnelHTTPS(w, r)
		return
	}
	if serverCert == nil {
		serverCert = proxy.cert
	}

	logrus.Debugln("hijack https request to", r.Host)

	sConfig := new(tls.Config)
	sConfig.Certificates = []tls.Certificate{*serverCert}

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
