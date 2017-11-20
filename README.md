# Dragonfly

[![Join the chat at https://gitter.im/alibaba/Dragonfly](https://badges.gitter.im/alibaba/Dragonfly.svg)](https://gitter.im/alibaba/Dragonfly?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![License](https://img.shields.io/badge/license-Apache%202-4EB1BA.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

![](https://github.com/alibaba/Dragonfly/raw/master/docs/images/logo.png)

Dragonfly is an intelligent P2P based file distribution system. It resolved issues like low-efficiency，low-success rate，waste of network bandwidth you faced in large-scale file distribution scenarios such as application deployment, large-scale cache file distribution, data file distribution, images distribution etc.
In Alibaba, the system transferred 2 billion times and distributed 3.4PB data every month, it becomes one of the most important infrastructure in Alibaba. The reliability is up to 99.9999%.


DevOps takes a lot of benefits from container technologies . but at the same time, it also bring a lot of challenges: the efficiency of image distribution, especially when you have a lot of applications and require image distribution at the same time. Dragonfly works extremely well with  both Docker and [Pouch](https://github.com/alibaba/pouch), and actually we compatible with any other container technologies without any modifications of container engine.

It delivers up to 57 times the throughput of native docker and saved up to 99.5% the out bandwidth of registry.

Dragonfly makes it simple and cost-effective to set up, operate, and scale your any kind of files/images/data distribution.

## Features
*The project is an open source version of the dragonfly and more internal features will be gradually opened*.

- **P2P based file distribution**: Using P2P technology for file transmission, which can make full 
use of the bandwidth resources of each peer to improve download efficiency.  saved lot of cross-IDC bandwidth, especially costly cross-board bandwidth
- **Non-invasive support all kinds of container technologies**: Dragonfly can seamlessly support various containers for distributing images.
- **Host level speed limit**: Many download tools(wget/curl) only have rate limit for the current download task,but dragonfly
still provides rate limit for the entire host.
- **Passive CDN**: The CDN mechanism can avoid repetitive remote downloads.
- **Strong consistency**： Dragonfly can guarantee that all downloaded files must be consistent even if users do not provide any check code(MD5).
- **Disk protection and high efficient IO**: Pre check Disk space, delay synchronization, write file-block in the best order,
split net-read / disk-write, and so on.
- **High performance**: Cluster Manager is completely closed-loop.that is, it does not rely on any DB and distributed cache,
processing requests with extremely high performance. 
- **Exception auto isolation**: Dragonfly will automatically isolate exception nodes(peer or Cluster Manager) to improve download stability.
- **No pressure on file source**: Generally, as long as a few Cluster Managers download file from the source.
- **Support standard http header**: Support http header, Submit authentication information through http header.
- **Effective concurrency control of Registry Auth**: Reduce the pressure of the Registry Auth Service.
- **Simple and easy to use**: Very few configurations are needed.

## Performance Benchmark (wget v.s dragonfly)

|Test Environment ||
|--------------------|-------------------|
|Dragonfly server|2 * (24core 64GB 2000Mb/s)|
|File Source server|2 * (24core 64GB 2000Mb/s)|
|Client|4core 8GB 200Mb/s|
|Target file size|200MB|
|Executed Date|2016-04-20|

**Results:**

<div>
<img src="https://github.com/alibaba/Dragonfly/raw/master/docs/images/performance.png"/>
</div>

For Dragonfly the average time of downloading is around 12 seconds no matter how many clients issued the file downloading.
and for wget time increased when you have more clients. and by 1200 clients, the file source crash, it can not serve any client.

## System Architecture

&nbsp;&nbsp;&nbsp;&nbsp;Please Read [Architecture Introduction.](https://github.com/alibaba/Dragonfly/blob/master/docs/architecture_introduction.md)

## Install & Run

1. [Install and run server.](https://github.com/alibaba/Dragonfly/blob/master/docs/install_clustermanager.md)
*you need record server ips, which will be used later*.

2. [Install client.](https://github.com/alibaba/Dragonfly/blob/master/docs/install_client.md)

3. [Using dragonfly.](https://github.com/alibaba/Dragonfly/blob/master/docs/configuration_and_run.md)

## License

Dragonfly is available under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0.html).

## Commercial Support

If you need commercial support of Dragonfly, please contact us for more information: [云效](https://www.aliyun.com/product/yunxiao).

Dragonfly is already integrated with AliCloud Container Services
If you need commercial support of AliCloud Container Service, please contact us for more information: [Container Service
](https://www.alibabacloud.com/product/container-service)
