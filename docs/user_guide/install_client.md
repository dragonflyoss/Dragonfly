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

2. Unzip the package.

    ```bash
    # Replace `xxx` with the installation directory.
    # Replace ${package} with a package appropriate for your operating system and location
    tar -zxf ${package} -C xxx
    ```

3. Add the directory of `df-client` to your `PATH` environment variable to make sure you can directly use `dfget` and `dfdaemon` command.

    ```bash
    # Replace `xxx` with the installation directory.
    # Execute or add this line to ~/.bashrc
    export PATH=$PATH:xxx/df-client/
    ```

    Available packages:

    - If you're in China:

        - [Linux 64-bit](http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_linux_amd64.tar.gz): `http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_linux_amd64.tar.gz`

        - [MacOS 64-bit](http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_darwin_amd64.tar.gz): `http://dragonfly-os.oss-cn-beijing.aliyuncs.com/df-client_0.2.0_darwin_amd64.tar.gz`

    - If you're not in China:

        - [Linux 64-bit](https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_linux_amd64.tar.gz): `https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_linux_amd64.tar.gz`

        - [MacOS 64-bit](https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_darwin_amd64.tar.gz): `https://github.com/dragonflyoss/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_darwin_amd64.tar.gz`

## Installing from the Source Code

You can also install from the source code.

**Note:** You must have installed Go 1.7+, and added the Go command to the `PATH` environment variable.

### Installing in $HOME/.dragonfly

1. Obtain the source code of Dragonfly.

    ```sh
    git clone https://github.com/dragonflyoss/Dragonfly.git
    ```

2. Enter the target directory.

    ```sh
    cd Dragonfly
    ```

3. Install `dfdaemon` and `dfget` in `$HOME/.dragonfly/df-client`.

    ```sh
    ./build/build.sh client
    ```

4. Add the directory of `df-client` to your `PATH` environment variable to make sure you can directly use `dfget` and `dfdaemon` command.

    ```sh
    # Execute or add this line to ~/.bashrc
    export PATH=$HOME/.dragonfly/df-client:$PATH
    ```

### Installing in Another Directory

1. Obtain the source code of Dragonfly.

    ```sh
    git clone https://github.com/dragonflyoss/Dragonfly.git
    ```

2. Enter the target directory.

    ```sh
    cd Dragonfly/build/client
    ```

3. Install the client.

    ```sh
    ./configure --prefix=${your_installation_directory}
    make && make install
    ```

4. Add the directory of `df-client` to your `PATH` environment variable to make sure you can directly use `dfget` and `dfdaemon` command.

    ```sh
    # Execute or add this line to ~/.bashrc
    export PATH=${your_install_directory}/df-client:$PATH
    ```

## After this Task

Test if the downloading works.

    ```sh
    dfget --url "http://${resourceUrl}" --output ./resource.png --node "127.0.0.1"
    ```