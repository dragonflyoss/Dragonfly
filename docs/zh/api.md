# API

本节介绍超级节点提供的API。

### 注册

```
POST /peer/registry
```

**参数**

`Content-type: application/x-www-form-urlencoded`

* `cid`: string, 客户端Id
* `ip`: ipv4 string, 客户端ip地址
* `hostName`: string, 客户端主机名
* `superNodeIp`: ipv4 string, 注册的超级节点ip地址
* `port`: integer, 客户端开放的端口号，用于P2P下载
* `callSystem`: string, 调用`dfget`的使用方标识
* `version`: string, 客户端版本
* `dfdaemon`: boolean, 是否由`dfdaemon`启动`dfget`
* `path`: string, 客户端开放的服务路径
* `rawUrl`: string, 由命令行参数指定的原始资源url
* `taskUrl`: string, 资源url
* `md5`: string, 要下载的资源的md5值，可选
* `identifier`: string, 资源标识
* `headers`: map, 需要发送给`rawUrl`的额外HTTP头信息

当客户端发送注册请求时，超级节点会根据请求参数构造一个新的任务实例。每个任务有一个`taskId`，基于参数`rawUrl`，`md5`，`identifier`由超级节点生成。

然后该任务会被保存到内存缓存中。同时，超级节点会获取额外的信息，比如一般都会设置的文件大小(Content-Length)。另外会按照如下策略计算分片大小(pieceSize)：

1. 如果总大小比`200MB`小，则分片大小为默认的`4MB`。
2. 否则，取`(${totalSize} / 100 MB) + 2`MB与`15MB`中的最小值。

之后，`peer`信息和任务会被一起记录下来：
* `peer<ip, cid, hostname>`
* `task<taskId, cid, pieceSize, port, path>`

最后触发一个流程处理后续流程。

**响应**

成功返回的例子：

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

其他情况下的`code`值：
1. `taskId`已经存在, `606`。
2. url不正确，`607`。
3. 需要认证，`608`或者`609`。

### 获取任务信息

```
GET /peer/task
```

**参数**

参数位于请求参数列表中

*   `superNode`: ipv4 string, 超级节点ip地址。
*   `dstCid`: string, 目标客户端Id。
*   `range`: string, byte range.
*   `status`: integer.
*   `result`: integer.
*   `taskId`: string, 任务Id。
*   `srcCid`: string, 源客户端Id。

The super node will analyze the `status` and `result` firstly:

超级节点首先分析`status`和`result`：
* 若`status == 701`，则为`Running`
* 若`status == 702` 且 `result == 501`，则为`Success`
* 若`status == 702` 且 `result == 500`，则为`Fail`
* 若``status == 700``，则为`Wait`

如果是waiting状态，则超级节点会执行：
* 将状态保存为`Running`

在`Running`状态下，超级节点提取分片状态：
* 若`result == 501`，则为`Success`
* 若`result == 500`，则为`Fail`
* 若`result == 503`，则为`SemiSuc`

(注): `result == 502` 是非法状态码

然后超级节点更新此任务的进度，检查任务状态和对等者(peer)的状态。之后，若其他peer有足够信息下载下一个分片，那么超级节点告诉客户端对应peer的信息；否则就会失败。

**响应**

例子：

```json
{
  "code": 602,
  "msg": "client sucCount:0,cdn status:Running,cdn sucCount: 0"
}
```

上面响应内容的意思是，客户端必须等待，因为目前还没有peer可用下载当前的分片。

若有peer可用，则响应如下：

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

**参数**

参数位于请求参数列表中

*   `dstCid`: string, 目标客户端Id。
*   `pieceRange`: byte range.
*   `taskId`: string, 任务Id。
*   `cid`: string, 客户端Id。

**响应**

例子：

```json
{
  "code": 200,
  "msg": "success"
}
```
