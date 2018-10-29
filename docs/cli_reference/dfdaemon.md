---
title: "dfdaemon"
weight: 1
---

# dfdaemon

This topic explains how to use the dfdaemon command.

## NAME

dfdaemon - a proxy between pouchd/dockerd and registry used for pulling images.

## SYNOPSIS

dfdaemon [options]...

## OPTIONS

```text
  -callsystem string
    	caller name (default "com_ops_dragonfly")
  -certpem string
    	cert.pem file path
  -dfpath string
    	dfget path (default is your installed path)
  -h	help
  -hostIp string
    	dfdaemon host ip, default: 127.0.0.1 (default "127.0.0.1")
  -keypem string
    	key.pem file path
  -localrepo string
    	temp output dir of dfdaemon (default is "${HOME}/.small-dragonfly/dfdaemon/data")
  -maxprocs int
    	the maximum number of CPUs that the dfdaemon can use (default 4)
  -notbs
    	not try back source to download if throw exception (default true)
  -port uint
    	dfdaemon will listen the port (default 65001)
  -ratelimit string
    	net speed limit,format:xxxM/K
  -registry string
    	registry addr(https://abc.xx.x or http://abc.xx.x) and must exist if dfdaemon is used to mirror mode
  -rule string
    	download the url by P2P if url matches the specified pattern,format:reg1,reg2,reg3
  -urlfilter string
    	filter specified url fields (default "Signature&Expires&OSSAccessKeyId")
  -v	version
  -verbose
    	verbose
```

## FILES

### Local Repository Directory

The default local repository is: **${HOME}/.small-dragonfly/dfdaemon/data/**, you can change it by setting the option: **-localrep**.