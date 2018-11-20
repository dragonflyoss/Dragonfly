# Dragonfly

[![Join the chat at https://gitter.im/alibaba/Dragonfly](https://badges.gitter.im/alibaba/Dragonfly.svg)](https://gitter.im/alibaba/Dragonfly?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![License](https://img.shields.io/badge/license-Apache%202-brightgreen.svg)](https://github.com/dragonflyoss/Dragonfly/blob/master/LICENSE)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Falibaba%2FDragonfly.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Falibaba%2FDragonfly?ref=badge_shield)
[![GoDoc](https://godoc.org/github.com/dragonflyoss/Dragonfly?status.svg)](https://godoc.org/github.com/dragonflyoss/Dragonfly)
[![Go Report Card](https://goreportcard.com/badge/github.com/dragonflyoss/Dragonfly)](https://goreportcard.com/report/github.com/dragonflyoss/Dragonfly)
[![Build Status](https://travis-ci.org/dragonflyoss/Dragonfly.svg?branch=master)](https://travis-ci.org/dragonflyoss/Dragonfly)
[![CircleCI](https://circleci.com/gh/dragonflyoss/Dragonfly.svg?style=svg)](https://circleci.com/gh/dragonflyoss/Dragonfly)
[![codecov](https://codecov.io/gh/dragonflyoss/Dragonfly/branch/master/graph/badge.svg)](https://codecov.io/gh/dragonflyoss/Dragonfly)

![Dragonfly](docs/images/logo/dragonfly-linear.png)

## Contents

- [Introduction](#introduction)
- [Features](#features)
- [Comparison](#comparison)
- [Quick Start](./docs/quick_start/_index.md)
- [Documents](https://d7y.io/)
- [Contributing](CONTRIBUTING.md)
- [FAQ](FAQ.md)
- [Adoptors](./docs/_index.md#who-has-adopted-dragonfly)
- [LICENSE](LICENSE)

## Introduction

Dragonfly is an intelligent P2P based image and file distribution system. It aims to resolve issues related to low-efficiency, low-success rate and waste of network bandwidth in file transferring process. Especially in large-scale file distribution scenarios such as application distribution, cache distribution, log distribution, image distribution, etc.
In Alibaba, Dragonfly is invoked 2 Billion times and the data distributed is 3.4PB every month. Dragonfly has become one of the most important pieces of infrastructure at Alibaba. The reliability is up to 99.9999% (*1).

While container technologies makes devops life easier most of the time, it sure brings a some challenges: the efficiency of image distribution, especially when you have to replicate image distribution on several hosts. Dragonfly works extremely well with both Docker and [PouchContainer](https://github.com/alibaba/pouch) for this scenario. It also is compatible with any other container formats.

It delivers up to 57 times the throughput of native docker and saves up to 99.5% the out bandwidth of registry(*2).

Dragonfly makes it simple and cost-effective to set up, operate, and scale any kind of files/images/data distribution.

## Features

*The project is an open source version of the dragonfly and more internal features will be gradually opened*.

- **P2P based file distribution**: Using P2P technology for file transmission, which can make full use of the bandwidth resources of each peer to improve download efficiency,  saves a lot of cross-IDC bandwidth, especially costly cross-board bandwidth
- **Non-invasive support all kinds of container technologies**: Dragonfly can seamlessly support various containers for distributing images.
- **Host level speed limit**: Many downloading tools(wget/curl) only have rate limit for the current download task, but dragonfly also provides rate limit for the entire host.
- **Passive CDN**: The CDN mechanism can avoid repetitive remote downloads.
- **Strong consistency**: Dragonfly can guarantee that all downloaded files must be consistent even if users do not provide any check code(MD5).
- **Disk protection and high efficient IO**: Precheck Disk space, delay synchronization, write file-block in the best order, split net-read / disk-write, and so on.
- **High performance**: Cluster Manager is completely closed-loop, which means, it does not rely on any DB or distributed cache, processing requests with extremely high performance.
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

![Performance](docs/images/performance.png)

For Dragonfly, no matter how many clients issue the file downloading, the average downloading time is always around 12 seconds.
And for wget, the downloading time keeps increasing when you have more clients, and as the amount of wget clients reaches 1200, the file source will crash, then it can not serve any client.

## Roadmap

For more details about roadmap, please refer to file [ROADMAP.md](ROADMAP.md).

## Community

You are encouraged to communicate most things via GitHub issues or pull requests.

Other active channels:

- Gitter Chat: [dragonfly](https://gitter.im/alibaba/Dragonfly)
- Twitter: [@dragonfly_oss](https://twitter.com/dragonfly_oss)

## Contributing

You are warmly welcomed to hack on Dragonfly. We have prepared a detailed guide [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Dragonfly is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.
