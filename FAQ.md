# Frequently Asked Questions (FAQ)

FAQ contains the frequently asked questions about two aspects:

- First, user-facing functionalities.
- Second, underlying concept and thoery.

Techinical question will not be included in FAQ.

## What is Dragonfly

**Dragonfly is an intelligent P2P based image and file distribution system.**

It aims to resolve issues related to low-efficiency, low-success rate and waste of network bandwidth in file transferring process. Especially in large-scale file distribution scenarios such as application distribution, cache distribution, log distribution, image distribution, etc.

In Alibaba, Dragonfly is invoked 2 Billion times and the data distributed is 3.4PB every month. Dragonfly has become one of the most important pieces of infrastructure at Alibaba.

While container technologies makes devops life easier most of the time, it sure brings a some challenges: the efficiency of image distribution, especially when you have to replicate image distribution on several hosts. Dragonfly works extremely well with both Docker and [PouchContainer](https://github.com/alibaba/pouch) for this scenario. It also is compatible with any other container formats.

It delivers up to 57 times the throughput of native docker and saves up to 99.5% the out bandwidth of registry(*2).

Dragonfly makes it simple and cost-effective to set up, operate, and scale any kind of files/images/data distribution.

## Is Dragonfly only designed for distribute container images

No, Dragonfly can be used to distribute all kinds of files, not only container images. For downloading files by Dragonfly, please refer to [Download a File](https://github.com/dragonflyoss/Dragonfly/blob/master/docs/quick_start/README.md#downloading-a-file-with-dragonfly). For pulling images by Dragonfly, please refer to [Pull an Image](https://github.com/dragonflyoss/Dragonfly/blob/master/docs/quick_start/README.md#pulling-an-image-with-dragonfly).

## What is SuperNode

SuperNode is a long-time process with two primary responsibilities:

- It's the tracker and scheduler in the P2P network that choose appropriate downloading net-path for each peer.
- It's also a CDN server that caches downloaded data from source to avoid downloading same files repeatedly.

## What is dfget

Dfget is the client of Dragonfly used for downloading files. It's similar to using wget.

At the same time, it also plays the role of peer, which can transfer data between each other in p2p network.

## What is dfdaemon

Dfdaemon is a local long-running process which is used for translating images pulling request from container engine. It establishes a proxy between container engine and registry.

Dfdaemon filters out layer fetching requests from all requests send by dockerd/pouchd when pulling images, then it uses dfget to downloading these layers.

## What is the sequence of supernode's CDN functionality and P2P distribution

When dfget starts to pull an image which has not been cached in supernode yet, supernode will do as the following sequence:

- First Step: supernode triggers the downloading task:
  - fetch the file/image length;
  - divide the length into pieces;
  - start to download the file piece by piece;
- Second Step:
  - supernode finishes to download one piece
  - supernode starts the P2P distribution work for the downloaded one piece among peers;

In a word, supernode does not have to wait for all the piece downloadings finished, it can concurrently start piece downloading once one piece downloaded.

## What is the peer scheduling algorithm of supernpde when many peers start to pull the same file

## What is the size of block(piece) when distribution

Dragonfly tries to make block(piece) size dynamically to ensure efficiency.

The size of pieces which is calculated as per the following strategy:

- If file's total size is less than 200MB, then the piece size is `4MB` by default.
- Otherwise, it equals to `min{ totalSize/100MB + 2 MB, 15MB }`.

## What is the difference between Dragonfly's P2P algorithm and bit-torrent(BT)

Dragonfly's P2P algorithm and bit-torrent(BT) are both the implementation of peer-to-peer protocol. For the difference between them, we describe them in the following table

|Aspect|Dragonfly|Bit-Torrent(BT)|
|:-:|:-:|:-:|
|seed's dynamically compress| support| no support|
|resume from break-point|support|no support|
|dynamically block size setting|support, see [question](#what-is-the-size-of-blockpiece-when-distribution)| no support(block size is fixed)|
|transparent of seed to client|support(user only needs to provide URL of file)|no support(BT needs to generate seed in tracker first, and then provide it to client) |

## What is the policy of bandwidth limit in peer network

We can enforce network bandwidth limit on both peer nodes and supernode of Dragonfly cluster.

For peer node itself, Dragonfly can set network bandwidth limit for two parts:

- one single downloading task; by default, limit is 10MB/s per downloading task. User can set task network bandwidth limit by parameter `--locallimit` within `dfget`.
- the whole network bandwidth consumed by P2P distribution on the peer node. This can be configured via parameter `totallimit` within `dfget`. If user has set `--totallimit=40M`, then both TX and RX limit are 40 MB/s.

For supernode, Dragonfly allows user to limit both the input and output network bandwidth.

## Why does dfget still keep running after file/image distribution finished

In a P2P network, a peer not only plays a role of downloader, but also plays a role of uploader.

- For downloader, peer needs to download block/piece of the file/image from other peers (supernode is one special peer as well);
- For uploader, peer needs to provide block/piece downloading service for other peers. At this time, other peers downloads block/piece from this peer, and this peer can be treated as an uploader.

Back to question, when a peer finishes to download file/image, while there may be other peers which is still downloading block/piece from this peer. Then this dfget process still exist for the potential uploading service.

Only when there is no new task to download file/images or no other peers coming to download block/piece from this peer, the dfget process will terminate.

## Do I have to change my container engine configuration to support Dragonfly

Currently Dragonfly supports almost all kinds of container engines, such as Docker, [PouchContainer](https://github.com/alibaba/pouch). When using Dragonfly, only one part of container engine's configuration needs update. It is the `registry-mirrors` configuration. This configuration update aims at making image pulling request from container engine will all be sent to `dfdaemon` process locally. And dfget does the request translation thing and proxies it to `dfget` locally.

Configure container engine's `registry-mirrors` is quite easy. We take docker as an example. Administrator should modify configuration file of docker `/etc/docker/daemon.json` to add the following item:

```yml
"registry-mirrors": ["http://127.0.0.1:65001"]
```

> Note: please remember restarting container engine after updating configuration.

## Do we support HA of supernode in Dragonfly

Currently no. Supernode in Dragonfly suffers the single node of failure. In the later release, we will try to realise the HA of Dragonfly.

## How to use Dragonfly in Kubernetes

It is very easy to deloy Dragonfly in Kubernetes with [Helm](https://github.com/helm/helm). For more information of Dragonfly's Helm Chart, please refer to project [dragonflyoss/helm-chart](https://github.com/dragonflyoss/helm-chart).

## Can an image from a third-party registry be pulled via Dragonfly

We **CANNOT** pull an image from a third-party registry via Dragonfly. Images from third-party registry means that the name of image has no registry address, for example, image `a.b.com/admin/mysql:5.6` is a third-party image, and it is from registry `a.b.com`. To the opposite, `admin/mysql:5.6` is not from third-party registry, because name of the image does not contain a registry address.

Because administrator needs to configure `--registry-mirrors=["http://127.0.0.1:65001"]` for container engine to proxy part of image pulling requests to `dfdaemon` which listens on 65001 locally, images not from a third-part registry will use this registry mirror. However, when user pulls image `a.b.com/admin/mysql:5.6` which contains a third-party registry address, the container engine will directly access the third-party registry to pull the image, ignoring the configured registry mirror. Then It has nothing to do with the `dfdaemon` within Dragonfly.

## Where is the log directory of Dragonfly client dfget

Log is very important for debugging or tracing. `dfget`'s log directory is `$HOME/.small-dragonfly/logs/dfclient.log`.

## Where is the data directory of Dragonfly client dfget

The meta data directory of `dfget` is `$HOME/.small-dragonfly/meta`.

This directory caches the address list of the local node (IP, hostname, IDC, security domain, and so on), managing nodes, and supernodes.

When started for the first time, the Dragonfly client will access the managing node. The managing node will retrieve the security domains, IDC, geo-location, and more information of this node by querying armory, and assign the most suitable supernode. The managing node also distributes the address lists of all other supernodes. dfget stores the above information in the meta directory on the local drive. When the next job is started, dfget reads the meta information from this directory, avoiding accessing the managing node repeatedly.

dfget always finds the most suitable assigned node to register. If fails, it requests other supernodes in order. If all fail, then it requests the managing node once again, updates the meta information, and repeats the above steps. If still fails, then the job fails.

> Note: If the dfclient.log suggests that the supernode registration fails all the time, try delete all files from the meta directory and try again.

## How to clean up the data directory of Dragonfly

Normally, Dragonfly automatically cleans up files which are not accessed in three minutes from the uploading file list, and the residual files that live longer than one hour from the data directory.

Follow these steps to clean up the data directory manually:

- Identify the account to which the residual files belong.
- Check if there is any running dfget process started by other accounts.
- If there is such a process, then terminate the process, and start a dfget process with the account identified at Step 1. This process will automatically clean up the residual files from the data directory.

## Does Dragonfly support pulling images from an HTTPS enabled registry

Currently, Dragonfly **DOES NOT** support pulling images from an HTTPS enabled registry.

`dfdaemon` is a proxy between container engine and registry. It captures every request sent from container engine to registry, filters out all the image layer downloading requests and uses `dfget` to download them.

We should investigate HTTPS proxy implementation and make it possible for dfdaemon to get HTTP data from tcp data, analysis them and download image layers correctly.

We are planning to support this feature in Dragonfly 0.5.0.

## Does Dragonfly support pulling an private image which needs username/password authentication

We are planning to support this feature in Dragonfly 0.5.0.