# Installing Dragonfly Server

This topic explains how to install the Dragonfly server.

## Context

Install supernode in one of the following ways:

- Deploying with Docker.
- Deploying with physical machines: Recommended for production usage.

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
Nginx|0.8+

## Procedure - When Deploying with Docker

### Get supernode image

You can get it from [DockerHub](https://hub.docker.com/) directly.

1. Obtain the latest Docker image ID of the supernode.

    ```sh
    docker pull dragonflyoss/supernode:1.0.0
    ```

Or you can build your own supernode image.

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
    make docker-build-supernode DF_VERSION=$TAG
    ```

4. Obtain the latest Docker image ID of the supernode.

    ```sh
    docker image ls | grep 'supernode' | awk '{print $3}' | head -n1
    ```

### Start supernode

**NOTE:** Replace ${supernodeDockerImageId} with the ID obtained at the previous step.

```sh
docker run -d --name supernode --restart=always -p 8001:8001 -p 8002:8002 -v /home/admin/supernode:/home/admin/supernode ${supernodeDockerImageId} --download-port=8001
```

## Procedure - When Deploying with Physical Machines

### Get supernode executable file

1. Download a binary package of the supernode. You can download one of the latest builds for Dragonfly on the [github releases page](https://github.com/dragonflyoss/Dragonfly/releases).

    ```sh
    version=1.0.0
    wget https://github.com/dragonflyoss/Dragonfly/releases/download/v$version/Dragonfly_$version_linux_amd64.tar.gz
    ```

2. Unzip the package.

    ```bash
    # Replace `xxx` with the installation directory.
    tar -zxf Dragonfly_1.0.0_linux_amd64.tar.gz -C xxx
    ```

3. Move the `supernode` to your `PATH` environment variable to make sure you can directly use `supernode` command.

Or you can build your own supernode executable file.

1. Obtain the source code of Dragonfly.

    ```sh
    git clone https://github.com/dragonflyoss/Dragonfly.git
    ```

2. Enter the project directory.

    ```sh
    cd Dragonfly
    ```

3. Compile the source code.

    ```sh
    make build-supernode && make install-supernode
    ```

### Start supernode

```sh
supernodeHomeDir=/home/admin/supernode
supernodeDownloadPort=8001
supernode --home-dir=$supernodeHomeDir --port=8002 --download-port=$supernodeDownloadPort
```

### Start file server

You can start a file server in any way. However, the following conditions must be met:

- It must be rooted at `${supernodeHomeDir}/repo` which is defined in the previous step.
- It must listen on the port `supernodeDownloadPort` which is defined in the previous step.

And let's take nginx as an example.

1. Add the following configuration items to the Nginx configuration file.

   ```conf
   server {
   # Must be ${supernodeDownloadPort}
   listen 8001;
   location / {
     # Must be ${supernodeHomeDir}/repo
     root /home/admin/supernode/repo;
    }
   }
   ```

2. Start Nginx.

   ```sh
   sudo nginx
   ```

## After this Task

- After supernode is installed, run the following commands to verify if Nginx and **supernode** are started, and if Port `8001` and `8002` are available.

    ```sh
    telnet 127.0.0.1 8001
    telnet 127.0.0.1 8002
    ```

- [Install the Dragonfly client](./install_client.md) and test if the downloading works.

    ```sh
    dfget --url "http://${resourceUrl}" --output ./resource.png --node "127.0.0.1:8002=1"
    ```
