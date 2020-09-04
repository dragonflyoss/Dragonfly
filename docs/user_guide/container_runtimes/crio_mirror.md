# Use Dfdaemon as Registry Mirror for CRI-O

## Prerequisites

All steps in this document is doing on the same machine using the docker container, so make sure the docker container engine installed and started on your machine. You can also refer to the documentation: [multi-machine deployment](../multi_machines_deployment.md) to experience Dragonfly.

## Step 1: Validate Dragonfly Configuration

To use dfdaemon as Registry Mirror, first you need to ensure configuration in
`/etc/dragonfly/dfdaemon.yml`:

```yaml
registry_mirror:
  # we make a registry mirror to docker.io
  remote: https://index.docker.io
proxies:
- regx: blobs/sha256.*
```

This will proxy all requests for image layers with dfget.

## Step 2: Validate CRI-O Configuration

Then, enable mirrors in CRI-O registries configuration in
`/etc/containers/registries.conf`:

```yaml
[[registry]]
location = "docker.io"
  [[registry.mirror]]
  location = "localhost:65001"
  insecure = true
```

## Step 3: Restart CRI-O Daemon

```
systemctl restart crio
```

If encounter error like these:
`mixing sysregistry v1/v2 is not supported` or `registry must be in v2 format but is in v1`,
please convert your registries configuration to v2.

## Step 4: Pull Image

You can pull image like this:

```
crictl pull docker.io/library/busybox
```

## Step 5: Validate Dragonfly

You can execute the following command to check if the busybox image is distributed via Dragonfly.

```bash
docker exec dfclient grep 'downloading piece' /root/.small-dragonfly/logs/dfclient.log
```

If the output of command above has content like

```
2020-07-23 06:26:03.148 INFO sign:130-1595485563.124 : downloading piece:{"taskID":"cc9ca1e5b24b5f3ab84d75884756c817cdc272f19670d3a25e2ca016af6bd0f7","superNode":"192.168.0.204:8002","dstCid":"","range":"","result":502,"status":700,"pieceSize":0,"pieceNum":0}
```