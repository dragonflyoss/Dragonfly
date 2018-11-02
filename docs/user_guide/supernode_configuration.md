---
title: "Supernode Configuration"
weight: 15
---

The supernode is written in Java based on Spring Boot. You can easily set properties with command line parameters or with the configuration file.
<!--more-->

## Supernode Properties

Property Name | Default Value | Description
---|---|---
supernode.baseHome | /home/admin/supernode | Working directory of the supernode
supernode.systemNeedRate | 20 | Network rate reserved for the system (Unit: MB/s)
supernode.totalLimit | 200 | Network rate reserved for the supernode (Unit: MB/s)
supernode.schedulerCorePoolSize | 10 | Core pool size of ScheduledExecutorService

## Setting Properties

You have two options when setting properties of a supernode.

- Setting properties with command line parameters.

    ```bash
    java -D<propertyName>=<propertyValue> -jar supernode.jar
    ```

- Setting properties with the configuration file.

    ```bash
    java -Dspring.config.location=./config.properties,<otherConfigFilePath> -jar supernode.jar
    ```