---
title: "什么是 Dragonfly"
weight: 1
---

Dragonfly is an intelligent P2P based image and file distribution system. It aims to resolve issues related to low-efficiency, low-success rate and waste of network bandwidth in file transferring process.
<!--more-->

Especially in large-scale file distribution scenarios such as application distribution, cache distribution, log distribution, image distribution, etc.

At Alibaba, Dragonfly is invoked 2 billion times and the data distributed is 3.4PB every month. Dragonfly has become one of the most important pieces of infrastructure at Alibaba. The reliability is up to 99.9999%.

While container technologies makes devops life easier most of the time, it sure brings a some challenges: the efficiency of image distribution, especially when you have to replicate image distribution on several hosts. Dragonfly works extremely well with both Docker and [PouchContainer](https://github.com/alibaba/pouch) for this scenario. It also is compatible with any other container formats.

It delivers up to 57 times the throughput of native docker and saves up to 99.5% the out bandwidth of registry.

Dragonfly makes it simple and cost-effective to set up, operate, and scale any kind of files/images/data distribution.

## Why Dragonfly?

*The project is an open source version of the dragonfly and more internal features will be gradually opened*.

- **P2P based file distribution**: Using P2P technology for file transmission, which can make full use of the bandwidth resources of each peer to improve download efficiency,  saves a lot of cross-IDC bandwidth, especially costly cross-board bandwidth
- **Non-invasive support all kinds of container technologies**: Dragonfly can seamlessly support various containers for distributing images.
- **Host level speed limit**: Many downloading tools(wget/curl) only have rate limit for the current download task,but dragonfly
also provides rate limit for the entire host.
- **Passive CDN**: The CDN mechanism can avoid repetitive remote downloads.
- **Strong consistency**： Dragonfly can guarantee that all downloaded files must be consistent even if users do not provide any check code(MD5).
- **Disk protection and high efficient IO**: Precheck Disk space, delay synchronization, write file-block in the best order,
split net-read / disk-write, and so on.
- **High performance**: Cluster Manager is completely closed-loop, which means, it does not rely on any DB or distributed cache,
processing requests with extremely high performance. 
- **Exception auto isolation**: Dragonfly will automatically isolate exception nodes(peer or Cluster Manager) to improve download stability.
- **No pressure on file source**: Generally, as long as a few Cluster Managers download file from the source.
- **Support standard http header**: Support http header, Submit authentication information through http header.
- **Effective concurrency control of Registry Auth**: Reduce the pressure of the Registry Auth Service.
- **Simple and easy to use**: Very few configurations are needed.

## How does it stack up against traditional solution?

|Test Environment ||
|---|---|
|Dragonfly server|2 * (24core 64GB 2000Mb/s)|
|File Source server|2 * (24core 64GB 2000Mb/s)|
|Client|4core 8GB 200Mb/s|
|Target file size|200MB|
|Executed Date|2016-04-20|

![How it stacks up](../../../images/performance.png)

For Dragonfly, no matter how many clients issue the file downloading, the average downloading time is always around 12 seconds.
And for wget, the downloading time keeps increasing when you have more clients, and as the amount of wget clients reaches 1200, the file source will crash, then it can not serve any client.

## How does it work?

### Distributing General Files

![Distributing General Files](../../../images/dfget.png)

The cluster manager is also called supernode, which is responsible for CDN and scheduling every peer to transfer blocks between them. dfget is the client of P2P, also called 'peer',which is mainly used to download and share blocks. 

### Distributing Container Images

![Distributing Container Images](../../../images/dfget-combine-container.png)

Registry is similar to the file server above. dfget proxy is also called dfdaemon, which intercepts http-requests from docker pull or docker push,and then determines which requests need use dfget to handle.

### How file blocks are downloaded

![How file blocks are downloaded](../../../images/distributing.png)

Every File is divided into multiple blocks, which are transmitted between peers,one peer is one P2P client.
Cluster manager will judge whether the corresponding file exists in the local disk, if not, 
it will be downloaded into cluster manager from file server.
