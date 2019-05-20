## dfget

client of Dragonfly used to download and upload files

### Synopsis

dfget is the client of Dragonfly which takes a role of peer in a P2P network.
When user triggers a file downloading task, dfget will download the pieces of
file from other peers. Meanwhile, it will act as an uploader to support other
peers to download pieces from it if it owns them. In addition, dfget has the
abilities to provide more advanced functionality, such as network bandwidth
limit, transmission encryption and so on.

```
dfget [flags]
```

### Examples

```

$ dfget -u https://www.taobao.com -o /tmp/test/b.test --notbs --expiretime 20s
--2019-02-02 18:56:34--  https://www.taobao.com
dfget version:0.3.0
workspace:/root/.small-dragonfly sign:96414-1549104994.143
client:127.0.0.1 connected to node:127.0.0.1
start download by dragonfly
download SUCCESS(0) cost:0.026s length:141898 reason:0

```

### Options

```
      --alivetime duration    Alive duration for which uploader keeps no accessing by any uploading requests, after this period uploader will automically exit (default 5m0s)
      --callsystem string     The name of dfget caller which is for debugging. Once set, it will be passed to all components around the request to make debugging easy
      --clientqueue int       specify the size of client queue which controls the number of pieces that can be processed simultaneously (default 6)
      --console               show log on console, it's conflict with '--showbar'
      --dfdaemon              identify whether the request is from dfdaemon
      --expiretime duration   caching duration for which cached file keeps no accessed by any process, after this period cache file will be deleted (default 3m0s)
  -f, --filter string         filter some query params of URL, use char '&' to separate different params
                              eg: -f 'key&sign' will filter 'key' and 'sign' query param
                              in this way, different but actually the same URLs can reuse the same downloading task
      --header strings        http header, eg: --header='Accept: *' --header='Host: abc'
  -h, --help                  help for dfget
  -i, --identifier string     The usage of identifier is making different downloading tasks generate different downloading task IDs even if they have the same URLs. conflict with --md5.
  -s, --locallimit string     network bandwidth rate limit for single download task, in format of 20M/m/K/k
  -m, --md5 string            md5 value input from user for the requested downloading file to enhance security
  -n, --node strings          specify the addresses(IP:port) of supernodes
      --notbs                 disable back source downloading for requested file when p2p fails to download it
  -o, --output string         Destination path which is used to store the requested downloading file. It must contain detailed directory and specific filename, for example, '/tmp/file.mp4'
  -p, --pattern string        download pattern, must be p2p/cdn/source, cdn and source do not support flag --totallimit (default "p2p")
  -b, --showbar               show progress bar, it is conflict with '--console'
  -e, --timeout int           Timeout set for file downloading task. If dfget has not finished downloading all pieces of file before --timeout, the dfget will throw an error and exit
      --totallimit string     network bandwidth rate limit for the whole host, in format of 20M/m/K/k
  -u, --url string            URL of user requested downloading file(only HTTP/HTTPs supported)
      --verbose               be verbose
```

### SEE ALSO

* [dfget gen-doc](dfget_gen-doc.md)	 - Generate Document for dfget command line tool with MarkDown format
* [dfget server](dfget_server.md)	 - Launch a peer server for uploading files.
* [dfget version](dfget_version.md)	 - Show the current version

