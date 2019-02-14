## dfdaemon

The dfdaemon is a proxy that intercepts image download requests.

### Synopsis

The dfdaemon is a proxy between container engine and registry used for pulling images.

```
dfdaemon [flags]
```

### Options

```
      --callsystem string   caller name (default "com_ops_dragonfly")
      --certpem string      cert.pem file path
      --dfpath string       dfget path (default "/go/src/github.com/dragonflyoss/Dragonfly/bin/linux_amd64/dfget")
  -h, --help                help for dfdaemon
      --hostIp string       dfdaemon host ip, default: 127.0.0.1 (default "127.0.0.1")
      --keypem string       key.pem file path
      --localrepo string    temp output dir of dfdaemon (default "/root/.small-dragonfly/dfdaemon/data")
      --maxprocs int        the maximum number of CPUs that the dfdaemon can use (default 4)
      --notbs               not try back source to download if throw exception (default true)
      --port uint           dfdaemon will listen the port (default 65001)
      --ratelimit string    net speed limit,format:xxxM/K
      --registry string     registry addr(https://abc.xx.x or http://abc.xx.x) and must exist if dfdaemon is used to mirror mode (default "https://index.docker.io")
      --rule string         download the url by P2P if url matches the specified pattern,format:reg1,reg2,reg3
      --urlfilter string    filter specified url fields (default "Signature&Expires&OSSAccessKeyId")
      --verbose             verbose
  -v, --version             version
```

### SEE ALSO

* [dfdaemon gen-doc](dfdaemon_gen-doc.md)	 - Generate Document for dfdaemon command line tool with MarkDown format

