+++
title = "Quick Start"
weight = 20
pre = "<b>2. </b>"
+++

Simply by starting a supernode in your Docker container, and installing the Dragonfly client, you can start downloading with Dragonfly.
<!--more-->

## Prerequisites

You have started your Docker container.

## Starting a Supernode in Your Docker Container

1. Pull the docker image we provided.

    ```bash
    # Replace ${imageName} with the real image name
    docker pull ${imageName}
    ```

2. Start a supernode.

    ```bash
    # Replace ${imageName} with the real image name
    docker run -d -p 8001:8001 -p 8002:8002 ${imageName}
    ```

We provided two images in different locations:

- China: `registry.cn-hangzhou.aliyuncs.com/alidragonfly/supernode:0.2.0`
- US: `registry.us-west-1.aliyuncs.com/alidragonfly/supernode:0.2.0`

For example, if you're in China, run the following commands:

```bash
docker pull registry.cn-hangzhou.aliyuncs.com/alidragonfly/supernode:0.2.0
docker run -d -p 8001:8001 -p 8002:8002 registry.cn-hangzhou.aliyuncs.com/alidragonfly/supernode:0.2.0
```

## Installing Dragonfly Client

1. Download a package of the client.

    ```bash
    cd $HOME
    # Replace ${package} with a package appropriate for your operating system and location
    wget ${package}
    ```

2. Unzip the package.

    ```bash
    tar -zxf df-client_0.2.0_linux_amd64.tar.gz
    ```

3. Add the directory of `df-client` to your `PATH` environment variable to make sure you can directly use `dfget` and `dfdaemon` command.

    ```bash
    # Execute or add this line to ~/.bashrc
    export PATH=$PATH:$HOME/df-client/
    ```

We provided different packages to suit your need. Please choose one and replace the `${package}` with it.

- If you're in China:

    - [Linux 64-bit](http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_linux_amd64.tar.gz): `http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_linux_amd64.tar.gz`

    - [MacOS 64-bit](http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_darwin_amd64.tar.gz): `http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_darwin_amd64.tar.gz`

- If you're not in China:

    - [Linux 64-bit](https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_linux_amd64.tar.gz): `https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_linux_amd64.tar.gz`

    - [MacOS 64-bit](https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_darwin_amd64.tar.gz): `https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_darwin_amd64.tar.gz`

For example, if you're in China and using Linux, run the following commands:

```bash
cd $HOME
wget http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_linux_amd64.tar.gz
tar -zxf df-client_0.2.0_linux_amd64.tar.gz
# execute or add this line to ~/.bashrc
export PATH=$PATH:$HOME/df-client/
```

## Downloading a File with Dragonfly

Once you have installed the Dragonfly client, you can use the `dfget` command to download a file.

```bash
dfget -u 'https://github.com/dragonflyoss/Dragonfly/blob/master/docs/images/logo.png' -o /tmp/logo.png
```

{{% notice tip %}}
For more information on the dfget command, see [dfget]({{< ref "dfget.md" >}}).
{{% /notice %}}

## Pulling an Image with Dragonfly

1. Start `dfdaemon` with a specified registry, such as `https://index.docker.io`.

    ```bash
    nohup dfdaemon --registry https://index.docker.io > /dev/null 2>&1 &
    ```

2. Add the following line to the dockerd configuration file [/etc/docker/daemon.json](https://docs.docker.com/registry/recipes/mirror/#configure-the-docker-daemon).

    ```json
    "registry-mirrors": ["http://127.0.0.1:65001"]
    ```

3. Restart dockerd.

    ```bash
    systemctl restart docker
    ```

4. Download an image with Dragonfly.

    ```bash
    docker pull nginx:latest
    ```

## Related Topics

- [Installing Server]({{< ref "install_server.md" >}})
- [Installing Client]({{< ref "install_client.md" >}})
- [Downloading Files]({{< ref "download_files.md" >}})
- [supernode Configuration]({{< ref "supernode_configuration.md" >}})
- [dfget]({{< ref "dfget.md" >}})