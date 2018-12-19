+++
title = "FAQ"
weight = 60
chapter = false
pre = "<b>6. </b>"
+++

Find the answers to the frequently asked questions.
<!--more-->

## How can I pull images with Dragonfly?

See [Pulling an Image with Dragonfly](../quick_start/README.md#pulling-an-image-with-dragonfly).

## How can I download files with Dragonfly?

See [Downloading a File with Dragonfly](../quick_start/README.md#downloading-a-file-with-dragonfly).

## What is a supernode?

A supernode is a long-time process that plays the following roles:

- A tracker and scheduler in the P2P network that chooses appropriate downloading net-path for each peer.
- A CDN server that caches downloaded data from source to avoid downloading same files repeatedly.

## What is dfget?

Dfget is the Dragonfly client used for downloading files. It's similar to wget.

Meanwhile, it also plays the role of a peer, which can transfer data between each other in the P2P network.

## What is dfdaemon?

Dfdaemon is only used for pulling images. It establishes a proxy between dockerd/pouchd and registry.

Dfdaemon filters out layer fetching requests from all requests sent by dockerd/pouchd when pulling images, then uses dfget to download these layers.

## Where is the installation directory of Dragonfly client dfget?

Normally, there are two installation directories:

- Dragonfly plugin for StarAgent: `/home/staragent/plugins/dragonfly/dfget`
- StarAgent's built-in dfget: `/home/staragent/bin/dfget`

The Dragonfly plugin is used by default. If the Dragonfly plugin is not installed, then the StarAgent's build-in dfget is used.

## Where is the log directory of Dragonfly client dfget?

The log directory is `$HOME/.small-dragonfly/logs/dfclient.log`.

## Where is the data directory of Dragonfly client dfget?

The data directory is `$HOME/.small-dragonfly/data`.

Each account has its own data directory. A P2P downloading job generates two data files under this directory:

- A temporary downloading file for the target file, with the name `targetFileName-sign`. This file is moved to the target directory after the download is complete.
- A copy of the temporary downloading file for uploading, with the name `targetFileName-sign.service`. If no downloading jobs are downloading this file from this node, then it gets cleaned after three minutes. Before the process is completed normally, any file lasting longer than 60 minutes will be purged.

## Where is the meta directory of Dragonfly client dfget?

The meta directory is `$HOME/.small-dragonfly/meta`.

This directory caches the address list of the local node (IP, hostname, IDC, security domain, and so on), managing nodes, and supernodes.

When started for the first time, the Dragonfly client will access the managing node. The managing node will retrieve the security domains, IDC, geo-location, and more information of this node by querying armory, and assign the most suitable supernode. The managing node also distributes the address lists of all other supernodes. dfget stores the above information in the meta directory on the local drive. When the next job is started, dfget reads the meta information from this directory, avoiding accessing the managing node repeatedly.

dfget always finds the most suitable assigned node to register. If fails, it requests other supernodes in order. If all fail, then it requests the managing node once again, updates the meta information, and repeats the above steps. If still fails, then the job fails.

**Tip**: If the dfclient.log suggests that the supernode registration fails all the time, try delete all files from the meta directory and try again.

## How to check the version of Dragonfly client dfget?

If you have installed the Dragonfly client, run this command:

```bash
dfget -v
```

## Why is the dfget process still there after the download is complete?

Because Dragonfly adopts a P2P downloading approach, dfget provides uploading services in addition to downloading services. If there are no new jobs within five minutes, the dfget process will be terminated automatically.

## How to clean up the data directory of Dragonfly?

Normally, Dragonfly automatically cleans up files which are not accessed in three minutes from the uploading file list, and the residual files that live longer than one hour from the data directory.

Follow these steps to clean up the data directory manually:

1. Identify the account to which the residual files belong.
2. Check if there is any running dfget process started by other accounts.
3. If there is such a process, then terminate the process, and start a dfget process with the account identified at Step 1. This process will automatically clean up the residual files from the data directory.
