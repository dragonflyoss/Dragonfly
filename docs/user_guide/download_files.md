---
title: "Distribuiting Files"
weight: 10
---

# File Distribution

This topic explains how to distribute files with Dragonfly.
<!--more-->

## Requirements

1.on linux host

2.need python 2.7 and make sure it's in the environment variable named PATH.

## Step By Step

### Configuration

specify cluster manager nodes by configuration file or by cmd param *cluster manager nodes is deployed [above](https://github.com/alibaba/Dragonfly/blob/master/docs/install_clustermanager.md)*

*nodeIp is cluster manager ip*

by configuration file for distributing container images or general files `vi /etc/dragonfly.conf` and add or update cluster manager nodes as follows:

```
[node]
address=nodeIp1,nodeIp2,...
```

by cmd param only for distributing general files *cmd param will cover items in configuration file*.

you must apply this param named --node every time the dfget is executed,<br/> for example `dfget -u "http://xxx.xx.x" --node nodeIp1,nodeIp2,...`

2.configure container daemon

*please ignore this step if you only distribute general files with dragonfly.*

start dfget proxy

A. you can execute `dfdaemon -h` to show help info

B. the simplest way:  `dfdaemon --registry https://xxx.xx.x` or `dfdaemon --registry http://xxx.xx.x` , "xxx.xx.x" is the domain of registry

C. dfdaemon's log info in ~/.small-dragonfly/logs/dfdaemon.log

configure daemon mirror

*standard method of configuring daemon mirror for docker*

A. `vi /etc/docker/daemon.json`, please refer to [official document](https://docs.docker.com/registry/recipes/mirror/#configure-the-cache)

B. add or update the item: `"registry-mirrors": ["http://127.0.0.1:65001"]`,65001 is default port of dfget-proxy

C. restart docker `systemctl restart docker`

### Run

To distribute general files, you can execute `dfget -h` to show help info. the simplest way: `dfget --url "http://xxx.xx.x"`. dfget' log info in ~/.small-dragonfly/logs/dfclient.log

To distribute docker images, execute `docker pull xxx/xx` as usual to download images.

> Note: "xxx/xx" is the path of image addr and can not contain registry domain that was configured in dfdaemon
`dfdaemon --registry xxxxxx`
