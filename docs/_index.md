---
title: "Dragonfly"
---

[![Join the chat at https://gitter.im/alibaba/Dragonfly](https://badges.gitter.im/alibaba/Dragonfly.svg)](https://gitter.im/alibaba/Dragonfly?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![License](https://img.shields.io/badge/license-Apache%202-brightgreen.svg)](https://github.com/dragonflyoss/Dragonfly/blob/master/LICENSE)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Falibaba%2FDragonfly.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Falibaba%2FDragonfly?ref=badge_shield)
[![GoDoc](https://godoc.org/github.com/dragonflyoss/Dragonfly?status.svg)](https://godoc.org/github.com/dragonflyoss/Dragonfly)
[![Go Report Card](https://goreportcard.com/badge/github.com/dragonflyoss/Dragonfly)](https://goreportcard.com/report/github.com/dragonflyoss/Dragonfly)
[![Build Status](https://travis-ci.org/dragonflyoss/Dragonfly.svg?branch=master)](https://travis-ci.org/dragonflyoss/Dragonfly)
[![CircleCI](https://circleci.com/gh/dragonflyoss/Dragonfly.svg?style=svg)](https://circleci.com/gh/dragonflyoss/Dragonfly)
[![codecov](https://codecov.io/gh/dragonflyoss/Dragonfly/branch/master/graph/badge.svg)](https://codecov.io/gh/dragonflyoss/Dragonfly)

![Dragonfly](images/logo.png)

Dragonfly is an intelligent P2P-based image and file distribution tool. It aims to improve the efficiency and success rate of file transferring, and maximize the usage of network bandwidth, especially for the distribution of larget amounts of data, such as application distribution, cache distribution, log distribution, and image distribution.
<!--more-->

At Alibaba, every month Dragonfly is invoked two billion times and distributes 3.4PB of data. Dragonfly has become one of the most important pieces of infrastructure at Alibaba.

While container technologies makes DevOps life easier most of the time, it surely brings some challenges: for example the efficiency of image distribution, especially when you have to replicate image distribution on several hosts.

Dragonfly works extremely well with both Docker and [PouchContainer](https://github.com/dragonflyoss/pouch) in this scenario. It's also compatible with containers of other formats. It delivers up to 57 times the throughput of native docker and saves up to 99.5% of the out bandwidth of registry.

Dragonfly makes it simple and cost-effective to set up, operate, and scale any kind of file, image, or data distribution.

## Why Dragonfly?

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

![How it stacks up](images/performance.png)

As you can see in the chart, for Dragonfly, no matter how many clients are downloading, the average downloading time is always about 12 seconds. But for wget, the downloading time keeps increasing with the number of clients. When the number of wget clients reaches 1,200, the file source crashed and therefore cannot serve any client.

## How Does It Work?

Dragonfly works slightly differently when downloading general files and downloading container images.

### Downloading General Files

The Cluster Manager is also called a supernode, which is responsible for CDN and scheduling every peer to transfer blocks between each other. dfget is the P2P client, which is also called "peer". It's mainly used to download and share blocks.

![Downloading General Files](images/dfget.png)

### Downloading Container Images

Registry is similar to the file server above. dfget proxy is also called dfdaemon, which intercepts HTTP requests from docker pull or docker push, and then decides which requests to process with dfget.

![Downloading Container Images](images/dfget-combine-container.png)

### Downloading Blocks

Every file is divided into multiple blocks, which are transmitted between peers. Each peer is a P2P client. Cluster Manager will check if the corresponding file exists in the local disk. If not, it will be downloaded into Cluster Manager from file server.

![How file blocks are downloaded](images/distributing.png)

## Who has adopted Dragonfly?

Here are some of the adoptors of project Dragonfly. If you are using Dragonfly to improve your distribution, please don't hesitate to reach out to us and show your support.

<a href="https://www.alibabagroup.com" border="0" target="_blank"><img alt="trendmicro" src="images/adoptor_logo/AlibabaGroup.jpg" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;
<a href="https://www.alibabacloud.com/zh" border="0" target="_blank"><img alt="trendmicro" src="images/adoptor_logo/AlibabaCloud.png" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;
<a href="http://www.10086.cn/" border="0" target="_blank"><img alt="OnStar" src="images/adoptor_logo/ChinaMobile.png" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;
<a href="https://www.antfin.com/" border="0" target="_blank"><img alt="OnStar" src="images/adoptor_logo/AntFinancial.png" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;
<a href="https://www.cainiao.com/" border="0" target="_blank"><img alt="OnStar" src="images/adoptor_logo/CaiNiao.gif" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;
<a href="http://www.iflytek.com/" border="0" target="_blank"><img alt="OnStar" src="images/adoptor_logo/iFLYTEK.jpeg" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;
<a href="https://www.didiglobal.com" border="0" target="_blank"><img alt="OnStar" src="images/adoptor_logo/didi.png" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;
<a href="https://www.meituan.com" border="0" target="_blank"><img alt="OnStar" src="images/adoptor_logo/meituan.png" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;
<a href="https://www.amap.com/" border="0" target="_blank"><img alt="OnStar" src="images/adoptor_logo/amap.png" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;
<a href="https://www.lazada.com/" border="0" target="_blank"><img alt="OnStar" src="images/adoptor_logo/lazada.png" height="50"></a>&nbsp; &nbsp; &nbsp; &nbsp;

## Community

You are encouraged to communicate with us via GitHub issues or pull requests.

Follow us on other platforms:

- Gitter Chat: [dragonfly](https://gitter.im/alibaba/Dragonfly)
- Twitter: [@dragonfly_oss](https://twitter.com/dragonfly_oss)

## License

Dragonfly is available under the [Apache 2.0 License](https://github.com/dragonflyoss/Dragonfly/blob/master/LICENSE).
