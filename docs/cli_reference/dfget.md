## dfget

The dfget is the client of Dragonfly.

### Synopsis

The dfget is the client of Dragonfly, a non-interactive P2P downloader.

```
dfget [flags]
```

### Options

```
      --alivetime duration    server will stop if there is no uploading task within this duration (default 5m0s)
      --callsystem string     system name that executes dfget
      --console               show log on console, it's conflict with '--showbar'
      --dfdaemon              caller is from dfdaemon
      --expiretime duration   server will delete cached files if these files doesn't be modification within this duration (default 3m0s)
  -f, --filter string         filter some query params of url, use char '&' to separate different params
                              eg: -f 'key&sign' will filter 'key' and 'sign' query param
                              in this way, different urls correspond one same download task that can use p2p mode
      --header strings        http header, eg: --header='Accept: *' --header='Host: abc'
  -h, --help                  help for dfget
  -i, --identifier string     identify download task, it is available merely when md5 param not exist
  -s, --locallimit string     rate limit about a single download task, its format is 20M/m/K/k
  -m, --md5 string            expected file md5
  -n, --node strings          specify supnernodes
      --notbs                 not back source when p2p fail
  -o, --output string         output path that not only contains the dir part but also name part
  -p, --pattern string        download pattern, must be 'p2p' or 'cdn' or 'source'
                              cdn/source pattern not support 'totallimit' flag (default "p2p")
  -b, --showbar               show progress bar, it's conflict with '--console'
  -e, --timeout int           download timeout(second)
      --totallimit string     rate limit about the whole host, its format is 20M/m/K/k
  -u, --url string            will download a file from this url
      --verbose               be verbose
```

### SEE ALSO

* [dfget gen-doc](dfget_gen-doc.md)	 - Generate Document for dfget command line tool with MarkDown format
* [dfget server](dfget_server.md)	 - Launch a peer server for uploading files.
* [dfget version](dfget_version.md)	 - Show the current version

