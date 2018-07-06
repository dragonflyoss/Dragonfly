# Dragonfly

[![Join the chat at https://gitter.im/alibaba/Dragonfly](https://badges.gitter.im/alibaba/Dragonfly.svg)](https://gitter.im/alibaba/Dragonfly?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![License](https://img.shields.io/badge/license-Apache%202-brightgreen.svg)](https://github.com/alibaba/Dragonfly/blob/master/LICENSE)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Falibaba%2FDragonfly.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Falibaba%2FDragonfly?ref=badge_shield)
[![GoDoc](https://godoc.org/github.com/alibaba/Dragonfly?status.svg)](https://godoc.org/github.com/alibaba/Dragonfly)
[![Go Report Card](https://goreportcard.com/badge/github.com/alibaba/Dragonfly)](https://goreportcard.com/report/github.com/alibaba/Dragonfly)
[![Build Status](https://travis-ci.org/alibaba/Dragonfly.svg?branch=master)](https://travis-ci.org/alibaba/Dragonfly)
[![CircleCI](https://circleci.com/gh/alibaba/Dragonfly.svg?style=svg)](https://circleci.com/gh/alibaba/Dragonfly)
[![codecov](https://codecov.io/gh/alibaba/Dragonfly/branch/master/graph/badge.svg)](https://codecov.io/gh/alibaba/Dragonfly)


## ![Dragonfly](docs/images/logo.png)

## Contents

- [Introduction](#introduction)
- [Features](#features)
- [Comparison](#comparison) 
- [Documents](./docs/documents.md)
- [Contributing](CONTRIBUTING.md)
- [FAQ](FAQ.md)
- [LICENSE](LICENSE)
- [Commercial Support](#commercial-support)

## Introduction

Dragonfly is an intelligent P2P based file distribution system. It resolves issues like low-efficiency, low-success rate and waste of network bandwidth in large-scale file distribution scenarios such as application deployment, large-scale cache file distribution, data file distribution, image distribution etc.
At Alibaba, the system transfers 2 billion times and distributes 3.4PB of data every month, it has become one of the most important piece of infrastructure at Alibaba. The reliability is up to 99.9999%.


DevOps takes a lot of benefits from container technologies, but at the same time, it also brings a lot of challenges: the efficiency of image distribution, especially when you have a lot of applications and require image distribution at the same time. Dragonfly works extremely well with  both Docker and [PouchContainer](https://github.com/alibaba/pouch), and actually we are compatible with any other container technologies without any modifications of container engine.

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
<img src="docs/images/performance.png"/>
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
