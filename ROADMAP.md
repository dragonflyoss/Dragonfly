# Dragonfly Roadmap

The `Supernode` of `Dragonfly` was written in Java, as a Sandbox Level Project in CNCF(Cloud Native Computing Foundation), we choose to refactor `Supernode` with Golang. In the meantime, we cherish every contributor who is willing to participate in the project.

This document is so crucial in order for contributors to have a better understanding of the progress of the project. For our project, the following versions are tentatively defined as project milestones. In order to welcome contributors to the community, I'll also detail the important features that need to be implemented in this document.

We are currently planning three versions:

1. v0.3.0, as the transition version between Java and Go.
2. v0.4.0, as a preliminary version of Go implementation.
3. v0.5.0, as a pre-GA version of Go implementation.

## v0.3.0

This version will exist as a transition version. In this version, we will support Java version and go version in parallel. Including:

* Bugfix with the Java version.
* Refactor with Golang with the function of beggar version.

NOTE: **In the meantime,The delivery of documentations is also critical.**

We will try to finish it before **February 1, 2019**.

### Bug fix with the Java version

In this release, the project for `Go` is not fully functional or even runnable, so we will continue to fix some bugs in the Java version to serve the general public.

// TODO

### Refactor with Golang

In this release, we support the implementation of a beggar version of the project for the Go version. We will implement the main logic of work flow with simplify the scheduler, CDN and some other functions. Including:

* P2P Network Register. The entire registration logic needs to be implemented. The CDN module is not implemented for the time being, we download files from the source directly and saved locally if not exist temporarily.
* P2P Network Download. We use a simplified version of the scheduler to complete the p2p network, which works, but is fragile.

## v0.4.0

At this point, `Dragonfly` is a completely project written by `Go`, we will implement the basic functionality of the Java version and replace it. Including:

* Delete Java source code and archive it.
* Implement the functionality basically.

We will try to finish it before **March 15, 2019**.

### Archive Java source code

In this version, the `Go` version is basically a replacement for the Java version. So we will remove the Java source code and stop maintain it, although we'll still provide an image of the latest Java version for archive.

* Package the Java code.
* Delete the Java source code.

### Implement the functionality basically

In this release, we've implemented most of the functionality and some advanced features of `Dragonfly` with `Go`, to the point of being aligned with the Java version. Including:

* Scheduler. Complete scheduling logic, including blacklist or something others. In addition, for better extensibility, we should support it as a plug-in.
* CDN support. Use CDN to Cache downloaded data from source to avoid downloading same files repeatedly.
* Dynamic downloading rate limiting.
* Disk GC.
* Preheat image.

## v0.5.0

This is a basically stable version, reaching the pre-GA stage. Including:

1. Stability & Security enhancement.
2. Function Enhancement.

We will try to finish it before **April 15, 2019**.

### Stability & Security enhancement

Enhance the stability of the project.

* Support more configuration items.
* Support private container image.
* Support authentication in SuperNode API.
* Different encryption algorithm in data transmission.

### Function Enhancement

Add more advanced functionality on the basis of previous versions.

* Support multi-registrys.
* Support supernode cluster. Cluster the SuperNode to decrease possibility of failure.
* Supernode support storage driver.
* Use [IPFS](https://github.com/ipfs/go-ipfs) to share block datas between SuperNodes.
* Highly user-customized modules.

## Future

We're going to do a lot more work on the community ecosystem, and before GA, we need to make our project not only to be useful, but also to be perfect. Including but not limited to the following list:

* Deploy SuperNode using [Helm](https://github.com/helm/helm) in Kubernetes to simplify the complexity of scaling SuperNodes in Kubernetes.
* Deploy Supernode cluster using [Operator](https://coreos.com/operators/).
* Deploy dfget & dfdaemon using DaemonSet in Kubernetes.
* Integration with [Harbor](https://github.com/goharbor/harbor).
* Plug-In policy for CNCF projects.
* Integrated monitoring mechanism, the combination of [Prometheus](https://github.com/prometheus/prometheus) and [Grafana](https://github.com/grafana/grafana) may be our first choice.
* ......

This document give us a trunk direction, however it's not perfect, we may adjust the plan appropriately according to the urgency of demand. Finally, we plan to reach GA of the project at **May 21, 2019**, which is the time of [Kubecon Europe 2019](https://events.linuxfoundation.org/events/kubecon-cloudnativecon-europe-2019/). Let's do it together.