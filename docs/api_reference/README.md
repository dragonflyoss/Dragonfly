+++
title = "APIs Provided by Supernode"
menuTitle = "API Reference"
weight = 50
pre = "<b>5. </b>"
chapter = false
+++

This topic explains how to use the APIs provided by the supernode (the cluster manager).
<!--more-->

## Registration

```
POST /peer/registry
```

### Parameters

Parameters are encodeds as `application/x-www-form-urlencoded`.

| Parameter | Type | Description |
|---|---|---|
| cid | String | The client ID. |
| ip | String (IPv4) | The client IP address. |
| hostName | String | The hostname of the client. |
| superNodeIp | String (IPv4) | The IP address of the registered supernode. |
| port | Integer | The port which the client opens for P2P downloading. |
| callSystem | String | The caller identifier. |
| version | String | The client version. |
| dfdaemon | Boolean | Tells whether it is a call from dfdaemon. |
| path | String | The service path which the client offers. |
| rawUrl | String | The resource URL provided by command line parameter. |
| taskUrl | String | The resource URL. |
| md5 | String | The MD5 checksum for the resource. Optional. |
| identifier | String | The resource identifer. |
| headers | Map | Extra HTTP headers to be sent to the raw URL. |

Upon receiving a request of registration from the client, the supernode constructs a new instance of based on the information provided by parameters. Specifically, it generates a task with `taskId` based on `rawUrl`, `md5`, and `identifier`.

Then the task is stored in the memory cache. Meanwhile, the supernode retrieves extra information such as the `content-length` which normally would be set. The `pieceSize` is calculated as per the following strategy:

1. If the total size is less than 200MB, then the piece size is 4MB by default.
2. Otherwise, it equals to the smaller value between `(${totalSize} / 100MB) + 2` MB and 15MB.

Next, the peer information along with the task will be recorded:

1. The peer<ip, cid, hostname> will be saved.
2. The task<taskId, cid, pieceSize, port, path> will be saved.

The last step is to trigger a progress.

### Response

An example response:

```json
{
  "code": 200,
  "data": {
    "fileLength": 687481,
    "pieceSize": 4194304,
    "taskId": "ba270626349198840d0255de8358b6c93fe6d57d922d036fbf40bcf3499f44a8"
  }
}
```

Other possible values of `code` in the response include:

- `606`: The task ID already exists.
- `607`: The URL is invalid.
- `608` or `609`: Authentication required.

## Get Task

```
GET /peer/task
```

### Parameters

| Parameter | Type | Description |
|---|---|---|
| superNode | String (IPv4) | The IP address of the super node. |
| dstCID | String | The destination client ID. |
| range | String | Byte range. |
| status | Integer |  |
| result | Integer |  |
| taskId | String | The task ID. |
| srcCID | String | The source client ID. |

First, the supernode will analyze the `status` and `result`.

- `Running`: if `status == 701`.
- `Success`: if `status == 702` and `result == 501`.
- `Fail`: if `status == 702` and `result == 500`.
- `Wait`: if `status == 700`.

In `Wait` status, the supernode will save the status to be `Running`.

In `Running` status, the supernode will extract the piece status:

- `Success`: if `result == 501`.
- `Fail`: if `result == 500`.
- `SemiSuc`: if `result == 503`.

**Tip:** `result == 502` suggests invalid code.

Then the supernode updates the progress for this specific task, and checks the status of  the task and the peers. After this, the supernode tells the client which peer has enough details to download another piece. Otherwise, it fails.

### Response

An example resonse:

```json
{
  "code": 602,
  "msg": "client sucCount:0,cdn status:Running,cdn sucCount: 0"
}
```

This example response means that the client has to wait, since no peer can serve this piece now. If there is a peer which can serve this request, the response will be something like:

```json
{
  "code": 601,
  "data": [
    {
      "cid": "cdnnode:10.148.177.242~ba270626349198840d0255de8358b6c93fe6d57d922d036fbf40bcf3499f44a8",
      "downLink": "20480",
      "path": "/qtdown/ba2/ba270626349198840d0255de8358b6c93fe6d57d922d036fbf40bcf3499f44a8",
      "peerIp": "10.148.177.242",
      "peerPort": 8001,
      "pieceMd5": "d78ef0af9e95e880fa583b41cf5ad791:687486",
      "pieceNum": 0,
      "pieceSize": 4194304,
      "range": "0-4194303"
    }
  ]
}
```

## Peer Progress

```
GET /peer/piece/suc
```

### Parameters

| Parameter | Type | Description |
|---|---|---|
| dstCID | String | The destination client ID. |
| pieceRange | String | Byte range. |
| taskId | String | The task ID. |
| cID | String | The client ID. |

### Response

An example response:

```json
{
  "code": 200,
  "msg": "success"
}
```