# Customize dfget properties 

This topic explains how to customize the dragonfly dfget startup parameters. 

**NOTE**: By default, dragonfly config files locate `/etc/dragonfly`. You can create `dfget.yml` for configing dfget startup params. When deploying with Docker, you can mount default path. 

## Parameter instructions

The following startup parameters are supported for `dfget`

| Parameter  | Description |
| ------------- | ------------- |
| nodes	| Nodes specify supernodes |
| localLimit | LocalLimit rate limit about a single download task,format: 20M/m/K/k |
| minRate | Minimal rate about a single download task,format: 20M/m/K/k |
| totalLimit | TotalLimit rate limit about the whole host,format: 20M/m/K/k |
| clientQueueSize | ClientQueueSize is the size of client queue, which controls the number of pieces that can be processed simultaneously. It is only useful when the Pattern equals "source". The default value is 6 |

**NOTE**: Custom more properties see [init_dfdaemon.md](init_dfdaemon.md)

## Examples

Parameters are configured in `/etc/dragonfly/dfget.yml`.

```yaml
nodes:  
　- 127.0.0.1  
　- 10.10.10.1  
minRate: 10M
localLimit: 20M  
totalLimit: 40M  
clientQueueSize: 6  
```

## SEE ALSO

* [dfget Reference ](https://github.com/dragonflyoss/Dragonfly/blob/master/docs/cli_reference/dfget.md)	 - The instruction manual of dfget
* [dfget config code](https://github.com/dragonflyoss/Dragonfly/blob/master/dfget/config/config.go)	 - The source code of dfget config