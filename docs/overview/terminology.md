---
title: "Terminology"
weight: 5
---

This topic lists the common terms used throughout Dragonfly.
<!--more-->

## Supernode

Supernode is a long-time process with two primary responsibilities:

* It's the tracker and scheduler in the P2P network that choose appropriate downloading net-path for each peer.
* It's also a CDN server that caches downloaded data from source to avoid downloading same files repeatedly.

## dfget

Dfget is the client of Dragonfly used for downloading files. It's similar to wget.

At the same time, it also plays the role of peer, which can transfer data between each other in P2P network.

## dfdaemon

Dfdaemon is used for pulling images only. It establishes a proxy between dockerd/pouchd and registry.

Dfdaemon filters out layer fetching requests from all requests sent by dockerd/pouchd when pulling images, then uses dfget to downloading these layers.
