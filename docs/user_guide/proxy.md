# Use dfdaemon as an HTTP proxy

## Prerequisites

You need to first install and configure [supernode](install_server.md) and [dfdaemon](install_client.md).

## Proxy Configuration

Proxy rules are configured in `/etc/dragonfly/dfdaemon.yml`.

```yaml
# Requests that match the regular expressions will be proxied with dfget,
# otherwise they'll be proxied directly. Requests will be handled by the first
# matching rule.
proxies:
  # proxy all http image layer download requests with dfget
- regx: blobs/sha256.*
  # proxy requests directly, without dfget
- regx: no-proxy-reg
  direct: true
  # change http requests to some-registry to https, and proxy them with dfget
- regx: some-registry/
  use_https: true

# If an https request's host matches any of the hijacking rules, dfdaemon will
# decrypt the request with given key pair and proxy it with the proxy rules.
hijack_https:
  cert: df.crt
  key: df.key
  hosts:
    # match hosts by regular expressions. certificate will be validated normally
  - regx: host-1
    # ignore certificate errors
  - regx: host-2
    insecure: true
    # use the given certificate for validation
  - regx: host-3
    certs: ["server.crt"]
```

## Usage

You can use dfdaemon like any other HTTP proxy. For example on linux and
macOS, you can use the `HTTP_PROXY` or `HTTPS_PROXY` environment variables.

## Get the Certificate of Your Server

```
openssl x509 -in <(openssl s_client -showcerts -servername xxx -connect xxx:443 -prexit 2>/dev/null)
```
