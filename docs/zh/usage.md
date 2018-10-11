# 使用指南

## 环境要求

* Linux
* python 2.7 并且在`PATH`环境变量中

## 操作步骤

### 配置

#### 通过配置文件或命令行参数指定超级节点
> 确保超级节点已经启动，部署参考：*[超级节点部署指南](./install_server.md)*

* 配置文件方式：用于分发容器镜像和普通文件
  ```sh
  # 编辑蜻蜓配置文件
  vi /etc/dragonfly.conf
  # 在配置文件中添加超级节点信息，多个节点用','分割
  [node]
  address=nodeIp1,nodeIp2
  ```

* 命令行参数方式：适用于下载普通文件

  在每次执行`dfget`命令时添加参数`--node`，如下：
  ```sh
  dfget -u "http://www.taobao.com" -o /tmp/test.html --node nodeIp1,nodeIp2
  ```

  注意：**命令行参数会覆盖掉配置文件内容**

#### 配置 Container Daemon

> 如果仅用蜻蜓下载普通文件，则忽略此步骤。**若用于镜像下载，则此步骤必须。**

* 启动`dfget proxy`(即`df-daemon`)
  ```sh
  # 查看帮助信息
  df-deaemon -h
  # 启动`df-daemon`，指定镜像仓库地址，默认端口为`65001`
  df-daemon --registry https://xxx.xx.x
  # 查看`df-daemon`日志
  tailf ~/.small-dragonfly/logs/dfdaemon.log
  ```

* 配置 Daemon Mirror

  _如下是配置Docker Daemon Mirror的标准方法_
  ```sh
  # 1. 编辑`/etc/docker/daemon.json`
  vi /etc/docker/daemon.json
  # 2. 在配置文件里添加或更新配置项`registry-mirrors`
  "registry-mirrors": ["http://127.0.0.1:65001"]
  # 3. 重启docker deamon
  systemctl restart docker
  ```
  > 关于`/etc/docker/daemon.json`，详情参考[官网文档](https://docs.docker.com/registry/recipes/mirror/#configure-the-cache)

### 运行

#### 分发普通文件

```sh
# 查看`dfget`帮助信息
dfget -h
# 下载文件，默认使用`/etc/dragonfly.conf`配置
dfget --url "http://xxx.xx.x"
# 下载文件，指定超级节点
dfget --url "http://xxx.xx.x" --node "127.0.0.1"
# 下载文件，指定输出文件
dfget --url "http://xxx.xx.x" -o a.txt
# 查看下载日志
less ~/.small-dragonfly/logs/dfclient.log
```

#### 分发docker镜像

直接使用`docker pull imageName`下载镜像即可。

> **注意**：镜像名称不要包含镜像仓库地址，因为仓库域名已经由`df-daemon`的启动参数`--registry`指定。
