# Installing Server

This topic explains how to install the Dragonfly server.

**Tip:** For a data center or a cluster, we recommend that you use at least two machines with eight cores, 16GB RAM and Gigabit Ethernet connections for deploying supernodes.

## Context

There are two layers in Dragonfly’s architecture: server (supernodes) and client (hosts). Install the supernodes in one of the following ways:

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
JDK|1.7+
Maven|3.0.3+
Nginx|0.8+

## Procedure - When Deploying with Docker

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
    make build-supernode
    ```

4. Obtain the latest Docker image ID of the supernode.

    ```sh
    docker image ls|grep 'supernode' |awk '{print $3}' | head -n1
    ```

5. Start the supernode.

    ```sh
    # Replace ${supernodeDockerImageId} with the ID obtained at the previous step
    docker run -d -p 8001:8001 -p 8002:8002 ${supernodeDockerImageId}
    ```

## Procedure - When Deploying with Physical Machines

1. Obtain the source code of Dragonfly.

    ```sh
    git clone https://github.com/dragonflyoss/Dragonfly.git
    ```

2. Enter the project directory.

    ```sh
    cd Dragonfly/src/supernode
    ```

3. Compile the source code.

    ```sh
    mvn clean -U install -DskipTests=true
    ```

4. Start the supernode.

    ```sh
    # If the 'supernode.baseHome’ is not specified, then the default value '/home/admin/supernode’ will be used.
    java -Dsupernode.baseHome=/home/admin/supernode -jar target/supernode.jar
    ```

5. Add the following configuration items to the Nginx configuration file.

    **Tip:** The path of the Nginx configuration file is something like `src/supernode/src/main/docker/sources/nginx.conf`.

    ```
    server {
    listen 8001;
    location / {
      # Must be ${supernode.baseHome}/repo
      root /home/admin/supernode/repo;
     }
    }

    server {
    listen 8002;
    location / {
      proxy_pass http://127.0.0.1:8080;
     }
    }
    ```

6. Start Nginx.

    ```sh
    sudo nginx
    ```

## After this Task

- After the supernode is installed, run the following commands to verify if Nginx and Tomcat are started, and if Port `8001` and `8002` are available.

    ```sh
    ps aux|grep nginx
    ps aux|grep tomcat
    telnet 127.0.0.1 8001
    telent 127.0.0.1 8002
    ```

- Install the Dragonfly client and test if the downloading works.

    ```sh
    dfget --url "http://${resourceUrl}" --output ./resource.png --node "127.0.0.1"
    ```
