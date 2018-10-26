---
title: "Installing Server"
weight: 1
---

This guide provides instructions for developers to build and run cluster manager (supernode) from source code. 
<!--more-->

{{% notice tip %}}
The recommended deployment for the cluster manager is that at least two machines with at least 8-core 16G and best to provide Gigabit Ethernet.
{{% /notice %}}

## Step 1: Requirements
You can either deploy the cluster manager (supernode) on the Docker container or on the physical machine.

### 1. Deployed on the Docker container

Software              | Required Version
----------------------|--------------------------
Git                   | 1.9.1 +
Docker                | 1.12.0 +

### 2. Deployed on the physical machine

Software              | Required Version
----------------------|--------------------------
Git                   | 1.9.1 +
Jdk                   | 1.7 +
Maven                 | 3.0.3 +
Nginx                 | 0.8 +

## Step2: Getting the source code
   ```
   $ git clone https://github.com/alibaba/Dragonfly.git
   ```

## Step3：Build and run
### 1. Run on Docker
* Enter the project directory

   ```
   $ cd Dragonfly
   ```
* Build Docker image

   - Build image

   ```bash
   ./build/build.sh supernode
   ```
   - Show Docker images

   ```bash
   docker image ls
   ```
   - Get latest `supernode` Docker imageId

   ```bash
   docker image ls | grep supernode |awk '{print $3}' | head -n1
   ```
* Start Docker container

   ```
   $ docker run -d -p 8001:8001 -p 8002:8002 ${superNodeDockerImageId}
   ```

### 2. Run on physical machine
* Enter the project directory

   ```bash
   cd dragonfly/src/supernode
   ```
* Build the source code

   ```bash
   mvn clean -U install -DskipTests=true
   ```
* Start `supernode` server

   - just start it with this command

   ```bash
   # the default value of 'supernode.baseHome' is '/home/admin/supernode' if you don't set
   java -Dsupernode.baseHome=/home/admin/supernode -jar target/supernode.jar
   ```
* Start nginx

  - Add nginx config

  ```
  server {
        listen              8001;
        location / {
            # must be ${supernode.baseHome}/repo
            root /home/admin/supernode/repo;
        }
    }

    server {
        listen              8002;
        location /peer {
            proxy_pass   http://127.0.0.1:8080;
        }
    }
  ```
  - Example of nginx config

  ```bash
  less src/supernode/src/main/docker/sources/nginx.conf
  ```
  - Start nginx

  ```
  $ sudo nginx
  ```

  ## Step4: Verify installation
  - Check if nginx and tomcat is started and port (8001,8002) is available.

  ```
  $ ps aux|grep nginx
  $ ps aux|grep tomcat
  $ telnet 127.0.0.1 8001
  $ telent 127.0.0.1 8002
  ```
  - Install dragonfly client and use dragonfly client to download resource through dragonfly.

  ```
  $ dfget --url "http://${resourceUrl}" --output ./resource.png --node "127.0.0.1"
  ```
