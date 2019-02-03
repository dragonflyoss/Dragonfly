## dfget server

Launch a peer server for uploading files.

### Synopsis

Launch a peer server for uploading files.

```
dfget server [flags]
```

### Options

```
      --alivetime duration    server will stop if there is no uploading task within this duration (default 5m0s)
      --data string           the directory which stores temporary files for p2p uploading (default "/root/.small-dragonfly/data")
      --expiretime duration   server will delete cached files if these files doesn't be modification within this duration (default 3m0s)
  -h, --help                  help for server
      --home string           the work home of dfget server (default "/root/.small-dragonfly")
      --ip string             the ip that server will listen on
      --meta string           meta file path (default "/root/.small-dragonfly/meta/host.meta")
      --port int              the port that server will listen on
      --verbose               be verbose
```

### SEE ALSO

* [dfget](dfget.md)	 - The dfget is the client of Dragonfly.

