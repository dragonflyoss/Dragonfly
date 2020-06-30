# About this folder

All the documents in `config` folder help user to configure Dragonfly.

## How to configure dragonfly

We can use cli and yaml file to configure Dragonfly when deploying the system.
This tutorial only teaches you how to configure by yaml file.
If you want to config Dragonfly by cli, you can read docs in [cli_reference folder](https://github.com/dragonflyoss/Dragonfly/tree/master/docs/cli_reference).
In fact, learn this tutorial also will help you a lot, because the two ways are similar.

## About the yaml file

Because Dragonfly is composed of supernode, dfget, dfdaemon, you should learn how to configure them separately.
You can reference the three tutorials([supernode](supernode_properties.md), [dfget](dfget_properties.md), [dfdaemon](dfdaemon_properties.md)) to finish the yaml file and deploy.

## About deploying in docker

When deploying with Docker, you can mount the default path when starting up image with `-v`.
For supernode, you should start a supernode image using the following command.

```sh
docker run -d --name supernode \
    --restart=always \
    -p 8001:8001 \
    -p 8002:8002 \
    -v /etc/dragonfly/supernode.yml:/etc/dragonfly/supernode.yml \
    dragonflyoss/supernode:1.0.2
```

For dfdaemon, you can start the image in the same way.

```sh
docker run -d --net=host --name dfclient \
    -p 65001:65001 \
    -v /etc/dragonfly/dfdaemon.yml:/etc/dragonfly/dfdaemon.yml \
    -v /root/.small-dragonfly:/root/.small-dragonfly \
    dragonflyoss/dfclient:1.0.2
```