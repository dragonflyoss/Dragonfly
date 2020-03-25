## supernode

the central control server of Dragonfly used for scheduling and cdn cache

### Synopsis

SuperNode is a long-running process with two primary responsibilities:
It's the tracker and scheduler in the P2P network that choose appropriate downloading net-path for each peer.
It's also a CDN server that caches downloaded data from source to avoid downloading the same files from source repeatedly.

```
supernode [flags]
```

### Options

```
      --advertise-ip string             the supernode ip is the ip we advertise to other peers in the p2p-network
      --cdn-pattern string              cdn pattern, must be in ["local", "source"]. Default: local (default "local")
      --config string                   the path of supernode's configuration file (default "/etc/dragonfly/supernode.yml")
  -D, --debug                           switch daemon log level to DEBUG mode
      --down-limit int                  download limit for supernode to serve download tasks (default 4)
      --download-port int               downloadPort is the port for download files from supernode (default 8001)
      --fail-access-interval duration   fail access interval is the interval time after failed to access the URL (default 3m0s)
      --gc-initial-delay duration       gc initial delay is the delay time from the start to the first GC execution (default 6s)
      --gc-meta-interval duration       gc meta interval is the interval time to execute the GC meta (default 2m0s)
  -h, --help                            help for supernode
      --home-dir string                 homeDir is the working directory of supernode (default "/home/admin/supernode")
      --max-bandwidth rate              network rate that supernode can use (default 200MB)
      --peer-gc-delay duration          peer gc delay is the delay time to execute the GC after the peer has reported the offline (default 3m0s)
      --pool-size int                   pool size is the core pool size of ScheduledExecutorService (default 10)
      --port int                        listenPort is the port that supernode server listens on (default 8002)
      --profiler                        profiler sets whether supernode HTTP server setups profiler
      --system-bandwidth rate           network rate reserved for system (default 20MB)
      --task-expire-time duration       task expire time is the time that a task is treated expired if the task is not accessed within the time (default 3m0s)
      --up-limit int                    upload limit for a peer to serve download tasks (default 5)
```

### SEE ALSO

* [supernode config](supernode_config.md)	 - Manage the configurations of supernode
* [supernode gen-doc](supernode_gen-doc.md)	 - Generate Document for supernode command line tool in MarkDown format
* [supernode version](supernode_version.md)	 - Show the current version of supernode

