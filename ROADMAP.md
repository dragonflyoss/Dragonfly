# Dragonfly Roadmap

This document is used for contributors to have a better understanding of project's progress.

For `Dragonfly`, the following versions are tentatively defined as project milestones.
The following three versions are being planned currently:

1. v0.4.x, improved performance and stability.
2. v0.5.x, more adoptive with cloud env.
3. v1.0.x, GA version.

## v0.4.x

This version will focus on to improved the performance and stability of `Dragonfly`. Including:

* Golang SDK
* Supernode HA
* Merge the dfdaemon and dfget components
* Decentralized distribution

We will try to finish it before **October 31, 2019**.

### Golang SDK

`Dragonfly` needs a Golang SDK which is stored in https://github.com/dragonflyoss/Dragonfly/tree/master/client. And then we can achieve the following goals:

* every caller of `Dragonfly` can take advantages of this SDK by integrating it
* golang sdk encapsulates of the details of HTTP communication
* it could provide scalability and portability for other system to integrate `Dragonfly`.

Related issue: [#348](https://github.com/dragonflyoss/Dragonfly/issues/348)

### Supernode HA

As for now, we can specify multiple `supenodes` for dfclient at the same time. However, `supernode` is not aware of each other.
When one of the `supernode server` is unavailable, dfclient will try to register to the one of the remained `supernodes`.
The `supernode` registered will download the file from source server, reschedule pieces and re-build a P2P network because it lacks the information about the task and the peer node which will cost a lot of time.
So we should make multiple `supernodes` be aware of each other.

Related issue: [#232](https://github.com/dragonflyoss/Dragonfly/issues/232)

### Merge the dfdaemon and dfget components

Both `dfdaemon` and `dfget` are meant for the client and packaged together into dfclient.
So we can merge them into one single command to make the `Dragonfly` look simpler and easier to be deployed.

Related issue: [#806](https://github.com/dragonflyoss/Dragonfly/issues/806)

### Decentralized distribution

Now, `dfget` talks to `supernode` to discover peers information and publish its own status.
It works well, however, the user has to maintain a `supernode` server and `supernode` will become the bottleneck of the cluster.
So we can use a new approach where `dfget` can discover the same information without talking to `supernode` and
all the clients can form a gossip cluster, where they can fire and listen for events.

Related PR: [#594](https://github.com/dragonflyoss/Dragonfly/pull/594)

## v0.5.x

This version will focus on enhancing the Dragonfly's features to make it more adoptive with cloud env. Including:

* Support for more deployment options
* Support multi-server computing framework.
* Development of operation and maintenance tools.

We will try to finish it before **December 30, 2019**.

### Support for more deployment options

As a cloud native project, we should do more work to support deploy the `Dragonfly` on the kubernets platform. Including but not limited to the following list:

* Deploy `supernode` using [Helm](https://github.com/helm/helm) in Kubernetes to simplify the complexity of scaling SuperNodes in Kubernetes.
* Deploy `supernode` cluster using Operator.
* Deploy `dfget & dfdaemon` using DaemonSet in Kubernetes.

Related issue: [#346](https://github.com/dragonflyoss/Dragonfly/issues/346)

### Support multi-server computing framework

With the rise of ARM servers, x86 servers will not be the only choice.
However, the images of the two computing frameworks are incompatible for one application.
So the user will have to maintain the different versions for the same image which will bring challenges to management.

Related issue: [#775](https://github.com/dragonflyoss/Dragonfly/issues/775)

### Development of operation and maintenance tools

For better to use `Dragonfly`, we plan to provide more operation and maintenance tools to make it easy, including:

* A tool to validate the config file.
* A tool for performance testing of `Dragonfly` clusters.
* A dashboard tool for `Dragonfly`.

## v1.0.x

This is a stable version in GA stage. And we will focus on making `Dragonfly` support more scenarios. Including:

* Support different encryption algorithm in data transmission.
* Support multiple file transfer protocols, such as ftp, etc.
* Distribution images across cloud vendors.
* Support publish and subscribe mechanism.
* ......

This document give us a trunk direction, however it's not perfect, we may adjust the plan appropriately according to the urgency of demand.