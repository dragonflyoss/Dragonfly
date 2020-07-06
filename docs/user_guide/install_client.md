# Installing Dragonfly Client

This topic explains how to install the Dragonfly `dfclient`.

## Context

Install the `dfclient` in one of the following ways:

- Deploying with Docker.
- Deploying with physical machines.

## Prerequisites

When deploying with Docker, the following conditions must be met.

Required Software | Version Limit
---|---
Git|1.9.1+
Docker|1.12.0+

When deploying with physical machines, the following conditions must be met.

Required Software | Version Limit
---|---
Git|1.9.1+
Golang|1.12.x

## Procedure - When Deploying with Docker

### Get dfclient image

You can get it from [DockerHub](https://hub.docker.com/) directly.

1. Obtain the latest Docker image ID of the SuperNode.

    ```sh
    docker pull dragonflyoss/dfclient:1.0.0
    ```

Or you can build your own dfclient image.

1. Obtain the source code of Dragonfly.

    ```sh
    git clone https://github.com/dragonflyoss/Dragonfly.git
    ```

2. Enter the project directory.

    ```sh
    cd Dragonfly
    ```

3. Build the Docker image.

    ```sh
    TAG="1.0.0"
    make docker-build-client DF_VERSION=$TAG
    ```

4. Obtain the latest Docker image ID of the `dfclient`.

    ```sh
    docker image ls | grep 'dfclient' | awk '{print $3}' | head -n1
    ```

### Start the dfdaemon

**NOTE:** You should prepare the [config files](../config) which should locate under `/etc/dragonfly` by default.

```sh
version=1.0.0
# Replace ${supernode} with your own supernode node with format `ip:port=weight`.
SUPERNODE=$supernode
docker run -d --name dfclient --restart=always -p 65001:65001 -v $HOME/.small-dragonfly:/root/.small-dragonfly -v /etc/dragonfly:/etc/dragonfly dragonflyoss/dfclient:$version --node $SUPERNODE
```

## Procedure - When Deploying with Physical Machines

### Get dfclient executable file

1. Download a binary package of the SuperNode. You can download one of the latest builds for Dragonfly on the [github releases page](https://github.com/dragonflyoss/Dragonfly/releases).

    ```sh
    version=1.0.0
    wget https://github.com/dragonflyoss/Dragonfly/releases/download/v$version/Dragonfly_$version_linux_amd64.tar.gz
    ```

2. Unzip the package.

    ```bash
    # Replace `xxx` with the installation directory.
    tar -zxf Dragonfly_1.0.0_linux_amd64.tar.gz -C xxx
    ```

3. Move the `dfget` and `dfdaemon` to your `PATH` environment variable to make sure you can directly use `dfget` and `dfdaemon` command.

Or you can build your own dfclient executable files.

1. Obtain the source code of Dragonfly.

    ```sh
    git clone https://github.com/dragonflyoss/Dragonfly.git
    ```

2. Enter the project directory.

    ```sh
    cd Dragonfly
    ```

3. Build `dfdaemon` and `dfget`.

    ```sh
    make build-client && make install-client
    ```

### Start the dfdaemon

**NOTE:** You can ignore this step when using only dfget for file distribution .

```sh
# Replace ${supernode} with your own supernode node with format `ip:port=weight`.
SUPERNODE=$supernode
dfdaemon --node $SUPERNODE
```

## After this Task

Test if the downloading works.

```sh
dfget --url "http://${resourceUrl}" --output ./resource.png --node "127.0.0.1:8002"
```

And test dfdaemon by [pulling an image](./download_files.md).
