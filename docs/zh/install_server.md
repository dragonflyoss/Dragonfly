# 安装服务端

本指南给开发者提供了从源码构建和运行超级节点的操作命令。建议在同一机房或集群内至少有2台8核16G且千兆网络的机器用于部署超级节点。

本文中提供了两种方式来安装超级节点:
* Docker部署： 本地快速部署测试。
* 物理机部署：建议生产环境使用。

## 1. 环境配置

### 1.1. Docker部署

软件                  | 版本要求
----------------------|--------------------------
Git                   | 1.9.1 +
Docker                | 1.12.0 +

### 1.2. 物理机部署

软件                  | 版本要求
----------------------|--------------------------
Git                   | 1.9.1 +
Jdk                   | 1.7 +
Maven                 | 3.0.3 +
Tomcat                | 7.0 +
Nginx                 | 0.8 +

## 2. 获取蜻蜓源码
```sh
git clone https://github.com/alibaba/Dragonfly.git
```

## 3. 构建与执行
### 3.1. Docker方式
* 进入项目目录

  ```sh
  cd dragonfly
  ```
* 构建Docker镜像

  - 构建镜像

    ```sh
    docker image build -t "dragonfly:supernode" . -f ./build/supernode/Dockerfile
    ```

  - 获取超级节点Docker镜像Id

    ```sh
    docker image ls|grep -E 'dragonfly.*supernode'|awk '{print $3}'
    ```
* 启动超级节点

   ```sh
   docker run -d -p 8001:8001 -p 8002:8002 ${superNodeDockerImageId}
   ```

### 3.2. 物理机部署
* 进入项目目录

  ```sh
  cd dragonfly/src/supernode
  ```
* 编译源码

  ```sh
  mvn clean -U install -DskipTests=true
  ```
* 将服务部署到tomcat

  - 将war包拷贝到tomcat目录

    ```sh
    copy target/supernode.war ${CATALINA_HOME}/webapps/supernode.war
    ```
  - 更改tomcat的context配置

    将下列配置添加到tomcat的`server.xml`中:

    ```xml
    <Context path="/" docBase="${CATALINA_HOME}/webapps/supernode" debug="0" reloadable="true" crossContext="true" />
    ```
  - 启动tomcat

    ```sh
    ./${CATALINA_HOME}/bin/catalina.sh run
    ```
* 启动Nginx

  - 添加下列nginx配置

    ```
    server {
      listen              8001;
      location / {
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

    > nginx配置例子: _build/supernode/docker/nginx/nginx.conf_

  - 启动nginx

    ```sh
    sudo nginx
    ```

## 4. 测试验证
* 检测nginx和tomcat是否启动，端口`8001`，`8002`是否可用

  ```sh
  ps aux|grep nginx
  ps aux|grep tomcat
  telnet 127.0.0.1 8001
  telent 127.0.0.1 8002
  ```

* 安装蜻蜓客户端并使用蜻蜓下载资源

  ```sh
  dfget --url "http://${resourceUrl}" --output ./resource.png --node "127.0.0.1"
  ```
