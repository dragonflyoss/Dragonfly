---
title: "APIs Provided by Supernode"
weight: 1
---

This topic explains how to use the APIs provided by Supernode.
<!--more-->

This section describes APIs that is provided by the supernode, aka cluster manager.

### Registration

```
POST /peer/registry
```

**Parameters**

Parameters are encodeds as `application/x-www-form-urlencoded`.

*   `cid`: string, the client id.
*   `ip`: ipv4 string, the client ip address.
*   `hostName`: string, the host name of client node.
*   `superNodeIp`: ipv4 string, the ip address of super node.
*   `port`: integer, the port which client opens.
*   `callSystem`: string, the caller identifier.
*   `version`: string, client version.
*   `dfdaemon`: boolean, tells whether it is a call from dfdaemon.
*   `path`: string, the path which client can serve.
*   `rawUrl`: string, the resource url provided by command line parameter.
*   `taskUrl`: string, the resource url.
*   `md5`: string, the md5 checksum for the resource, optional.
*   `identifier`: string, identifer for the resource.
*   `headers`: map, extra http headers sent to the raw url.

Under the scene, the cluster manager creates a new instance of Task, which is built
from the information provided by parameters. Specifically, it generates a `taskId`
based on `rawUrl`, `md5` and `identifier`.

Then the task would be saved in the memory state. At the same time, it will fetch
extra information like the `content-length` which would generally be set. Also,
the `pieceSize` is computed with the strategy below:

1.  if the total size is less than 200MB, the piece size would be 4MB by default.
1.  otherwise, the minimum of `(${totalSize} / 100MB) + 2`MB and 15MB.

The next step is, the peer information along with the task will be recorded:

1.  The peer<ip, cid, hostname> will be saved.
1.  The task<taskId, cid, pieceSize, port, path> will be saved.

The last step is about triggering a progress.

**Response**

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

Other cases could happen:

1.  the task id might duplicate with an existing one. `606`
1.  the url might be invalid. `607`
1.  the access requires authentication. `608` or `609`

### Get Task

```
GET /peer/task
```

**Parameters**

Parameters are encoded as query string.

*   `superNode`: ipv4 string, the ip address of super node.
*   `dstCid`: string, destination client id.
*   `range`: string, byte range.
*   `status`: integer.
*   `result`: integer.
*   `taskId`: string, the task id.
*   `srcCid`: string, the source client id.

The super node will analyze the `status` and `result` firstly:

*   `Running` if `status == 701`.
*   `Success` if `status == 702` and `result == 501`.
*   `Fail` if `status == 702` and `result == 500`.
*   `Wait` if `status == 700`.

In waiting status, the super node would:

1.  Save the status to be running.

In running status, the super node would will extract the piece status:

*   `Success` if `result == 501`.
*   `Fail` if `result == 500`.
*   `SemiSuc` if `result == 503`.

(side note): `result == 502` means invalid code.

And update the progress for this specific task. Then it checks the status itself
and also the peer status, after that, the super node will tell the client another
task which has enough detail for the next piece, or fails if no one is available.

**Response**

An example resonse:

```json
{
  "code": 602,
  "msg": "client sucCount:0,cdn status:Running,cdn sucCount: 0"
}
```

This means the client has to wait, since no peer can serve this piece now. And if
there is a peer which can serve this request, there will be a response like this:

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


### Peer Progress

```
GET /peer/piece/suc
```

**Parameters**

Parameters are encoded as query string.

*   `dstCid`: string, the destination client id.
*   `pieceRange`: byte range.
*   `taskId`: string, the task id.
*   `cid`: string, the client id.

**Response**

An example response:

```json
{
  "code": 200,
  "msg": "success"
}
```
