Quick Start
---

The latest release version is 0.2.0, you can quickly experience Dragonfly in the following simple steps.

## Start SuperNode on Docker Container

We provide 2 images in different places to speed up your pulling, you can choose one of them according to your location.
* China: registry.cn-hangzhou.aliyuncs.com/alidragonfly/supernode:0.2.0
* USA: registry.us-west-1.aliyuncs.com/alidragonfly/supernode:0.2.0

Here is the commands if you choose the registry `registry.cn-hangzhou.aliyuncs.com/alidragonfly/supernode:0.2.0`:
```bash
imageName="registry.cn-hangzhou.aliyuncs.com/alidragonfly/supernode:0.2.0"
docker pull ${imageName}
docker run -d -p 8001:8001 -p 8002:8002 ${imageName}
```

## Install Dragonfly Client

Download the proper package for your operating system and architecture:
* [df-client: macOS 64-bit](https://github.com/alibaba/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_darwin_amd64.tar.gz)
* [df-client: linux 64-bit](https://github.com/alibaba/Dragonfly/releases/download/v0.2.0/df-client_0.2.0_linux_amd64.tar.gz)

Uncompress the package and add the directory `df-client` to your `PATH` environment variable to make you can directly use `dfget` and `dfdaemon`.

## Use Dragonfly to Download a File

It's very simple to use Dragonfly to download a file, just like this:
```bash
dfget -u 'https://github.com/alibaba/Dragonfly/blob/master/docs/images/logo.png' -o /tmp/logo.png
```

## Use Dragonfly to Pull an Image

We have 2 steps to do before we pull an image:
1. start `df-daemon` with a specified registry:
    ```bash
    df-daemon --registry https://index.docker.io
    ```
2. configure dockerd and restart:
    ```json
    "registry-mirrors": ["http://127.0.0.1:65001"]
    ```

> NOTE: make sure the SuperNode is running

That's all we need to do, then we can pull an image by Dragonfly just as usual:
```bash
docker pull nginx:latest
```
