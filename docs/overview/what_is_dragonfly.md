---
title: "What Is Dragonfly?"
weight: 1
---

Dragonfly is an intelligent P2P-based image and file distribution tool. It aims to improve the efficiency and success rate of file transferring, and maximize the usage of network bandwidth, especially for the distribution of larget amounts of data, such as application distribution, cache distribution, log distribution, and image distribution.
<!--more-->

At Alibaba, every month Dragonfly is invoked two billion times and distributes 3.4PB of data. Dragonfly has become one of the most important pieces of infrastructure at Alibaba.

While container technologies makes DevOps life easier most of the time, it surely brings some challenges: for example the efficiency of image distribution, especially when you have to replicate image distribution on several hosts.

Dragonfly works extremely well with both Docker and [PouchContainer](https://github.com/alibaba/pouch) in this scenario. It's also compatible with containers of other formats. It delivers up to 57 times the throughput of native docker and saves up to 99.5% of the out bandwidth of registry.

Dragonfly makes it simple and cost-effective to set up, operate, and scale any kind of file, image, or data distribution.

## Why Dragonfly

This project is an open-source version of the Dragonfly used at Alibaba. It has the following features:

**Note:** More Alibaba-internal features will be made available to open-source users soon. Stay tuned!

- **P2P-based file distribution**: By using the P2P technology for file transmission, it makes the most out of the bandwidth resources of each peer to improve downloading efficiency,  and saves a lot of cross-IDC bandwidth, especially the costly cross-board bandwidth.
- **Non-invasive support to all kinds of container technologies**: Dragonfly can seamlessly support various containers for distributing images.
- **Host level speed limit**: In addition to rate limit for the current download task like many other downloading tools (for example wget and curl), Dragonfly also provides rate limit for the entire host.
- **Passive CDN**: The CDN mechanism can avoid repetitive remote downloads.
- **Strong consistency**: Dragonfly can make sure that all downloaded files are consistent even if users do not provide any check code (MD5).
- **Disk protection and highly efficient IO**: Prechecking disk space, delaying synchronization, writing file blocks in the best order, isolating net-read/disk-write, and so on.
- **High performance**: Cluster Manager is completely closed-loop, which means that it doesn't rely on any database or distributed cache, processing requests with extremely high performance.
- **Auto-isolation of Exception**: Dragonfly will automatically isolate exception nodes (peer or Cluster Manager) to improve download stability.
- **No pressure on file source**: Generally, only a few Cluster Managers will download files from the source.
- **Support standard HTTP header**: Support submitting authentication information through HTTP header.
- **Effective concurrency control of Registry Auth**: Reduce the pressure on the Registry Auth Service.
- **Simple and easy to use**: Very few configurations are needed.

## How Does It Stack Up Against Traditional Solution?

We carried out an experiment to compare the performance of Dragonfly and wget.

|Test Environment ||
|---|---|
|Dragonfly Server|2 * (24-Core 64GB-RAM 2000Mb/s)|
|File Source Server|2 * (24-Core 64GB-RAM 2000Mb/s)|
|Client|4-Core 8GB-RAM 200Mb/s|
|Target File Size|200MB|
|Experiment Date|April 20, 2016|

The expeirment result is as shown in the following figure.

![How it stacks up](../images/performance.png)

As you can see in the chart, for Dragonfly, no matter how many clients are downloading, the average downloading time is always about 12 seconds. But for wget, the downloading time keeps increasing with the number of clients. When the number of wget clients reaches 1,200, the file source crashed and therefore cannot serve any client.

## How Does It Work?

Dragonfly works slightly differently when downloading general files and downloading container images.

### Downloading General Files

The Cluster Manager is also called a supernode, which is responsible for CDN and scheduling every peer to transfer blocks between each other. dfget is the P2P client, which is also called "peer". It's mainly used to download and share blocks.

![Downloading General Files](../images/dfget.png)

### Downloading Container Images

Registry is similar to the file server above. dfget proxy is also called dfdaemon, which intercepts HTTP requests from docker pull or docker push, and then decides which requests to process with dfget.

![Downloading Container Images](../images/dfget-combine-container.png)

### Downloading Blocks

Every file is divided into multiple blocks, which are transmitted between peers. Each peer is a P2P client. Cluster Manager will check if the corresponding file exists in the local disk. If not, it will be downloaded into Cluster Manager from file server.

![How file blocks are downloaded](../images/distributing.png)
