# Installing Dragonfly Server

This topic explains how to install the Dragonfly server with **Golang version**.

**NOTE**: The Golang version supernode is **not ready for production usage**. However, you can use it more easily in your test environment.

## Context

Install the SuperNodes in one of the following ways:

- Deploying with Docker: Recommended for quick local deployment and test.
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

### Get SuperNode image

You can get it from [DockerHub](https://hub.docker.com/) directly.

1. Obtain the latest Docker image ID of the SuperNode.

    ```sh
    docker pull dragonflyoss/supernode:0.4.3
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
    TAG="0.4.3"
    make docker-build-supernode DF_VERSION=$TAG
    ```

4. Obtain the latest Docker image ID of the SuperNode.

    ```sh
    docker image ls|grep 'supernode' |awk '{print $3}' | head -n1
    ```

### Start the SuperNode

**NOTE**: Replace ${supernodeDockerImageId} with the ID obtained at the previous step.

```sh
docker run -d --name supernode --restart=always -p 8001:8001 -p 8002:8002 -v /home/admin/supernode:/home/admin/supernode dragonflyoss/supernode:0.4.3 --download-port=8001

or

docker run -d --name supernode --restart=always -p 8001:8001 -p 8002:8002 -v /home/admin/supernode:/home/admin/supernode ${supernodeDockerImageId} --download-port=8001
```

## Procedure - When Deploying with Physical Machines

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

4. Start the SuperNode.

    ```sh
    supernode --home-dir=/home/admin/supernode --port=8002 --download-port=8001
    ```

5. Add the following configuration items to the Nginx configuration file.

    ```conf
    server {
    listen 8001;
    location / {
      # Must be ${supernode.baseHome}/repo
      root /home/admin/supernode/repo;
     }
    }
    ```

6. Start Nginx.

    ```sh
    sudo nginx
    ```

## After this Task

- After the SuperNode is installed, run the following commands to verify if Nginx and **Supernode** are started, and if Port `8001` and `8002` are available.

    ```sh
    telnet 127.0.0.1 8001
    telnet 127.0.0.1 8002
    ```

- Install the Dragonfly client and test if the downloading works.

    ```sh
    dfget --url "http://${resourceUrl}" --output ./resource.png --node "127.0.0.1:8002"
    ```
