# Using dragonfly with harbor

This document will help you experience how to use dragonfly with harbor.

If you are using Dragonfly in your production environment to handle production image distribution, please refer to [supernode and dfget's detailed production parameter configuration](../config).

## Prerequisites

Assuming that experiment requires us to prepare three host machines in addition to the harbor server. One to play a role of supernode, and the other two for dfclient. Then the topology of the three nodes cluster is like the following:

![quick start cluster topology](../images/quick-start-topo.png)

Then, we should provide:

1. three host nodes in a LAN, and we assume that 3 machine IPs are replaced by the following names.

    - **dfsupernode**: Dragonfly server
    - **dfclient0**: Dragonfly client one
    - **dfclient1**: Dragonfly client two

2. every node has deployed docker daemon
3. assuming that the address of harbor server is `core.harbor.domain:32443`

## Step 1: Deploy Dragonfly Server (SuperNode)

Deploy the Dragonfly server (Supernode) on the machine `dfsupernode`.

### Prepare the configuration file

Dragonfly supernode's configuration file is located in the `/etc/dragonfly/supernode.yml` directory by default. When using the container to deploy the supernode, you need to mount the configuration file to the container.

As for how to config the supernode, please refer to [supernode-config-template.yml](../config/supernode_config_template.yml).

### Start Dragonfly Server

```bash
$ docker run -d --name supernode --restart=always -p 8001:8001 -p 8002:8002 \
	-v /home/admin/supernode:/home/admin/supernode \
	-v /etc/dragonfly/supernode.yml:/etc/dragonfly/supernode.yml \
	dragonflyoss/supernode:latest --download-port=8001
```

## Step 2: Deploy Dragonfly Client (dfclient)

The following operations should be performed both on the client machine `dfclient0`, `dfclient1`.

### Prepare the configuration file

Dragonfly's configuration file is located in the `/etc/dragonfly` directory by default. When using the container to deploy the client, you need to mount the configuration file to the container.

```bash
# cat /etc/dragonfly/dfdaemon.yml
# config the dfget flags according to your own needs, please refer to https://github.com/dragonflyoss/Dragonfly/blob/master/docs/cli_reference/dfget.md
dfget_flags: ["--node","dfsupernode=1","-f","Expires&Signature"]
proxies:
  # proxy all http image layer download requests with dfget
  - regx: blobs/sha256.*
hijack_https:
  # key pair is used to hijack https requests between the caller(E.g. dockerd, containerd) and dfdaemon. You can generate them with make df.crt at the root directory of dragonfly project.
  cert: df.crt
  key: df.key
  hosts:
    - regx: core.harbor.domain:32443
      # If your registry uses a self-signed certificate, please provide the certificate
      # or choose to ignore the certificate error with `insecure: true`.
      certs: ["ca.crt"]
```

**NOTE**: You can generate the key pair(df.crt, df.key) by running [make df.crt](https://github.com/dragonflyoss/Dragonfly/blob/master/Makefile#L166) under the root directory of project.

### Start Dragonfly Client

```bash
$ docker run -d --name dfclient --restart=always -p 65001:65001 \
    -v /etc/dragonfly:/etc/dragonfly \
    -v $HOME/.small-dragonfly:/root/.small-dragonfly \
    dragonflyoss/dfclient:latest
```

## Step 3. Configure Docker Daemon

We need to modify the Docker Daemon configuration to use the Dragonfly as a pull through registry both on the client machine `dfclient0`, `dfclient1`.

1. Add your private registry to `insecure-registries` in `/etc/docker/daemon.json`, in order to ignore the certificate error:

	```
	$ cat /etc/docker/daemon.json
	{
	  "insecure-registries": ["core.harbor.domain:32443"]
	}
	```

	**Tip:** For more information on `/etc/docker/daemon.json`, see [Docker documentation](https://docs.docker.com/registry/recipes/mirror/#configure-the-cache).

2. Set dfdaemon as `HTTP_PROXY` and `HTTPS_PROXY` for docker daemon:

	```
	$ cat /etc/systemd/system/docker.service.d/http-proxy.conf
	[Service]
	Environment="HTTP_PROXY=http://127.0.0.1:65001"
	```

	and

	```
	$ cat /etc/systemd/system/docker.service.d/https-proxy.conf
	[Service]
	Environment="HTTPS_PROXY=http://127.0.0.1:65001"
	```

3. Restart Docker Daemon.

	```bash
	$ systemctl daemon-reload && systemctl restart docker.service
	```

4. Validate the docker info, and you should make sure the `HTTP Proxy` and `HTTPS Proxy` and `Insecure Registries` are configured correctly like

	```
	$ docker info
	......
	HTTP Proxy: http://127.0.0.1:65001/
	HTTPS Proxy: http://127.0.0.1:65001/
	......
	Insecure Registries:
	 core.harbor.domain:32443
	```

## Step 4: Login your harbor registry

Through the above steps, we can start to validate if Dragonfly works as expected.

And you should login the harbor on either `dfclient0` or `dfclient1`, for example:

```bash
$ docker login core.harbor.domain:32443
......
Login Succeeded
```

## Step 5: Pull images with Dragonfly

Through the above steps, we can start to validate if Dragonfly works as expected.

And then you can pull the image as usual on either `dfclient0` or `dfclient1`, for example:

```bash
docker pull core.harbor.domain:32443/library/nginx:latest
```

## Step 6: Validate Dragonfly

You can execute the following command to check if the nginx image is distributed via Dragonfly.

```bash
docker exec dfclient grep 'downloading piece' /root/.small-dragonfly/logs/dfclient.log
```

If the output of command above has content like

```
2019-03-29 15:49:53.913 INFO sign:96027-1553845785.119 : downloading piece:{"taskID":"00a0503ea12457638ebbef5d0bfae51f9e8e0a0a349312c211f26f53beb93cdc","superNode":"127.0.0.1","dstCid":"127.0.0.1-95953-1553845720.488","range":"67108864-71303167","result":503,"status":701,"pieceSize":4194304,"pieceNum":16}
```

that means that the image download is done by Dragonfly.

If you need to ensure that if the image is transferred through other peer nodes, you can execute the following command:

```bash
docker exec dfclient grep 'downloading piece' /root/.small-dragonfly/logs/dfclient.log | grep -v cdnnode
```

If the above command does not output the result, the mirror does not complete the transmission through other peer nodes. Otherwise, the transmission is completed through other peer nodes.
