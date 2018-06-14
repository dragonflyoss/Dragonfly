# 安装客户端

- **通过最新软件包安装**

  - 下载软件包: [df-client.linux-amd64.tar.gz](https://github.com/alibaba/Dragonfly/raw/master/package/df-client.linux-amd64.tar.gz)
  - 执行命令，其中“xxx”是安装目录：
  ```sh
  tar xzvf df-client.linux-amd64.tar.gz -C xxx
  ```
  - 设置`PATH`环境变量:
  ```sh
  export PATH=$PATH:xxx/df-client
  ```

- **通过源码安装**

  > **要求**: go1.7+, 并且`go`命令在`PATH`环境变量中.

  - 从GitHub获取源码:
  ```sh
  git clone https://github.com/alibaba/Dragonfly.git
  ```

  - 一键安装

    ```bash
    cd Dragonfly && ./build/build.sh client && export PATH=$HOME/.dragonfly/df-client:$PATH
    ```
  - 或者自定义安装
    - 进入客户端编译脚本目录:
        ```sh
        cd Dragonfly/build/client
        ```

    - 编译安装:
        ```sh
        # --prefix=xxx 指定安装目录, 默认为当前目录
        ./configure --prefix=xxx
        # 编译
        make
        # 也可以使用'make package'生成安装包
        make install
        # 清理文件
        make clean
        ```

    - 设置`PATH`环境变量:
        ```sh
        export PATH=$PATH:xxx/df-client
        ```
