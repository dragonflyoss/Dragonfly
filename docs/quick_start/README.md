# Dragonfly Quick Start

Dragonfly Quick Start document aims to help you to quick start Dragonfly journey. This experiement is quite easy and simplified.

If you are using Dragonfly in your production environment to handle production image distribution, please refer to supernode and dfget's detailed production parameter configuration.

## Prerequisites

All steps in this document are done on the same machine using the docker container, so make sure the docker container engine is installed and started on your machine. You can also refer to the documentation: [multi-machine deployment] (../userguide/multi_machines_deployment.md) to experience Dragonfly.

## Step 1: Deploy Dragonfly Server (SuperNode)

```bash
docker run -d --name supernode --restart=always -p 8001:8001 -p 8002:8002 \
    dragonflyoss/supernode:0.4.0 -Dsupernode.advertiseIp=127.0.0.1
```

> **NOTE**: `supernode.advertiseIp` should be the ip that clients can connect to, `127.0.0.1` here is an example for testing, and it can only be used if the server and client are in the same machine.

## Step 2：Deploy Dragonfly Client (dfclient)

```bash
docker run -d --name dfclient -v /etc/localtime:/etc/localtime:ro -p 65001:65001 dragonflyoss/dfclient:0.4.0 --registry https://index.docker.io
```

**NOTE**: The `--registry` parameter specifies the mirrored image registry address,`-v /etc/localtime:/etc/localtime:ro` makes docker container and the host time synchronization,and `https://index.docker.io` is the address of official image registry, you can also set it to the others.

## Step 3. Configure Docker Daemon

We need to modify the Docker Daemon configuration to use the Dragonfly as a pull through registry.

1. Add or update the configuration item `registry-mirrors` in the configuration file`/etc/docker/daemon.json`.

```json
{
  "registry-mirrors": ["http://127.0.0.1:65001"]
}
```

**Tip:** For more information on `/etc/docker/daemon.json`, see [Docker documentation](https://docs.docker.com/registry/recipes/mirror/#configure-the-cache).

2. Restart Docker Daemon。

```bash
systemctl restart docker
```

## Step 4：Pull images with Dragonfly

Through the above steps, we can start to validate if Dragonfly works as expected.

And you can pull the image as usual, for example:

```bash
docker pull nginx:latest
```

## Step 5：Validate Dragonfly

You can execute the following command to check if the nginx image is distributed via Dragonfly.

```bash
docker exec dfclient grep 'downloading piece' /root/.small-dragonfly/logs/dfclient.log
```

If the output of command above has content like

```
2019-03-29 15:49:53.913 INFO sign:96027-1553845785.119 : downloading piece:{"taskID":"00a0503ea12457638ebbef5d0bfae51f9e8e0a0a349312c211f26f53beb93cdc","superNode":"127.0.0.1","dstCid":"127.0.0.1-95953-1553845720.488","range":"67108864-71303167","result":503,"status":701,"pieceSize":4194304,"pieceNum":16}
```

then Dragonfly is proved to work successfully.
