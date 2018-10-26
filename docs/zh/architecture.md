# 蜻蜓架构介绍

## 分发普通文件

<div align="center"><img src="../images/dfget.png"/></div>

> 注: 其中`cluster manager`即超级节点(supernode)

超级节点充当CDN，同时调度每个对等者(peer)在他们之间传输文件块。`dfget`是P2P客户端，也称为对等者(peer)，主要用于下载和共享文件块。

## 分发容器镜像

<div align="center"><img src="../images/dfget-combine-container.png"/></div>

图中镜像仓库(registry)类似于文件服务器。`dfget proxy`也称为`dfdaemon`，它拦截来自`docker pull`和`docker push`的HTTP请求，然后将那些跟镜像分层相关的请求使用`dfget`来处理。

## 文件分块是怎么下载的

<div align="center"><img src="../images/distributing.png"/></div>

> 注: 其中`cluster manager`即超级节点(supernode)

每个文件会被分成多个块在对等者(peer)间进行传输。一个peer就是一个P2P客户端。

超级节点会判断文件是否存在本地，如果不存在，则会将其从文件服务器下载到本地。
