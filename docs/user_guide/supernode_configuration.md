# Supernode Configuration

The supernode is written in Java based on Spring Boot. You can easily set properties with command line parameters or with the configuration file.
<!--more-->

## Supernode Properties

### Simple Property

Property Name | Default Value | Description
---|---|---
supernode.baseHome | /home/admin/supernode | Working directory of the supernode
supernode.systemNeedRate | 20 | Network rate reserved for the system (Unit: MB/s)
supernode.totalLimit | 200 | Network rate reserved for the supernode (Unit: MB/s)
supernode.schedulerCorePoolSize | 10 | Core pool size of ScheduledExecutorService
supernode.dfgetPath | /usr/local/bin/dfget/ | The `dfget` path

### Cluster Property

#### supernode.cluster

This is an array property, and every member of it has these attributes:

Name | Default Value | Description
---- | ------------- | -----------
ip   | None          | The ip of the cluster member.
downloadPort | 8001  | The download port of the cluster member.
registerPort | 8002  | The register port of the cluster member.

- Config it in `.properties` file, for example:

    ```ini
    supernode.cluster[0].ip = '192.168.0.1'
    supernode.cluster[0].registerPort = 8002
    supernode.cluster[1].ip = '192.168.0.2'
    ```

- Config it in `.yaml` file, for example:

    ```yaml
    supernode:
      cluster:
        - ip: '192.168.0.1'
          registerPort: 8002
        - ip: '192.168.0.2'
    ```

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
