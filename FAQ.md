Frequently Asked Questions
---

## What is Dragonfly
**Dragonfly is an intelligent P2P based image and file distribution system.**

It aims to resolve issues related to low-efficiency, low-success rate and waste of network bandwidth in file transferring process. Especially in large-scale file distribution scenarios such as application distribution, cache distribution, log distribution, image distribution, etc.

In Alibaba, Dragonfly is invoked 2 Billion times and the data distributed is 3.4PB every month. Dragonfly has become one of the most important pieces of infrastructure at Alibaba.

While container technologies makes devops life easier most of the time, it sure brings a some challenges: the efficiency of image distribution, especially when you have to replicate image distribution on several hosts. Dragonfly works extremely well with both Docker and [PouchContainer](https://github.com/alibaba/pouch) for this scenario. It also is compatible with any other container formats.

It delivers up to 57 times the throughput of native docker and saves up to 99.5% the out bandwidth of registry(*2).

Dragonfly makes it simple and cost-effective to set up, operate, and scale any kind of files/images/data distribution.

## How can I pull images by Dragonfly

See [Use Dragonfly to Pull an Image](docs/en/quick_start.md#use-dragonfly-to-pull-an-image)

## How can I download files by Dragonfly

See [Use Dragonfly to Download a File](docs/en/quick_start.md#use-dragonfly-to-download-a-file)

## What is SuperNode

SuperNode is a long-time process with two primary responsibilities:
* It's the tracker and scheduler in the P2P network that choose appropriate downloading net-path for each peer. 
* It's also a CDN server that caches downloaded data from source to avoid downloading same files repeatedly.

## What is dfget

Dfget is the client of Dragonfly used for downloading files. It's similar to using wget.

At the same time, it also plays the role of peer, which can transfer data between each other in p2p network.

## What is dfdaemon

Dfdaemon is only used for pulling images. It establishes a proxy between dockerd/pouchd and registry.

Dfdaemon filters out layer fetching requests from all requests send by dockerd/pouchd when pulling images, then it uses dfget to downloading these layers.
