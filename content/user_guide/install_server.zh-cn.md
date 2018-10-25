---
title: "安装服务端"
weight: 1
---

本文介绍了如何安装 Dragonfly 服务端。
<!--more-->

{{% notice tip %}}
对一个机房或集群而言，建议至少准备 2 台 8 核、16G 内存、千兆网络的机器，用于部署超级节点。
{{% /notice %}}

## 背景信息

Dragonfly 的系统架构包含两层：超级节点（服务端）和主机（客户端）。您可以选择以下方法之一来安装超级节点。

- Docker 部署：适用于在本地快速部署和测试。
- 物理机部署：适用于生产环境。

## 前提条件

如果采用 Docker 部署方式安装超级节点，必须满足以下前提条件。

必需软件|版本要求
---|---
Git|1.9.1+
Docker|1.12.0+

如果采用物理机部署方式安装超级节点，必须满足以下前提条件。

必需软件|版本要求
---|---
Git|1.9.1+
JDK|1.7+
Maven|3.0.3+
Nginx|0.8+

## 操作步骤 - Docker 部署方式

1. 获取 Dragonfly 源代码。

    ```sh
    git clone https://github.com/alibaba/Dragonfly.git
    ```
2. 进入项目目录。

    ```sh
    cd Dragonfly
    ```
3. 构建 Docker 镜像。

    ```sh
    ./build/build.sh supernode
    ```
4. 获取最新的超级节点 Docker 镜像 ID。

    ```sh
    docker image ls|grep 'supernode' |awk '{print $3}' | head -n1
    ```
5. 启动超级节点。

    ```sh
    docker run -d -p 8001:8001 -p 8002:8002 ${superNodeDockerImageId}
    ```

## 操作步骤 - 物理机部署方式

1. 获取 Dragonfly 源代码。

    ```sh
    git clone https://github.com/alibaba/Dragonfly.git
    ```
2. 进入项目目录。

    ```sh
    cd Dragonfly/src/supernode
    ```
3. 编译源代码。

    ```sh
    mvn clean -U install -DskipTests=true
    ```
4. 启动 supernode 服务。

    ```sh
    # 如果不指定 'supernode.baseHome'，则使用默认值 '/home/admin/supernode'。
    ```
5. 在 Nginx 配置文件中添加以下配置。

    **提示：**Nginx 配置文件路径例如 `src/supernode/src/main/docker/sources/nginx.conf`。

    ```
    server {
    listen 8001;
    location / {
      # 必须是 ${supernode.baseHome}/repo
      root /home/admin/supernode/repo;
     }
    }

    server {
    listen 8002;
    location /peer {
      proxy_pass http://127.0.0.1:8080;
     }
    }
    ```
	
6. 启动 Nginx。

    ```sh
    sudo nginx
    ```

## 后续步骤

- 安装完超级节点后，可通过以下命令验证 Nginx 和 Tomcat 是否已启动，以及端口 `8001`、`8002` 是否可用。

    ```sh
    ps aux|grep nginx
    ps aux|grep tomcat
    telnet 127.0.0.1 8001
    telent 127.0.0.1 8002
    ```

- 安装 Dragonfly 客户端并测试能否下载资源。

    ```sh
    dfget --url "http://${resourceUrl}" --output ./resource.png --node "127.0.0.1"
    ```
