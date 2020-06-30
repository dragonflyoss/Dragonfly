# Frequently Asked Questions (FAQ)

FAQ contains some frequently asked questions about two aspects:

- First, user-facing functionalities.
- Second, underlying concept and theory.

Technical questions will not be included in FAQ.

## What is Dragonfly

**Dragonfly is an intelligent P2P based image and file distribution system.**

It aims to resolve issues related to low-efficiency, low-success rate and waste of network bandwidth in file transferring process. Especially in large-scale file distribution scenarios such as application distribution, cache distribution, log distribution, image distribution, etc.

In Alibaba, Dragonfly is invoked 2 Billion times and the data distributed is 3.4PB every month. Dragonfly has become one of the most important pieces of infrastructure in Alibaba.

While container technologies make DevOps life easier most of time, it sure brings some challenges: the efficiency of image distribution, especially when you have to replicate image distribution on several hosts. Dragonfly works extremely well with both Docker and [PouchContainer](https://github.com/alibaba/pouch) for this scenario. It also is compatible with any other container formats.

It delivers up to 57 times the throughput of native docker and saves up to 99.5% the out bandwidth of registry(*2).

Dragonfly makes it simple and cost-effective to set up, operate, and scale any kinds of files/images/data distribution.

## Is Dragonfly only designed for distribute container images

No, Dragonfly can be used to distribute all kinds of files, not only container images. For downloading files by Dragonfly, please refer to [Download a File](https://github.com/dragonflyoss/Dragonfly/blob/master/docs/quick_start/README.md#downloading-a-file-with-dragonfly). For pulling images by Dragonfly, please refer to [Pull an Image](https://github.com/dragonflyoss/Dragonfly/blob/master/docs/quick_start/README.md#pulling-an-image-with-dragonfly).

## What is the sequence of P2P distribution

Supernode will maintain a bitmap which records the correspondence between peers and pieces. When dfget starts to download, supernode will return several pieces info (4 by default）according to the scheduler.

**NOTE**: The scheduler will decide whether to download from the supernode or other peers. As for the detail of the scheduler, please refer to [scheduler algorithm](#what-is-the-peer-scheduling-algorithm-by-default)

## How do supernode and peers manage file cache which is ready for other peer's pulling

Supernode will download files and cache them via CDN. For more information, please refer to [The sequence of supernode's CDN functionality](#what-is-the-sequence-of-supernode's-cdn-functionality).

After finishing distributing file from other peers, `dfget` should do two kinds of things:

- first, construct all the pieces into unioned file(s);
- second, backup the unioned file(s) in a configured directory, by default "$HOME/.small-dragonfly/data" with a suffix of ".server";
- third, move the original unioned file(s) to the destination path.

**NOTE**: The supernode cannot manage the cached file which is already distributed to peer nodes at all. For  what will happen if the cached file on a peer is deleted, please refer to [What if you kill the dfget server process or delete the source files](#what-if-you-kill-the-dfget-server-process-or-delete-the-source-files)

## What is the sequence of supernode's CDN functionality

When dfget registers a task to supernode, supernode will check whether the file to be downloaded has a cache locally.

If the file to be downloaded  has not been cached in supernode yet, supernode will do as the following sequence:

- First Step: supernode triggers the downloading task asynchronously:
  - fetch the file/image length;
  - divide the length into pieces;
  - start to download the file piece by piece and store it locally;
- Second Step:
  - supernode finishes to download one piece
  - supernode starts the P2P distribution work for the downloaded one piece among peers;

If the requested file has already been cached in supernode, supernode will send an HTTP GET request, which contains both HTTP headers `If-None-Match:<eTag>` and `If-Modified-Since:<lastModified>`, to source file network address to determine whether the remote file has been updated.

In addition, supernode does not have to wait for all the piece downloading finished, so it can concurrently start pieces downloading once one piece has been downloaded.

## What will happen  if you kill the dfget server process or delete the source files

If a file on a peer is deleted manually or by GC, the supernode won't know that. And in the subsequent scheduling, if multiple download tasks fail from this peer, the scheduler will add it to a blacklist. So do with that if the server process is killed or other abnormal conditions.

## What is the peer scheduling algorithm by default

- Distribute the number of pieces evenly. Select the pieces with the smallest number in the entire P2P network so that the distribution of each piece in the P2P network is balanced to avoid "Nervous resources".

- Nearest distance priority. For a peer, the piece closest to the piece currently being downloaded is preferentially selected, so that the peer can achieve the effect of sequential read and write approximately, which will improve the I/O efficiency of file.

- Local blacklist and global blacklist. An example is easier to understand: When peer A fails to download from peer B, B will become the local blacklist of A, and then the download tasks of A will filter B out; When the number of failed downloads from B reaches a certain threshold, B will become the global blacklist, and all the download tasks will filter B out.

- Self-isolation. PeerA will only download files from the supernode after it fails to download multiple times from other peers, and will also be added to the global blacklist. So the peer A will no longer interact with peers other than the supernode.

- Peer load balancing. This mechanism will control the number of upload pieces and download pieces that each peer can provide simultaneously and the priority as the target peer.

## What is the size of block(piece) when distribution

Dragonfly tries to make block(piece) size dynamically to ensure efficiency.

The size of pieces which is calculated as per the following strategy:

- If file's total size is less than 200MB, then the piece size is `4MB` by default.
- Otherwise, it equals to `min{ totalSize/100MB + 2 MB, 15MB }`.

## What is the difference between Dragonfly's P2P algorithm and bit-torrent(BT)

Dragonfly's P2P algorithm and bit-torrent(BT) are both the implementation of peer-to-peer protocol. For the difference between them, we describe them in the following table:

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

You can also config it with config files, refer to [link](./docs/cli_reference/dfget.md#etcdragonflyconf).

## Why does dfget still keep running after file/image distribution finished

In a P2P network, a peer can not only plays a role of downloader, but also plays a role of uploader.

- For downloader, peer needs to download block/piece of the file/image from other peers (supernode is a special peer as well);
- For uploader, peer needs to provide block/piece downloading service for other peers. At this time, other peers downloads block/piece from this peer, and this peer can be treated as an uploader.

Back to question, when a peer finishes to download file/image, while there may be other peers which are still downloading block/piece from this peer. Then this dfget process still exists for the potential uploading service.

Only when there is no new task to download file/images or no other peers coming to download block/piece from this peer, the dfget process will terminate.

## Do I have to change my container engine configuration to support Dragonfly

Currently Dragonfly supports almost all kinds of container engines, such as Docker, [PouchContainer](https://github.com/alibaba/pouch). When using Dragonfly, only one part of container engine's configuration needs to be updated. It is the `registry-mirrors` configuration. This configuration update aims at making image pulling requests, which does not contain a third-party non-dockerhub registry address, from container engine will all be sent to `dfdaemon` process locally. And dfget does the request translation thing and proxies it to `dfget` locally.

Configure container engine's `registry-mirrors` is quite easy. We take docker as an example. Administrator should modify configuration file of docker `/etc/docker/daemon.json` to add the following item:

```yml
"registry-mirrors": ["http://127.0.0.1:65001"]
```

With updating the configuration, request `docker pull mysql:5.6` will be sent to dfdaemon, while request `docker pull a.b.com/mysql:5.6` will not be sent to dfdaemon. Because docker engine only deals with official images from docker hub to take advantages of registry mirror. However, if you set `--registry a.b.com` in dfdaemon, and send a request `docker pull mysql:5.6`, dfdaemon will proxy the request and distributes image `mysql:5.6` from registry `a.b.com`.

> Note: please remember restarting container engine after updating configuration.

## Can I set dfdaemon as HTTP_PROXY?

Yes, please refer to the [proxy guide](./docs/user_guide/proxy.md).

## Do we support HA of supernode in Dragonfly

Currently no. In the later release, we will try to release the HA of Dragonfly.

In fact, you can [provide multiple supernodes for dfget](#how-to-config-supernodes-for-dfget) as an alternative. When a peer started to download a task, it will register to one of the supernode list randomly. And when the supernode suffers failure, the task being downloaded on it will automatically migrate to the other supernodes in the supernode list.

## How to config supernodes for dfget

There are two ways to config one or multiple supernodes for dfget

- config with config file

```shell
cat <<EOD > /etc/dragonfly/dfget.yml
nodes:
    - supernode01
    - supernode02
    - supernode03
EOD
```

- config with cli

```shell
dfget --node supernode01 --node supernode02 --node supernode03

or

dfget --node supernode01,supernode02,supernode03
```

NOTE: If you use dfdaemon to call dfget, you can also pass this parameter to dfget via `dfdaemon --node`.

## How to use Dragonfly in Kubernetes

It is very easy to deploy Dragonfly in Kubernetes with [Helm](https://github.com/helm/helm). For more information of Dragonfly's Helm Chart, please refer to project [dragonflyoss/helm-chart](https://github.com/dragonflyoss/helm-chart).

## Can an image from a third-party registry be pulled via Dragonfly

We **CANNOT** pull an image from a third-party registry via Dragonfly. Images from third-party registry means that the name of image has no registry address, for example, image `a.b.com/admin/mysql:5.6` is a third-party image, and it is from registry `a.b.com`. To the opposite, `admin/mysql:5.6` is not from third-party registry, because name of the image does not contain a registry address.

Because administrator needs to configure `--registry-mirrors=["http://127.0.0.1:65001"]` for container engine to proxy part of image pulling requests to `dfdaemon` which listens on 65001 locally, images not from a third-part registry will use this registry mirror. However, when user pulls image `a.b.com/admin/mysql:5.6` which contains a third-party registry address, the container engine will directly access the third-party registry to pull the image, ignoring the configured registry mirror. Then It has nothing to do with the `dfdaemon` within Dragonfly.

## Where is the log directory of Dragonfly client dfget

Log is very important for debugging and tracing. `dfget`'s log directory is `$HOME/.small-dragonfly/logs/dfclient.log`.

## Where is the data directory of Dragonfly client dfget

The meta data directory of `dfget` is `$HOME/.small-dragonfly/meta`.

This directory caches the address list of the local node (IP, hostname, IDC, security domain, and so on), managing nodes, and supernodes.

When started for the first time, the Dragonfly client will access the managing node. The managing node will retrieve the security domains, IDC, geo-location, and more information of this node by querying armory, and assign the most suitable supernode. The managing node also distributes the address lists of all other supernodes. dfget stores the above information in the meta directory on the local drive. When the next job is started, dfget reads the meta information from this directory, avoiding accessing the managing node repeatedly.

dfget always finds the most suitable assigned node to register. If fails, it requests other supernodes in order. If all fail, then it requests the managing node once again, updates the meta information, and repeats the above steps. If still fails, then the job fails.

> Note: If the dfclient.log suggests that the supernode registration fails all the time, try delete all files from the meta directory and try again.

## What is temp directory of dfget and dfdaemon

The default temp data directory of `dfget` is `$HOME/.small-dragonfly/data` which cannot be configured at present.

The default temp data directory of `dfdaemon` is `$HOME/.small-dragonfly/dfdaemon/data` which can be configured by flag `--localrepo`.

When `dfdaemon` downloads images by `dfget`, it will set the target file path of `dfget` which  is `the temp data directory of dfdaemon`. So `dfget` will download the file to `$HOME/.small-dragonfly/data` and move to `$HOME/.small-dragonfly/dfdaemon/data` after downloaded totally.

## How to clean up the data directory of Dragonfly

Normally, Dragonfly automatically cleans up files which are not accessed in three minutes from the uploading file list, and the residual files that live longer than one hour from the data directory.

Follow these steps to clean up the data directory manually:

- Identify the account to which the residual files belong.
- Check if there are any running dfget process started by other accounts.
- If there is such a process, then terminate the process, and start a dfget process with the account identified at Step 1. This process will automatically clean up the residual files from the data directory.

## Does Dragonfly support pulling images from an HTTPS enabled registry

Currently, Dragonfly **DOES NOT** support pulling images from an HTTPS enabled registry.

`dfdaemon` is a proxy between container engine and registry. It captures every request sent from container engine to registry, filters out all the image layer downloading requests and uses `dfget` to download them.

We should investigate HTTPS proxy implementation and make it possible for dfdaemon to get HTTP data from tcp data, analysis them and download image layers correctly.

We are planning to support this feature in Dragonfly 0.5.0.

## Does Dragonfly support pulling an private image which needs username/password authentication

Yes, Dragonfly supports users to pull private image which needs username/password authentication. For example, when user wishes to pull private images from docker registries(such as Docker Hub https://index.docker.io/ ), add authentication info of the user for the private images in `/root/.docker/config.json`. The format of `config.json` after adding authentication will be like the following way:

```json
{
      "auths": {
              "https://index.docker.io/v1/": {
                      "auth": "${auth_value}"
              }
      }
}
```

In the above content the `auth_value` base64("${username}:${password}"). Since users definitely have their own username/password for private images, then use the following command to generate `$auth_value` and fill the generated result in `config.json`:

```shell
echo "${username}:${password}" | base64
```

## How to check if block piece is distributed among dfgets nodes

Supernode and dfget nodes together build a peer-to-peer network. The running log of dfget will explicitly show where the received block piece is from, supernode or other dfget node.

User can get dfget's log from file `$HOME/.small-dragonfly/logs/dfclient.log`, and search the keyword `downloading piece`. If the prefix of field `dstCid` is `cdnnode`, then this piece block is from supernode, otherwise it is from dfget node.

```
## download from supernode
2019-02-13 15:20:45.757 INFO sign:31923-1550042443.708 : downloading piece:{"taskID":"b4b0f175f7aef583ff6ff8da6b00024d7772b165caa66ff8ef3a9dce6701b690","superNode":"127.0.0.1","dstCid":"cdnnode:127.0.0.1~b4b0f175f7aef583ff6ff8da6b00024d7772b165caa66ff8ef3a9dce6701b690","range":"0-4194303","result":503,"status":701,"pieceSize":4194304,"pieceNum":0}

# download from other dfget peer
2019-02-13 15:22:40.062 INFO sign:32047-1550042560.044 : downloading piece:{"taskID":"b4b0f175f7aef583ff6ff8da6b00024d7772b165caa66ff8ef3a9dce6701b690","superNode":"127.0.0.1","dstCid":"127.0.0.1-31923-1550042443.708","range":"0-4194303","result":503,"status":701,"pieceSize":4194304,"pieceNum":0}
```

## How to view all the dfget logs of a task

You can follow the steps:

- find a failed task: `grep 'download FAIL' dfclient.log`, such as:

  ```sh
  2019-05-22 05:40:58.120 INFO sign:38923-1558496382.915 : download FAIL cost:75.208s length:4120442 reason:0
  ```

- get all the logs of this task through the sign `38923-1558496382.915`: `grep 38923-1558496382.915 dfclient.log`, such as:

  ```sh
  2019-05-22 05:39:42.919 INFO sign:38923-1558496382.915 : get cmd params:["dfget" "-u" "https://xxx" "-o" "./a.test"]
  ...
  ...
  2019-05-22 05:40:58.120 INFO sign:38923-1558496382.915 : download FAIL cost:75.208s length:4120442 reason:0
  ```

## Can I use self-specified ports for dragonfly

Announce the results firstly, and it is yes.

Here are the port list that dragonfly will use:

Name                           | Default Value | Description
------------------------------ | ------------- | ----------
dfdaemon proxy server port     | 65001         | The port that dfdaemon proxy will listen.
dfget uploader server port     | Random        | The port that the dfget uploader server will listen.
supernode register port        | 8002          | It's used for clients to register themselves as peers of p2p-network into supernode.
supernode cdn file server port | 8001          | It's used for clients to download file pieces from supernode.

And each item in the above table can be self-defined.

Name                           | Flag                      | Remark
------------------------------ | ------------------------- | ----------
dfdaemon proxy server port     | dfdaemon --port           | The port should   be in the range of 2000-65535.
dfget uploader server port     | dfget server --port       | You can use command `dfget server` to start a uploader server before using dfget to download if you don't want to use a random port.
supernode register port        | supernode --port          | You can use `dfget --node host:port` to register with the specified supernode register port.
supernode cdn file server port | supernode --download-port | You should prepare a file server firstly and listen on the port that flag `download-port` will use.

**NOTE**: The supernode maintains both Java and Golang versions currently. And the above table is for the Golang version. And you will get a guide [here](https://d7y.io/en-us/docs/userguide/supernode_configuration.html) for Java version.

## Why the time in the log is wrong

If you are in China, docker container uses UTC time(Coordinated Universal Time), and the host uses CST time(China Shanghai Time). So the log's time is 8 hours behind the host time. If you want to make their time consistent, you should add a config `-v /etc/localtime:/etc/localtime:ro` before you start a container. For example, you can run a command as follows to start a dfclient.

```sh
docker run -d --name dfclient \
    -v /etc/localtime:/etc/localtime:ro \
    -p 65001:65001 \
    dragonflyoss/dfclient:1.0.2 --registry https://index.docker.io
```

## How to join Dragonfly as a member

Please check the [CONTRIBUTING.md](CONTRIBUTING.md#join-dragonfly-as-a-member)

## How dfget connect to supernodes in multiple-supernode mode

If supernodes are set in multiple-supernode mode, dfget will connect to one of these supernodes randomly.
Because dfget will randomize the order of all supernodes it knows and store them in a slice.
If dfget connects to the first supernode unsuccessfully, it will connect to the second supernode in the slice.
And so on until all the known supernodes fail to access twice, the dfget will exit with download failure.