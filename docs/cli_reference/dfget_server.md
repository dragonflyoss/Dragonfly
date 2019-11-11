## dfget server

Launch a peer server for uploading files.

### Synopsis

Launch a peer server for uploading files.

```
dfget server [flags]
```

### Options

```
      --alivetime duration    alive duration for which uploader keeps no accessing by any uploading requests, after this period uploader will automatically exit (default 5m0s)
      --data string           local directory which stores temporary files for p2p uploading
      --expiretime duration   caching duration for which cached file keeps no accessed by any process, after this period cache file will be deleted (default 3m0s)
  -h, --help                  help for server
      --home string           the work home directory of dfget server
      --ip string             IP address that server will listen on
      --meta string           meta file path
      --port int              port number that server will listen on
      --verbose               be verbose
```

### SEE ALSO

* [dfget](dfget.md)	 - client of Dragonfly used to download and upload files

