---
title: "Installing Client"
weight: 5
---

You have two options when installing the Dragonfly client: installing from the latest package, or installing from the source code.
<!--more-->

## Installing from the Latest Package

You can install from the latest packages we provided.

1. Download a package of the client.

    ```bash
    cd $HOME
    # Replace ${package} with a package appropriate for your operating system and location
    wget ${package}
    ```

    Available packages:

    - If you're in China:

        - [Linux 64-bit](http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_linux_amd64.tar.gz): `http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_linux_amd64.tar.gz`

        - [MacOS 64-bit](http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_darwin_amd64.tar.gz): `http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_darwin_amd64.tar.gz`

    - If you're not in China:

        - [Linux 64-bit](https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_linux_amd64.tar.gz): `https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_linux_amd64.tar.gz`

        - [MacOS 64-bit](https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_darwin_amd64.tar.gz): `https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_darwin_amd64.tar.gz`

2. Unzip the package.

    ```bash
    # Replace `xxx` with the installation directory.
    tar -zxf df-client_0.2.0_linux_amd64.tar.gz -C xxx
    ```

3. Add the directory of `df-client` to your `PATH` environment variable to make sure you can directly use `dfget` and `dfdaemon` command.

    ```bash
    # Replace `xxx` with the installation directory.
    # Execute or add this line to ~/.bashrc
    export PATH=$PATH:xxx/df-client/
    ```

## Installing from the Source Code

You can also install from the source code.

**Note:** You must have started Docker.

### Installing in $HOME/.dragonfly

1. Obtain the source code of Dragonfly.

    ```sh
    git clone https://github.com/dragonflyoss/Dragonfly.git
    ```

2. Enter the target directory.

    ```sh
    cd Dragonfly
    ```

3. Build `dfdaemon` and `dfget`.

    ```sh
    make build-client
    ```

4. Install `dfdaemon` and `dfget` in `/opt/dragonfly/df-client` and create soft-link in `/usr/local/bin`.

    ```sh
    sudo make install
    ```

## After this Task

Test if the downloading works.

    ```sh
    dfget --url "http://${resourceUrl}" --output ./resource.png --node "127.0.0.1"
    ```
