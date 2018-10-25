---
title: "超级节点的配置"
weight: 15
---

This topic explains the configuration of SuperNode.
<!--more-->

## Properties

Property Name | Default Value | Description
------------- | ------------- | -----------
supernode.baseHome | /home/admin/supernode | working directory of supernode,
supernode.systemNeedRate | 20 | the network rate reserved for system, unit is: MB/s
supernode.totalLimit | 200 | the network rate that supernode can use, unit is: MB/s
supernode.schedulerCorePoolSize | 10 | the core pool size of ScheduledExecutorService

## Usage

Currently, the SuperNode is written by Java based on spring-boot. It can easily set properties through the following methods:

* Commandline parameter:
    ```bash
    java -D<propertyName>=<propertyValue> -jar supernode.jar
    ```
    
* Configuration file:
    ```bash
    java -Dspring.config.location=./config.properties,<otherConfigFilePath> -jar supernode.jar
    ```