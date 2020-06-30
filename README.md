# Dragonfly

[![License](https://img.shields.io/badge/license-Apache%202-brightgreen.svg)](https://github.com/dragonflyoss/Dragonfly/blob/master/LICENSE)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fdragonflyoss%2FDragonfly.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fdragonflyoss%2FDragonfly?ref=badge_shield)
[![GoDoc](https://godoc.org/github.com/dragonflyoss/Dragonfly?status.svg)](https://godoc.org/github.com/dragonflyoss/Dragonfly)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/2562/badge)](https://bestpractices.coreinfrastructure.org/en/projects/2562)
[![Go Report Card](https://goreportcard.com/badge/github.com/dragonflyoss/Dragonfly)](https://goreportcard.com/report/github.com/dragonflyoss/Dragonfly)
[![Build Status](https://travis-ci.org/dragonflyoss/Dragonfly.svg?branch=master)](https://travis-ci.org/dragonflyoss/Dragonfly)
[![CircleCI](https://circleci.com/gh/dragonflyoss/Dragonfly.svg?style=svg)](https://circleci.com/gh/dragonflyoss/Dragonfly)
[![codecov](https://codecov.io/gh/dragonflyoss/Dragonfly/branch/master/graph/badge.svg)](https://codecov.io/gh/dragonflyoss/Dragonfly)

![Dragonfly](docs/images/logo/dragonfly-linear.png)

> Note: The `master` branch may be in an unstable or even broken state during development. Please use [releases](https://github.com/dragonflyoss/Dragonfly/releases) instead of the `master` branch in order to get stable binaries.

## Contents

- [Introduction](#introduction)
- [Features](#features)
- [Comparison](#comparison)
- [Quick Start](./docs/quick_start/README.md)
- [Documents](https://d7y.io/en-us/docs/overview/what_is_dragonfly.html)
- [Contributing](CONTRIBUTING.md)
- [FAQ](FAQ.md)
- [Adoptors](./adopters.md)
- [LICENSE](LICENSE)

## Introduction

Dragonfly is an open source intelligent P2P based image and file distribution system. Its goal is to tackle all distribution problems in cloud native scenarios. Currently Dragonfly focuses on being:

- **Simple**: well-defined user-facing API (HTTP), non-invasive to all container engines;
- **Efficient**: CDN support, P2P based file distribution to save enterprise bandwidth;
- **Intelligent**: host level speed limit, intelligent flow control due to host detection;
- **Secure**: block transmission encryption, HTTPS connection support.

Dragonfly is now hosted by the [Cloud Native Computing Foundation](https://cncf.io) (CNCF) as an Incubating Level Project. Originally it was born to solve all kinds of distribution at very large scales, such as application distribution, cache distribution, log distribution, image distribution, and so on.

Dragonfly has finished refactoring in Golang. Now versions > 0.4.0 are totally in Golang, while those < 0.4.0 are in Java. We encourage adopters to try Golang version first, since Java versions will be out of support in the next few releases.

## Features

In details, Dragonfly has the following features:

- **P2P based file distribution**: Using P2P technology for file transmission, which can make full use of the bandwidth resources of each peer to improve download efficiency,  saves a lot of cross-IDC bandwidth, especially costly cross-board bandwidth
- **Non-invasive support for all kinds of container technologies**: Dragonfly can seamlessly support various containers for distributing images.
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

For Dragonfly, no matter how many clients start the file downloading, the average downloading time is almost stable without increasement (12s in experiment, which means it only takes 12s in total for all client to finish downloading file/image).

And for wget, the downloading time keeps increasing when you have more clients. As the number of wget clients reaches 1200 (in following experiment), the file source will crash, then it can not serve any client.

The following table shows the testing environment and the graph shows the comparison result.

|Test Environment |Statistics|
|--------------------|-------------------|
|Dragonfly server|2 * (24core 64GB 2000Mb/s)|
|File Source server|2 * (24core 64GB 2000Mb/s)|
|Client|4core 8GB 200Mb/s|
|Target file size|200MB|

![Performance](docs/images/performance.png)

## Roadmap

For more details about roadmap, please refer to file [ROADMAP.md](ROADMAP.md).

## Community

You are encouraged to communicate most things via [GitHub issues](https://github.com/dragonflyoss/Dragonfly/issues/new/choose) or pull requests.

Other active channels:

- Twitter: [@dragonfly_oss](https://twitter.com/dragonfly_oss)
- Dingtalk Group(钉钉群)

<div align="center">
  <img src="docs/images/df-dev-dingtalk.png" width="250" title="dingtalk">
</div>

## Contributing

You are warmly welcomed to hack on Dragonfly. We have prepared a detailed guide [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Dragonfly is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.
