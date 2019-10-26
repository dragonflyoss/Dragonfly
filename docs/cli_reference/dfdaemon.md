## dfdaemon

The dfdaemon is a proxy that intercepts image download requests.

### Synopsis

The dfdaemon is a proxy between container engine and registry used for pulling images.

```
dfdaemon [flags]
```

### Options

```
      --certpem string     cert.pem file path
      --config string      the path of dfdaemon's configuration file (default "/etc/dragonfly/dfdaemon.yml")
      --dfpath string      dfget path (default "/go/src/github.com/dragonflyoss/Dragonfly/bin/linux_amd64/dfget")
  -h, --help               help for dfdaemon
      --hostIp string      dfdaemon host ip, default: 127.0.0.1 (default "127.0.0.1")
      --keypem string      key.pem file path
      --localrepo string   temp output dir of dfdaemon (default "/root/.small-dragonfly/dfdaemon/data")
      --maxprocs int       the maximum number of CPUs that the dfdaemon can use (default 4)
      --node strings       specify the addresses(host:port) of supernodes that will be passed to dfget.
      --port uint          dfdaemon will listen the port (default 65001)
      --ratelimit rate     net speed limit (default 20MB)
      --registry string    registry mirror url, which will override the registry mirror settings in the config file if presented (default "https://index.docker.io")
      --verbose            verbose
```

### SEE ALSO

* [dfdaemon gen-doc](dfdaemon_gen-doc.md)	 - Generate Document for dfdaemon command line tool with MarkDown format
* [dfdaemon version](dfdaemon_version.md)	 - Show the current version of dfdaemon

