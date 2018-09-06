# Dragonfly

[![Join the chat at https://gitter.im/alibaba/Dragonfly](https://badges.gitter.im/alibaba/Dragonfly.svg)](https://gitter.im/alibaba/Dragonfly?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![License](https://img.shields.io/badge/license-Apache%202-brightgreen.svg)](https://github.com/alibaba/Dragonfly/blob/master/LICENSE)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Falibaba%2FDragonfly.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Falibaba%2FDragonfly?ref=badge_shield)
[![Build Status](https://travis-ci.org/alibaba/Dragonfly.svg?branch=master)](https://travis-ci.org/alibaba/Dragonfly)

## ![Dragonfly](./images/logo.png)

## Contents

- [Introduction](#introduction)
- [Features](#features)
- [Comparison](#comparison) 
- [Quick Start](./en/quick_start.md)
- [Documents](./documents.md)
- [Contributing](./en/CONTRIBUTING.md)
- [Adoptors](./en/adoptors.md)
- [LICENSE](./en/LICENSE)
- [Commercial Support](#commercial-support)

## Introduction

Dragonfly is an intelligent P2P based image and file distribution system. It aims to resolve issues related to low-efficiency, low-success rate and waste of network bandwidth in file transferring process. Especially in large-scale file distribution scenarios such as application distribution, cache distribution, log distribution, image distribution, etc.
In Alibaba, Dragonfly is invoiked 2 Billion times and the data distributed is 3.4PB every month. Dragonfly has become one of the most important pieces of infrastructure at Alibaba. The reliability is up to 99.9999%.


While container technologies makes devops life easier most of the time, it sure brings a some challenges: the efficiency of image distribution, especially when you have to replicate image distribution on several hosts. Dragonfly works extremely well with both Docker and [PouchContainer](https://github.com/alibaba/pouch) for this scenario. It also is compatible with any other container formats.

It delivers up to 57 times the throughput of native docker and saves up to 99.5% the out bandwidth of registry.

Dragonfly makes it simple and cost-effective to set up, operate, and scale any kind of files/images/data distribution.

## Features
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

## Comparison

|Test Environment ||
|--------------------|-------------------|
|Dragonfly server|2 * (24core 64GB 2000Mb/s)|
|File Source server|2 * (24core 64GB 2000Mb/s)|
|Client|4core 8GB 200Mb/s|
|Target file size|200MB|
|Executed Date|2016-04-20|

<div>
<img src="./images/performance.png"/>
</div>

For Dragonfly, no matter how many clients issue the file downloading, the average downloading time is always around 12 seconds.
And for wget, the downloading time keeps increasing when you have more clients, and as the amount of wget clients reaches 1200, the file source will crash, then it can not serve any client.

## License

Dragonfly is available under the [Apache 2.0 License](https://github.com/alibaba/Dragonfly/blob/master/LICENSE).

## Commercial Support

If you need commercial support of Dragonfly, please contact us for more information: [云效](https://www.aliyun.com/product/yunxiao).

Dragonfly is already integrated with AliCloud Container Services
If you need commercial support of AliCloud Container Service, please contact us for more information: [Container Service
](https://www.alibabacloud.com/product/container-service)
