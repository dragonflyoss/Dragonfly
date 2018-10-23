---
title: "Install Client"
weight: 5
---

This topic explains how to install the client.
<!--more-->

- **Install From Latest Package**

  - Download [df-client.linux-amd64.tar.gz](https://github.com/alibaba/Dragonfly/raw/master/package/df-client.linux-amd64.tar.gz)
  - `tar xzvf df-client.linux-amd64.tar.gz -C xxx`, "xxx" is installation directory.
  - Set environment variable named PATH: `export PATH=$PATH:xxx/df-client`

- **Install From Source Code**

  *Requirements: go1.7+ and the go cmd must be in environment variable named PATH.*

  - `git clone https://github.com/alibaba/Dragonfly.git`, clone source code from GitHub.
  - install in $HOME/.dragonfly

    ```bash
    cd Dragonfly
    # install `dfdaemon` and `dfget` in $HOME/.dragonfly/df-client
    ./build/build.sh client
    # set PATH
    export PATH=$HOME/.dragonfly/df-client:$PATH
    ```

  - install in other directory

    ```bash
    cd Dragonfly/build/client
    ./configure --prefix=${your_install_directory}
    make && make install
    export PATH=${your_install_directory}/df-client:$PATH
    ```
