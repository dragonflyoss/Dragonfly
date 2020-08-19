# Use Dfdaemon as HTTP Proxy for Docker Daemon

Currently, docker doesn't support private registries with `registry-mirrors`,
in order to do so, we need to use HTTP proxy for docker daemon.

To use dfdaemon as HTTP proxy, first you need to add a proxy rule in
`/etc/dragonfly/dfdaemon.yml`:

```yaml
proxies:
- regx: blobs/sha256.*
```

This will proxy all requests for image layers with dfget.

By default, only HTTP requests are proxied with dfget. If you're using an HTTPS
enabled private registry, you need to add the following HTTPS configuration to
`/etc/dragonfly/dfdaemon.yml`:

```yaml
hijack_https:
  cert: df.crt
  key: df.key
  hosts:
  - regx: your.private.registry
```

If your registry uses a self-signed certificate, you can either choose to
ignore the certificate error with:

```yaml
  hosts:
  - regx: your.private.registry
    insecure: true
```

Or provide a certificate with:

```yaml
  hosts:
  - regx: your.private.registry
    certs: ["server.crt"]
```

You can get the certificate of your server with:

```
openssl x509 -in <(openssl s_client -showcerts -servername xxx -connect xxx:443 -prexit 2>/dev/null)
```

Add your private registry to `insecure-registries` in
`/etc/docker/daemon.json`, in order to ignore the certificate error:

```json
{
  "insecure-registries": ["your.private.registry"]
}
```

Set dfdaemon as HTTP_PROXY and HTTPS_PROXY for docker daemon in
`/etc/systemd/system/docker.service.d/http-proxy.conf`:

```
[Service]
Environment="HTTP_PROXY=http://127.0.0.1:65001"
Environment="HTTPS_PROXY=http://127.0.0.1:65001"
```

Read [Control Docker with systemd](https://docs.docker.com/config/daemon/systemd/#httphttps-proxy) for more details. If you're not running docker daemon with systemd, you need to set the environment variables manually.

Finally you can restart docker daemon and pull images as you normally would.

More details on dfdaemon's proxy configuration can be found
[here](../proxy.md).
