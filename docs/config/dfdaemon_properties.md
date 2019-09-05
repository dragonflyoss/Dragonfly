# Customize dfdaemon properties

This topic explains how to customize the dragonfly dfdaemon startup parameters.

**NOTE**: Properties holds all configurable properties of dfdaemon including `dfget` properties. By default, dragonfly config files locate `/etc/dragonfly`. You can create `dfdaemon.yml` for configuring dfdaemon startup params. When deploying with Docker, you can mount the default path when starting up dfdaemon image with `-v`.

```sh
docker run -d --net=host --name dfclient -p 65001:65001 -v /etc/dragonfly:/etc/dragonfly -v /root/.small-dragonfly:/root/.small-dragonfly dragonflyoss/dfclient:0.4.3 --registry ${http://RegistryUrl:port} --node=127.0.0.1
```

If designating port with `--port=${port}` for starting supernode docker, dfdaemon startup parameter `--node` must designate port, such as `--node=127.0.0.1:${port}`

## Parameter instructions

The following startup parameters are supported for `dfdaemon`

| Parameter | Description |
| ------------- | ------------- |
| dfget_flags |	dfget properties |
| dfpath | dfget path |
| hijack_https | HijackHTTPS is the list of hosts whose https requests should be hijacked by dfdaemon. The first matched rule will be used |
| localrepo | Temp output dir of dfdaemon, by default `.small-dragonfly/dfdaemon/data/` |
| maxprocs| The maximum number of CPUs that the dfdaemon can use |
| proxies | Proxies is the list of rules for the transparent proxy |
| ratelimit | Net speed limit,format:xxxM/K |
| registry_mirror | Registry mirror settings |
| supernodes | Specify the addresses(host:port) of supernodes, it is just to be compatible with previous versions |
| verbose | Open detail info switch |

## Examples

Parameters are configured in `/etc/dragonfly/dfdaemon.yml`.

```yaml
　　# node: specify the addresses
　　# ip: IP address that server will listen on
　　# port: port number that server will listen on
　　# expiretime: caching duration for which cached file keeps no accessed by any process(default 3min). Deploying with Docker, this param is supported after dragonfly 0.4.3
　　# alivetime: Alive duration for which uploader keeps no accessing by any uploading requests, after this period uploader will automically exit (default 5m0s)
　　# f: filter some query params of URL, use char '&' to separate different params
dfget_flags: ["--node","192.168.33.21","--verbose","--ip","192.168.33.23",
                   "--port","15001","--expiretime","3m0s","--alivetime","5m0s",
                   "-f","filterParam1&filterParam2"]
registry_mirror:
　　# url for the registry mirror
　　remote: https://index.docker.io
　　# whether to ignore https certificate errors
　　insecure: false
　　# optional certificates if the remote server uses self-signed certificates
　　certs: []
proxies:
　　# proxy all http image layer download requests with dfget
　　- regx: blobs/sha256.*
　　# change http requests to some-registry to https and proxy them with dfget
　　- regx: some-registry/
　　　　use_https: true
　　# proxy requests directly, without dfget
　　- regx: no-proxy-reg
　　　　direct: true
hijack_https:
　　# key pair used to hijack https requests
　　cert: df.crt
　　key: df.key
　　hosts:
　　　　- regx: mirror.aliyuncs.com:443  # regexp to match request hosts
　　　　# whether to ignore https certificate errors
　　　　insecure: false
　　　　# optional certificates if the host uses self-signed certificates
　　　　certs: []
```

## SEE ALSO

* [dfdaemon Reference](https://github.com/dragonflyoss/Dragonfly/blob/master/docs/cli_reference/dfdaemon.md)	 - The instruction manual of dfdaemon
* [dfdaemon config code](https://github.com/dragonflyoss/Dragonfly/blob/master/dfdaemon/config/config.go)	 - The source code of dfdaemon config
* [custom dfget properties](https://github.com/xzy256/Dragonfly/blob/master/docs/config/dfget_properties.md)	 - Custom dfget properties with `dfget.yml`
* [dfget server](https://github.com/dragonflyoss/Dragonfly/blob/master/docs/cli_reference/dfget_server.md)	 - The dfget server options