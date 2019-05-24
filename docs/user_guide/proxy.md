# Use dfdaemon as an HTTP proxy

Dfdaemon can be used as an HTTP proxy to speed up image pulling from any registry
as well as general HTTP downloads.

Please first ensure that you know how to install and run [supernode](install_server.md)
and [dfdaemon](install_client.md).

**HTTPS support is currently very limited. All HTTPS request will be tunneled
directly, without dfget.**

## Proxy rule configuration

Proxy rules are configured in `/etc/dragonfly/dfdaemon.yml`. For performance
reason, dfdaemon will handle a request with the the first matching rule.

```yaml
proxies:
# proxy requests directly, without dfget
- regx: no-proxy-reg
  direct: true
# proxy all http image layer download requests with dfget
- regx: blobs/sha256:.*
# change http requests to some-registry to https, and proxy them with dfget
- regx: some-registry/
  use_https: true
```

## Download images

Add the following content to `/etc/dragonfly/dfdaemon.yml`.

```yaml
proxies:
# proxy all http image layer download requests with dfget
- regx: blobs/sha256:.*
```

Set HTTP_PROXY for docker daemon in `/etc/systemd/system/docker.service.d/http-proxy.conf`.
`65002` is the default proxy port for dfdaemon. This can be configured with the
`--proxyPort` cli parameter of dfdaemon.

```
[Service]
Environment="HTTP_PROXY=http://127.0.0.1:65002"
```

Set your registry as insecure in `/etc/docker/daemon.json`

```json
{
  "insecure-registries": [ "your.registry" ]
}
```

Start dfdaemon and restart docker daemon.

```
systemctl restart docker
```

Pull an image to see if it works. For registries that are not configured
insecure, you can still pull image from it, but dfdaemon will not be able to
speed up your downloads with dfget.

```
docker pull nginx
docker pull your.registry/team/repo:tag
```

Then you can [check if your image is downloaded with dfget](../../FAQ.md#how-to-check-if-block-piece-is-distributed-among-dfgets-nodes).

## Download files

You can simply use `HTTP_PROXY` environment variable to let dfdaemon download
requests that match the proxy rules. This works for any program that
respects the `HTTP_PROXY` environment variable.

```
HTTP_PROXY=http://127.0.0.1:65002 curl http://github.com
```

HTTPS requests and requests that are not matched, will be proxied directly,
and dragonfly is not able to speed up them.

