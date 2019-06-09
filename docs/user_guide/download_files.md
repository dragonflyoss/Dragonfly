# Downloading Files

Things are done differently when you download container images and download general files with Dragonfly.

## Prerequisites

- You are using Linux operating system.
- The supernode service is started.

    **Tip:** For more information on the dfget command, see [dfget](../cli_reference/dfget.md). For more information on the installation of supernodes, see [Installing Server](./install_server.md).

## Downloading container images

1. Specify the supernodes.

    a. Open the Dragonfly configuration file.

    ```sh
    vi /etc/dragonfly.conf
    ```

    b. Add the IP of supernodes separated by comma to the configuration file.

    ```sh
    [node]
    address=nodeIp1,nodeIp2
    ```

2. Start the dfget proxy (dfdaemon).

    ```sh
    # Start dfdaemon and specify the image repo URL. The default port is `65001`.
    dfdaemon --registry https://xxx.xx.x
    # Review dfdaemon logs
    tailf ~/.small-dragonfly/logs/dfdaemon.log
    ```

    **Tip:** To list all available parameters for dfdaemon, run `dfdeaemon -h`.

3. Configure the Daemon Mirror.

    a. Modify the configuration file `/etc/docker/daemon.json`.

    ```sh
    vi /etc/docker/daemon.json
    ```

    **Tip:** For more information on `/etc/docker/daemon.json`, see [Docker documentation](https://docs.docker.com/registry/recipes/mirror/#configure-the-cache).

    b. Add or update the configuration item `registry-mirrors` in the configuration file.

    ```sh
    "registry-mirrors": ["http://127.0.0.1:65001"]
    ```

    c. Restart Docker daemon.

    ```bash
    systemctl restart docker
    ```

    d. Add authentication info for the private docker registry in `~/.docker/config.json` if the registry is configured with auth.

    ```json
    {
          "auths": {
                  "https://index.docker.io/v1/": {
                          "auth": "${auth_value}"
                  }
          }
    }
    ```

    The ${auth_value} is `base64("${usename}:${password}")`.

    ```bash
    echo "${usename}:${password}" | base64
    ```

4. Download an image with Dragonfly.

    ```bash
    docker pull {imageName}
    ```

    **Note:** Don't include the image repo URL in {imageName}, because the repo URL has been specified with the `registry` parameter when starting dfdaemon.

## Downloading General Files

1. Specify the supernodes in one of the following ways.

    - Specifying with the configuration file.

        ```sh
        # Open the Dragonfly configuration file.
        vi /etc/dragonfly.conf

        # Add the IP of supernodes separated by comma to the configuration file
        [node]
        address=nodeIp1,nodeIp2
        ```

    - Specifying with the parameter in the command line.

        ```sh
        dfget -u "http://www.taobao.com" -o /tmp/test.html --node nodeIp1,nodeIp2
        ```

        **Note:** When using this method, you must add the `node` parameter every time when you run the dfget command. And the parameter in the command line takes precedence over the configuration file.

2. Download general files with Dragonfly in one of the following ways.

    - Download files with the default `/etc/dragonfly.conf` configuration.

        ```sh
        dfget --url "http://xxx.xx.x"
        ```

        **Tip:** To list all available parameters for dfget, run `dfget -h`.

    - Download files with your specified supernodes.

        ```sh
        dfget --url "http://xxx.xx.x" --node "127.0.0.1"
        ```

    - Download files to your specified output file.

        ```sh
        dfget --url "http://xxx.xx.x" -o a.txt
        ```

## After this Task

To review the downloading log, run `less ~/.small-dragonfly/logs/dfclient.log`.

