+++
title = "常见问题"
weight = 60
chapter = false
pre = "<b>6. </b>"
+++

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
