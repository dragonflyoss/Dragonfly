# Dragonfly SuperNode API


<a name="overview"></a>
## Overview
API is an HTTP API served by Dragonfly's SuperNode. It is the API dfget or Harbor uses to communicate
with the supernode.


### Version information
*Version* : 0.1


### URI scheme
*Schemes* : HTTP, HTTPS


### Tags

* Peer : Create and manage peer nodes in peer networks.
* Piece : create and manage image/file pieces in supernode.
* PreheatTask : Create and manage image or file preheat task in supernode.
* Task : create and manage image/file distribution task in supernode.


### Consumes

* `application/json`
* `text/plain`


### Produces

* `application/json`
* `text/plain`




<a name="paths"></a>
## Paths

<a name="ping-get"></a>
### Ping
```
GET /_ping
```


#### Description
This is a dummy endpoint you can use to test if the server is accessible.


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|string|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Example HTTP response

##### Response 200
```
json :
"OK"
```


<a name="api-v1-peers-post"></a>
### register dfget in Supernode as a peer node
```
POST /api/v1/peers
```


#### Description
dfget sends request to register in Supernode as a peer node


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Body**|**body**  <br>*optional*|request body which contains peer registrar information.|[PeerCreateRequest](#peercreaterequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**201**|no error|[PeerCreateResponse](#peercreateresponse)|
|**400**|bad parameter|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="api-v1-peers-get"></a>
### get all peers
```
GET /api/v1/peers
```


#### Description
dfget sends request to register in Supernode as a peer node


#### Parameters

|Type|Name|Description|Schema|Default|
|---|---|---|---|---|
|**Query**|**pageNum**  <br>*optional*||integer|`0`|
|**Query**|**pageSize**  <br>*required*||integer||
|**Query**|**sortDirect**  <br>*optional*|Determine the direction of sorting rules|enum (ASC, DESC)|`"ASC"`|
|**Query**|**sortKey**  <br>*optional*|"The keyword used to sort. You can provide multiple keys, if two peers have the same first key, sort by the second key, and so on"|< string > array||


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|< [PeerInfo](#peerinfo) > array|
|**400**|bad parameter|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="api-v1-peers-id-get"></a>
### get a peer in supernode
```
GET /api/v1/peers/{id}
```


#### Description
return low-level information of a peer in supernode.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of peer|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[PeerInfo](#peerinfo)|
|**404**|An unexpected 404 error occurred.|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`


<a name="api-v1-peers-id-delete"></a>
### delete a peer in supernode
```
DELETE /api/v1/peers/{id}
```


#### Description
dfget stops playing a role as a peer in peer network constructed by supernode.
When dfget lasts in five minutes without downloading or uploading task, the uploader of dfget
automatically sends a DELETE /peers/{id} request to supernode.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of peer|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**204**|no error|No Content|
|**404**|no such peer|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="api-v1-preheats-post"></a>
### Create a Preheat Task
```
POST /api/v1/preheats
```


#### Description
Create a preheat task in supernode to first download image/file which is ready.
Preheat action will shorten the period for dfget to get what it wants. In details,
after preheat action finishes downloading image/file to supernode, dfget can send
request to setup a peer-to-peer network immediately.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Body**|**PreheatCreateRequest**  <br>*optional*|request body which contains preheat task creation information|[PreheatCreateRequest](#preheatcreaterequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[PreheatCreateResponse](#preheatcreateresponse)|
|**400**|bad parameter|[Error](#error)|
|**409**|preheat task already exists|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="api-v1-preheats-get"></a>
### List Preheat Tasks
```
GET /api/v1/preheats
```


#### Description
List preheat tasks in supernode of Dragonfly. This API can list all the existing preheat tasks
in supernode. Note, when a preheat is finished after PreheatGCThreshold, it will be GCed, then
this preheat will not be gotten by preheat tasks list API.


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|< [PreheatInfo](#preheatinfo) > array|
|**400**|bad parameter|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="api-v1-preheats-id-get"></a>
### Get a preheat task
```
GET /api/v1/preheats/{id}
```


#### Description
get detailed information of a preheat task in supernode.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of preheat task|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[PreheatInfo](#preheatinfo)|
|**404**|no such preheat task|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`


<a name="api-v1-preheats-id-delete"></a>
### Delete a preheat task
```
DELETE /api/v1/preheats/{id}
```


#### Description
delete a preheat task


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of preheat task|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|No Content|
|**404**|no such preheat task|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`


<a name="api-v1-tasks-post"></a>
### create a task
```
POST /api/v1/tasks
```


#### Description
Create a peer-to-peer downloading task in supernode.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Body**|**body**  <br>*optional*|request body which contains task creation information|[TaskCreateRequest](#taskcreaterequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**201**|no error|[TaskCreateResponse](#taskcreateresponse)|
|**400**|bad parameter|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="api-v1-tasks-id-get"></a>
### get a task
```
GET /api/v1/tasks/{id}
```


#### Description
return low-level information of a task in supernode.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of task|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[TaskInfo](#taskinfo)|
|**404**|An unexpected 404 error occurred.|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`


<a name="api-v1-tasks-id-put"></a>
### update a task
```
PUT /api/v1/tasks/{id}
```


#### Description
Update information of a task.
This endpoint is mainly for operation usage. When the peer network or peer
meet some load issues, operation team can update a task directly, such as pause
a downloading task to ease the situation.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of task|string|
|**Body**|**TaskUpdateRequest**  <br>*optional*|request body which contains task update information"|[TaskUpdateRequest](#taskupdaterequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|No Content|
|**404**|An unexpected 404 error occurred.|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Consumes

* `application/json`


#### Produces

* `application/json`


<a name="api-v1-tasks-id-delete"></a>
### delete a task
```
DELETE /api/v1/tasks/{id}
```


#### Description
delete a peer-to-peer task in supernode.
This endpoint is mainly for operation usage. When the peer network or peer
meet some load issues, operation team can delete a task directly to ease
the situation.


#### Parameters

|Type|Name|Description|Schema|Default|
|---|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of task|string||
|**Query**|**full**  <br>*optional*|supernode will also delete the cdn files when the value of full equals true.|boolean|`"false"`|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**204**|no error|No Content|
|**404**|no such task|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="api-v1-tasks-id-pieces-get"></a>
### Get pieces in task
```
GET /api/v1/tasks/{id}/pieces
```


#### Description
When dfget starts to download pieces of a task, it should get fixed
number of pieces in a task and then use pieces information to download
the pieces. The request piece number is set in query.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of task|string|
|**Query**|**clientID**  <br>*required*|When dfget needs to get pieces of specific task, it must mark which peer it plays role of.|string|
|**Query**|**num**  <br>*optional*|Request number of pieces of task. If request number is larger than the total pieces in supernode,<br>supernode returns the total pieces of task. If not set, supernode will set 4 by default.|integer (int64)|
|**Body**|**PiecePullRequest**  <br>*required*|request body which contains the information of pieces that have been downloaded or being downloaded.|[PiecePullRequest](#piecepullrequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|< [PieceInfo](#pieceinfo) > array|
|**404**|no such task|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`


<a name="api-v1-tasks-id-pieces-piecerange-put"></a>
### Update a piece
```
PUT /api/v1/tasks/{id}/pieces/{pieceRange}
```


#### Description
Update some information of piece. When peer A finishes to download
piece B, A must send request to supernode to update piece B's info
to mark that peer A has the complete piece B. Then when other peers
request to download this piece B, supernode could schedule peer A
to those peers.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of task|string|
|**Path**|**pieceRange**  <br>*required*|the range of specific piece in the task, example "0-45565".|string|
|**Body**|**PieceUpdateRequest**  <br>*optional*|request body which contains task update information.|[PieceUpdateRequest](#pieceupdaterequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|No Content|
|**404**|An unexpected 404 error occurred.|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Consumes

* `application/json`


#### Produces

* `application/json`


<a name="api-v1-tasks-id-pieces-piecerange-error-post"></a>
### report a piece error
```
POST /api/v1/tasks/{id}/pieces/{pieceRange}/error
```


#### Description
When a peer failed to download a piece from supernode or
failed to validate the pieceMD5,
and then dfget should report the error info to supernode.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of task|string|
|**Path**|**pieceRange**  <br>*required*|the range of specific piece in the task, example "0-45565".|string|
|**Body**|**PieceErrorRequest**  <br>*optional*|request body which contains piece error information.|[PieceErrorRequest](#pieceerrorrequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|No Content|
|**404**|An unexpected 404 error occurred.|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Consumes

* `application/json`


#### Produces

* `application/json`


<a name="metrics-get"></a>
### Get Prometheus metrics
```
GET /metrics
```


#### Description
Get Prometheus metrics


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|string|


#### Example HTTP response

##### Response 200
```
json :
"go_goroutines 1"
```


<a name="peer-heartbeat-post"></a>
### report the heart beat to super node.
```
POST /peer/heartbeat
```


#### Description
This endpoint is mainly for reporting the heart beat to supernode.
And supernode could know if peer is alive in strem mode.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Body**|**body**  <br>*optional*|request body which contains base info of peer|[HeartBeatRequest](#heartbeatrequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[HeartBeatResponse](#heartbeatresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="peer-network-post"></a>
### peer request the p2p network info from supernode.
```
POST /peer/network
```


#### Description
In the new mode which dfdaemon will provide the seed file so that other peers
could download. This endpoint is mainly for fetching the p2p network info.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Body**|**body**  <br>*optional*|request body which filter urls.|[NetworkInfoFetchRequest](#networkinfofetchrequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[NetworkInfoFetchResponse](#networkinfofetchresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="peer-piece-suc-get"></a>
### report a piece has been success
```
GET /peer/piece/suc
```


#### Description
Update some information of piece. When peer A finishes to download
piece B, A must send request to supernode to update piece B's info
to mark that peer A has the complete piece B. Then when other peers
request to download this piece B, supernode could schedule peer A
to those peers.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Query**|**cid**  <br>*required*|the downloader clientID|string|
|**Query**|**dstCid**  <br>*optional*|the uploader peerID|string|
|**Query**|**pieceRange**  <br>*required*|the range of specific piece in the task, example "0-45565".|string|
|**Query**|**taskId**  <br>*required*|ID of task|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[ResultInfo](#resultinfo)|
|**404**|An unexpected 404 error occurred.|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`


<a name="peer-registry-post"></a>
### registry a task
```
POST /peer/registry
```


#### Description
Create a peer-to-peer downloading task in supernode.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Body**|**body**  <br>*optional*|request body which contains task creation information|[TaskRegisterRequest](#taskregisterrequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[ResultInfo](#resultinfo)|
|**400**|bad parameter|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="peer-service-down-get"></a>
### report a peer service will offline
```
GET /peer/service/down
```


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Query**|**cid**  <br>*required*|the downloader clientID|string|
|**Query**|**taskId**  <br>*required*|ID of task|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[ResultInfo](#resultinfo)|
|**404**|An unexpected 404 error occurred.|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`


<a name="peer-task-get"></a>
### Get pieces in task
```
GET /peer/task
```


#### Description
When dfget starts to download pieces of a task, it should get fixed
number of pieces in a task and then use pieces information to download
the pieces. The request piece number is set in query.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Query**|**dstCid**  <br>*optional*|the uploader cid|string|
|**Query**|**range**  <br>*optional*|the range of specific piece in the task, example "0-45565".|string|
|**Query**|**result**  <br>*optional*|pieceResult It indicates whether the dfgetTask successfully download the piece.<br>It's only useful when `status` is `RUNNING`.|enum (FAILED, SUCCESS, INVALID, SEMISUC)|
|**Query**|**srcCid**  <br>*required*|When dfget needs to get pieces of specific task, it must mark which peer it plays role of.|string|
|**Query**|**status**  <br>*optional*|dfgetTaskStatus indicates whether the dfgetTask is running.|enum (STARTED, RUNNING, FINISHED)|
|**Query**|**taskId**  <br>*required*|ID of task|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[ResultInfo](#resultinfo)|
|**404**|no such task|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`


<a name="task-metrics-post"></a>
### upload dfclient download metrics
```
POST /task/metrics
```


#### Description
This endpoint is mainly for observability. Dfget is a short-live job
and we use this endpoint to upload dfget download related metrics.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Body**|**body**  <br>*optional*|request body which contains dfget download related information|[TaskMetricsRequest](#taskmetricsrequest)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[ResultInfo](#resultinfo)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="version-get"></a>
### Get version and build information
```
GET /version
```


#### Description
Get version and build information, including GoVersion, OS,
Arch, Version, BuildDate, and GitCommit.


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|[DragonflyVersion](#dragonflyversion)|
|**500**|An unexpected server error occurred.|[Error](#error)|




<a name="definitions"></a>
## Definitions

<a name="cdnsource"></a>
### CdnSource
*Type* : enum (supernode, source)


<a name="dfgettask"></a>
### DfGetTask
A download process initiated by dfget or other clients.


|Name|Description|Schema|
|---|---|---|
|**cID**  <br>*optional*|CID means the client ID. It maps to the specific dfget process.<br>When user wishes to download an image/file, user would start a dfget process to do this.<br>This dfget is treated a client and carries a client ID.<br>Thus, multiple dfget processes on the same peer have different CIDs.|string|
|**callSystem**  <br>*optional*|This attribute represents where the dfget requests come from. Dfget will pass<br>this field to supernode and supernode can do some checking and filtering via<br>black/white list mechanism to guarantee security, or some other purposes like debugging.  <br>**Minimum length** : `1`|string|
|**dfdaemon**  <br>*optional*|tells whether it is a call from dfdaemon. dfdaemon is a long running<br>process which works for container engines. It translates the image<br>pulling request into raw requests into those dfget recognizes.|boolean|
|**path**  <br>*optional*|path is used in one peer A for uploading functionality. When peer B hopes<br>to get piece C from peer A, B must provide a URL for piece C.<br>Then when creating a task in supernode, peer A must provide this URL in request.|string|
|**peerID**  <br>*optional*|PeerID uniquely identifies a peer, and the cID uniquely identifies a<br>download task belonging to a peer. One peer can initiate multiple download tasks,<br>which means that one peer corresponds to multiple cIDs.|string|
|**pieceSize**  <br>*optional*|The size of pieces which is calculated as per the following strategy<br>1. If file's total size is less than 200MB, then the piece size is 4MB by default.<br>2. Otherwise, it equals to the smaller value between totalSize/100MB + 2 MB and 15MB.|integer (int32)|
|**status**  <br>*optional*|The status of Dfget download process.|enum (WAITING, RUNNING, FAILED, SUCCESS)|
|**supernodeIP**  <br>*optional*|IP address of supernode which the peer connects to|string|
|**taskId**  <br>*optional*||string|


<a name="dragonflyversion"></a>
### DragonflyVersion
Version and build information of Dragonfly components.


|Name|Description|Schema|
|---|---|---|
|**Arch**  <br>*optional*|Dragonfly components's architecture target|string|
|**BuildDate**  <br>*optional*|Build Date of Dragonfly components|string|
|**GoVersion**  <br>*optional*|Golang runtime version|string|
|**OS**  <br>*optional*|Dragonfly components's operating system|string|
|**Revision**  <br>*optional*|Git commit when building Dragonfly components|string|
|**Version**  <br>*optional*|Version of Dragonfly components|string|


<a name="error"></a>
### Error

|Name|Schema|
|---|---|
|**message**  <br>*optional*|string|


<a name="errorresponse"></a>
### ErrorResponse
It contains a code that identify which error occurred for client processing and a detailed error message to read.


|Name|Description|Schema|
|---|---|---|
|**code**  <br>*optional*|the code of this error, it's convenient for client to process with certain error.|integer|
|**message**  <br>*optional*|detailed error message|string|


<a name="heartbeatrequest"></a>
### HeartBeatRequest
The request is to report peer to supernode to keep alive.


|Name|Description|Schema|
|---|---|---|
|**IP**  <br>*optional*|IP address which peer client carries|string (ipv4)|
|**cID**  <br>*optional*|CID means the client ID. It maps to the specific dfget process.<br>When user wishes to download an image/file, user would start a dfget process to do this.<br>This dfget is treated a client and carries a client ID.<br>Thus, multiple dfget processes on the same peer have different CIDs.|string|
|**port**  <br>*optional*|when registering, dfget will setup one uploader process.<br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.  <br>**Minimum value** : `15000`  <br>**Maximum value** : `65000`|integer (int32)|


<a name="heartbeatresponse"></a>
### HeartBeatResponse

|Name|Description|Schema|
|---|---|---|
|**needRegister**  <br>*optional*|If peer do not register in supernode, set needRegister to be true, else set to be false.|boolean|
|**seedTaskIDs**  <br>*optional*|The array of seed taskID which now are selected as seed for the peer. If peer have other seed file which<br>is not included in the array, these seed file should be weed out.|< string > array|
|**version**  <br>*optional*|The version of supernode. If supernode restarts, version should be different, so dfdaemon could know<br>the restart of supernode.|string|


<a name="networkinfofetchrequest"></a>
### NetworkInfoFetchRequest
The request is to fetch p2p network info from supernode.


|Name|Description|Schema|
|---|---|---|
|**urls**  <br>*optional*|The urls is to filter the peer node, the url should be match with taskURL in TaskInfo.|< string > array|


<a name="networkinfofetchresponse"></a>
### NetworkInfoFetchResponse
The response is from supernode to peer which is requested to fetch p2p network info.


|Name|Schema|
|---|---|
|**nodes**  <br>*optional*|< [Node](#node) > array|


<a name="node"></a>
### Node
The object shows the basic info of node and the task belongs to the node.


|Name|Description|Schema|
|---|---|---|
|**basic**  <br>*optional*|Basic node info|[PeerInfo](#peerinfo)|
|**extra**  <br>*optional*||< string, string > map|
|**load**  <br>*optional*|The load of node, which as the schedule weight in peer schedule.|integer|
|**tasks**  <br>*optional*||< [TaskFetchInfo](#taskfetchinfo) > array|


<a name="peercreaterequest"></a>
### PeerCreateRequest
PeerCreateRequest is used to create a peer instance in supernode.
Usually, when dfget is going to register in supernode as a peer,
it will send PeerCreateRequest to supernode.


|Name|Description|Schema|
|---|---|---|
|**IP**  <br>*optional*|IP address which peer client carries|string (ipv4)|
|**hostName**  <br>*optional*|host name of peer client node, as a valid RFC 1123 hostname.  <br>**Minimum length** : `1`|string (hostname)|
|**port**  <br>*optional*|when registering, dfget will setup one uploader process.<br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.  <br>**Minimum value** : `15000`  <br>**Maximum value** : `65000`|integer (int32)|
|**version**  <br>*optional*|version number of dfget binary.|string|


<a name="peercreateresponse"></a>
### PeerCreateResponse
ID of created peer.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|Peer ID of the node which dfget locates on.<br>Every peer has a unique ID among peer network.<br>It is generated via host's hostname and IP address.|string|


<a name="peerinfo"></a>
### PeerInfo
The detailed information of a peer in supernode.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of peer|string|
|**IP**  <br>*optional*|IP address which peer client carries.<br>(TODO) make IP field contain more information, for example<br>WAN/LAN IP address for supernode to recognize.|string (ipv4)|
|**created**  <br>*optional*|the time to join the P2P network|string (date-time)|
|**hostName**  <br>*optional*|host name of peer client node, as a valid RFC 1123 hostname.  <br>**Minimum length** : `1`|string (hostname)|
|**port**  <br>*optional*|when registering, dfget will setup one uploader process.<br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.  <br>**Minimum value** : `15000`  <br>**Maximum value** : `65000`|integer (int32)|
|**version**  <br>*optional*|version number of dfget binary|string|


<a name="pieceerrorrequest"></a>
### PieceErrorRequest
Peer's detailed information in supernode.


|Name|Description|Schema|
|---|---|---|
|**dstIP**  <br>*optional*|the peer ID of the target Peer.|string|
|**dstPid**  <br>*optional*|the peer ID of the target Peer.|string|
|**errorType**  <br>*optional*|the error type when failed to download from supernode that dfget will report to supernode|enum (FILE_NOT_EXIST, FILE_MD5_NOT_MATCH)|
|**expectedMd5**  <br>*optional*|the MD5 value of piece which returned by the supernode that<br>in order to verify the correctness of the piece content which<br>downloaded from the other peers.|string|
|**range**  <br>*optional*|the range of specific piece in the task, example "0-45565".|string|
|**realMd5**  <br>*optional*|the MD5 information of piece which calculated by the piece content<br>which downloaded from the target peer.|string|
|**srcCid**  <br>*optional*|the CID of the src Peer.|string|
|**taskId**  <br>*optional*|the taskID of the piece.|string|


<a name="pieceinfo"></a>
### PieceInfo
Peer's detailed information in supernode.


|Name|Description|Schema|
|---|---|---|
|**pID**  <br>*optional*|the peerID that dfget task should download from|string|
|**path**  <br>*optional*|The URL path to download the specific piece from the target peer's uploader.|string|
|**peerIP**  <br>*optional*|When dfget needs to download a piece from another peer. Supernode will return a PieceInfo<br>that contains a peerIP. This peerIP represents the IP of this dfget's target peer.|string|
|**peerPort**  <br>*optional*|When dfget needs to download a piece from another peer. Supernode will return a PieceInfo<br>that contains a peerPort. This peerPort represents the port of this dfget's target peer's uploader.|integer (int32)|
|**pieceMD5**  <br>*optional*|the MD5 information of piece which is generated by supernode when doing CDN cache.<br>This value will be returned to dfget in order to validate the piece's completeness.|string|
|**pieceRange**  <br>*optional*|the range of specific piece in the task, example "0-45565".|string|
|**pieceSize**  <br>*optional*|The size of pieces which is calculated as per the following strategy<br>1. If file's total size is less than 200MB, then the piece size is 4MB by default.<br>2. Otherwise, it equals to the smaller value between totalSize/100MB + 2 MB and 15MB.|integer (int32)|


<a name="piecepullrequest"></a>
### PiecePullRequest
request used to pull pieces that have not been downloaded.


|Name|Description|Schema|
|---|---|---|
|**dfgetTaskStatus**  <br>*optional*|dfgetTaskStatus indicates whether the dfgetTask is running.|enum (STARTED, RUNNING, FINISHED)|
|**dstPID**  <br>*optional*|the uploader peerID|string|
|**pieceRange**  <br>*optional*|the range of specific piece in the task, example "0-45565".|string|
|**pieceResult**  <br>*optional*|pieceResult It indicates whether the dfgetTask successfully download the piece.<br>It's only useful when `status` is `RUNNING`.|enum (FAILED, SUCCESS, INVALID, SEMISUC)|


<a name="pieceupdaterequest"></a>
### PieceUpdateRequest
request used to update piece attributes.


|Name|Description|Schema|
|---|---|---|
|**clientID**  <br>*optional*|the downloader clientID|string|
|**dstPID**  <br>*optional*|the uploader peerID|string|
|**pieceStatus**  <br>*optional*|pieceStatus indicates whether the peer task successfully download the piece.|enum (FAILED, SUCCESS, INVALID, SEMISUC)|


<a name="preheatcreaterequest"></a>
### PreheatCreateRequest
Request option of creating a preheat task in supernode.


|Name|Description|Schema|
|---|---|---|
|**filter**  <br>*optional*|URL may contains some changeful query parameters such as authentication parameters. Dragonfly will<br>filter these parameter via 'filter'. The usage of it is that different URL may generate the same<br>download taskID.|string|
|**headers**  <br>*optional*|If there is any authentication step of the remote server, the headers should contains authenticated information.<br>Dragonfly will sent request taking the headers to remote server.|< string, string > map|
|**identifier**  <br>*optional*|This field is used for generating new downloading taskID to identify different downloading task of remote URL.|string|
|**type**  <br>*required*|this must be image or file|enum (image, file)|
|**url**  <br>*required*|the image or file location  <br>**Minimum length** : `3`|string|


<a name="preheatcreateresponse"></a>
### PreheatCreateResponse
Response of a preheat creation request.


|Name|Schema|
|---|---|
|**ID**  <br>*optional*|string|


<a name="preheatinfo"></a>
### PreheatInfo
return detailed information of a preheat task in supernode. An image preheat task may contain multiple downloading
task because that an image may have more than one layer.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of preheat task.|string|
|**finishTime**  <br>*optional*|the preheat task finish time|string (date-time)|
|**startTime**  <br>*optional*|the preheat task start time|string (date-time)|
|**status**  <br>*optional*|The status of preheat task.<br>  WAITING -----> RUNNING -----> SUCCESS<br>                           \|--> FAILED<br>The initial status of a created preheat task is WAITING.<br>It's finished when a preheat task's status is FAILED or SUCCESS.<br>A finished preheat task's information can be queried within 24 hours.|[PreheatStatus](#preheatstatus)|


<a name="preheatstatus"></a>
### PreheatStatus
The status of preheat task.
  WAITING -----> RUNNING -----> SUCCESS
                           |--> FAILED
The initial status of a created preheat task is WAITING.
It's finished when a preheat task's status is FAILED or SUCCESS.
A finished preheat task's information can be queried within 24 hours.

*Type* : enum (WAITING, RUNNING, FAILED, SUCCESS)


<a name="resultinfo"></a>
### ResultInfo
The returned information from supernode.


|Name|Description|Schema|
|---|---|---|
|**code**  <br>*optional*|the result code|integer (int32)|
|**data**  <br>*optional*|the result data|object|
|**msg**  <br>*optional*|the result msg|string|


<a name="taskcreaterequest"></a>
### TaskCreateRequest

|Name|Description|Schema|
|---|---|---|
|**cID**  <br>*optional*|CID means the client ID. It maps to the specific dfget process.<br>When user wishes to download an image/file, user would start a dfget process to do this.<br>This dfget is treated a client and carries a client ID.<br>Thus, multiple dfget processes on the same peer have different CIDs.|string|
|**callSystem**  <br>*optional*|This attribute represents where the dfget requests come from. Dfget will pass<br>this field to supernode and supernode can do some checking and filtering via<br>black/white list mechanism to guarantee security, or some other purposes like debugging.  <br>**Minimum length** : `1`|string|
|**dfdaemon**  <br>*optional*|tells whether it is a call from dfdaemon. dfdaemon is a long running<br>process which works for container engines. It translates the image<br>pulling request into raw requests into those dfget recognizes.|boolean|
|**fileLength**  <br>*optional*|This attribute represents the length of resource, dfdaemon or dfget catches and calculates<br>this parameter from the headers of request URL. If fileLength is vaild, the supernode need<br>not get the length of resource by accessing the rawURL.|integer (int64)|
|**filter**  <br>*optional*|filter is used to filter request queries in URL.<br>For example, when a user wants to start to download a task which has a remote URL of<br>a.b.com/fileA?user=xxx&auth=yyy, user can add a filter parameter ["user", "auth"]<br>to filter the url to a.b.com/fileA. Then this parameter can potentially avoid repeatable<br>downloads, if there is already a task a.b.com/fileA.|< string > array|
|**headers**  <br>*optional*|extra HTTP headers sent to the rawURL.<br>This field is carried with the request to supernode.<br>Supernode will extract these HTTP headers, and set them in HTTP downloading requests<br>from source server as user's wish.|< string, string > map|
|**identifier**  <br>*optional*|special attribute of remote source file. This field is used with taskURL to generate new taskID to<br>identify different downloading task of remote source file. For example, if user A and user B uses<br>the same taskURL and taskID to download file, A and B will share the same peer network to distribute files.<br>If user A additionally adds an identifier with taskURL, while user B still carries only taskURL, then A's<br>generated taskID is different from B, and the result is that two users use different peer networks.|string|
|**md5**  <br>*optional*|md5 checksum for the resource to distribute. dfget catches this parameter from dfget's CLI<br>and passes it to supernode. When supernode finishes downloading file/image from the source location,<br>it will validate the source file with this md5 value to check whether this is a valid file.|string|
|**path**  <br>*optional*|path is used in one peer A for uploading functionality. When peer B hopes<br>to get piece C from peer A, B must provide a URL for piece C.<br>Then when creating a task in supernode, peer A must provide this URL in request.|string|
|**peerID**  <br>*optional*|PeerID is used to uniquely identifies a peer which will be used to create a dfgetTask.<br>The value must be the value in the response after registering a peer.|string|
|**rawURL**  <br>*optional*|The is the resource's URL which user uses dfget to download. The location of URL can be anywhere, LAN or WAN.<br>For image distribution, this is image layer's URL in image registry.<br>The resource url is provided by command line parameter.|string|
|**supernodeIP**  <br>*optional*|IP address of supernode which the peer connects to|string|
|**taskId**  <br>*optional*|This attribute represents the digest of resource, dfdaemon or dfget catches this parameter<br>from the headers of request URL. The digest will be considered as the taskID if not null.|string|
|**taskURL**  <br>*optional*|taskURL is generated from rawURL. rawURL may contains some queries or parameter, dfget will filter some queries via<br>--filter parameter of dfget. The usage of it is that different rawURL may generate the same taskID.|string|


<a name="taskcreateresponse"></a>
### TaskCreateResponse
response get from task creation request.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of the created task.|string|
|**cdnSource**  <br>*optional*||[CdnSource](#cdnsource)|
|**fileLength**  <br>*optional*|The length of the file dfget requests to download in bytes.|integer (int64)|
|**pieceSize**  <br>*optional*|The size of pieces which is calculated as per the following strategy<br>1. If file's total size is less than 200MB, then the piece size is 4MB by default.<br>2. Otherwise, it equals to the smaller value between totalSize/100MB + 2 MB and 15MB.|integer (int32)|


<a name="taskfetchinfo"></a>
### TaskFetchInfo
It shows the task info and pieces info.


|Name|Description|Schema|
|---|---|---|
|**pieces**  <br>*optional*|The pieces which should belong to the peer node|< [PieceInfo](#pieceinfo) > array|
|**task**  <br>*optional*||[TaskInfo](#taskinfo)|


<a name="taskinfo"></a>
### TaskInfo
detailed information about task in supernode.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of the task.|string|
|**asSeed**  <br>*optional*|This attribute represents the node as a seed node for the taskURL.|boolean|
|**cdnStatus**  <br>*optional*|The status of the created task related to CDN functionality.|enum (WAITING, RUNNING, FAILED, SUCCESS, SOURCE_ERROR)|
|**fileLength**  <br>*optional*|The length of the file dfget requests to download in bytes<br>which including the header and the trailer of each piece.|integer (int64)|
|**headers**  <br>*optional*|extra HTTP headers sent to the rawURL.<br>This field is carried with the request to supernode.<br>Supernode will extract these HTTP headers, and set them in HTTP downloading requests<br>from source server as user's wish.|< string, string > map|
|**httpFileLength**  <br>*optional*|The length of the source file in bytes.|integer (int64)|
|**identifier**  <br>*optional*|special attribute of remote source file. This field is used with taskURL to generate new taskID to<br>identify different downloading task of remote source file. For example, if user A and user B uses<br>the same taskURL and taskID to download file, A and B will share the same peer network to distribute files.<br>If user A additionally adds an identifier with taskURL, while user B still carries only taskURL, then A's<br>generated taskID is different from B, and the result is that two users use different peer networks.|string|
|**md5**  <br>*optional*|md5 checksum for the resource to distribute. dfget catches this parameter from dfget's CLI<br>and passes it to supernode. When supernode finishes downloading file/image from the source location,<br>it will validate the source file with this md5 value to check whether this is a valid file.|string|
|**pieceSize**  <br>*optional*|The size of pieces which is calculated as per the following strategy<br>1. If file's total size is less than 200MB, then the piece size is 4MB by default.<br>2. Otherwise, it equals to the smaller value between totalSize/100MB + 2 MB and 15MB.|integer (int32)|
|**pieceTotal**  <br>*optional*||integer (int32)|
|**rawURL**  <br>*optional*|The is the resource's URL which user uses dfget to download. The location of URL can be anywhere, LAN or WAN.<br>For image distribution, this is image layer's URL in image registry.<br>The resource url is provided by command line parameter.|string|
|**realMd5**  <br>*optional*|when supernode finishes downloading file/image from the source location,<br>the md5 sum of the source file will be calculated as the value of the realMd5.<br>And it will be used to compare with md5 value to check whether this is a valid file.|string|
|**taskURL**  <br>*optional*|taskURL is generated from rawURL. rawURL may contains some queries or parameter, dfget will filter some queries via<br>--filter parameter of dfget. The usage of it is that different rawURL may generate the same taskID.|string|


<a name="taskmetricsrequest"></a>
### TaskMetricsRequest

|Name|Description|Schema|
|---|---|---|
|**IP**  <br>*optional*|IP address which peer client carries|string (string)|
|**backsourceReason**  <br>*optional*|when registering, dfget will setup one uploader process.<br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.|string|
|**cID**  <br>*optional*|CID means the client ID. It maps to the specific dfget process.<br>When user wishes to download an image/file, user would start a dfget process to do this.<br>This dfget is treated a client and carries a client ID.<br>Thus, multiple dfget processes on the same peer have different CIDs.|string|
|**callSystem**  <br>*optional*|This attribute represents where the dfget requests come from. Dfget will pass<br>this field to supernode and supernode can do some checking and filtering via<br>black/white list mechanism to guarantee security, or some other purposes like debugging.  <br>**Minimum length** : `1`|string|
|**duration**  <br>*optional*|Duration for dfget task.|number (float64)|
|**fileLength**  <br>*optional*|The length of the file dfget requests to download in bytes.|integer (int64)|
|**port**  <br>*optional*|when registering, dfget will setup one uploader process.<br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.  <br>**Minimum value** : `15000`  <br>**Maximum value** : `65000`|integer (int32)|
|**success**  <br>*optional*|whether the download task success or not|boolean|
|**taskId**  <br>*optional*|IP address which peer client carries|string (string)|


<a name="taskregisterrequest"></a>
### TaskRegisterRequest

|Name|Description|Schema|
|---|---|---|
|**IP**  <br>*optional*|IP address which peer client carries|string (ipv4)|
|**asSeed**  <br>*optional*|This attribute represents the node as a seed node for the taskURL.|boolean|
|**cID**  <br>*optional*|CID means the client ID. It maps to the specific dfget process.<br>When user wishes to download an image/file, user would start a dfget process to do this.<br>This dfget is treated a client and carries a client ID.<br>Thus, multiple dfget processes on the same peer have different CIDs.|string|
|**callSystem**  <br>*optional*|This attribute represents where the dfget requests come from. Dfget will pass<br>this field to supernode and supernode can do some checking and filtering via<br>black/white list mechanism to guarantee security, or some other purposes like debugging.  <br>**Minimum length** : `1`|string|
|**dfdaemon**  <br>*optional*|tells whether it is a call from dfdaemon. dfdaemon is a long running<br>process which works for container engines. It translates the image<br>pulling request into raw requests into those dfget recognizes.|boolean|
|**fileLength**  <br>*optional*|This attribute represents the length of resource, dfdaemon or dfget catches and calculates<br>this parameter from the headers of request URL. If fileLength is vaild, the supernode need<br>not get the length of resource by accessing the rawURL.|integer (int64)|
|**headers**  <br>*optional*|extra HTTP headers sent to the rawURL.<br>This field is carried with the request to supernode.<br>Supernode will extract these HTTP headers, and set them in HTTP downloading requests<br>from source server as user's wish.|< string > array|
|**hostName**  <br>*optional*|host name of peer client node.  <br>**Minimum length** : `1`|string|
|**identifier**  <br>*optional*|special attribute of remote source file. This field is used with taskURL to generate new taskID to<br>identify different downloading task of remote source file. For example, if user A and user B uses<br>the same taskURL and taskID to download file, A and B will share the same peer network to distribute files.<br>If user A additionally adds an identifier with taskURL, while user B still carries only taskURL, then A's<br>generated taskID is different from B, and the result is that two users use different peer networks.|string|
|**insecure**  <br>*optional*|tells whether skip secure verify when supernode download the remote source file.|boolean|
|**md5**  <br>*optional*|md5 checksum for the resource to distribute. dfget catches this parameter from dfget's CLI<br>and passes it to supernode. When supernode finishes downloading file/image from the source location,<br>it will validate the source file with this md5 value to check whether this is a valid file.|string|
|**path**  <br>*optional*|path is used in one peer A for uploading functionality. When peer B hopes<br>to get piece C from peer A, B must provide a URL for piece C.<br>Then when creating a task in supernode, peer A must provide this URL in request.|string|
|**port**  <br>*optional*|when registering, dfget will setup one uploader process.<br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.  <br>**Minimum value** : `15000`  <br>**Maximum value** : `65000`|integer (int32)|
|**rawURL**  <br>*optional*|The is the resource's URL which user uses dfget to download. The location of URL can be anywhere, LAN or WAN.<br>For image distribution, this is image layer's URL in image registry.<br>The resource url is provided by command line parameter.|string|
|**rootCAs**  <br>*optional*|The root ca cert from client used to download the remote source file.|< string (byte) > array|
|**superNodeIp**  <br>*optional*|The address of supernode that the client can connect to|string|
|**taskId**  <br>*optional*|Dfdaemon or dfget could specific the taskID which will represents the key of this resource<br>in supernode.|string|
|**taskURL**  <br>*optional*|taskURL is generated from rawURL. rawURL may contains some queries or parameter, dfget will filter some queries via<br>--filter parameter of dfget. The usage of it is that different rawURL may generate the same taskID.|string|
|**version**  <br>*optional*|version number of dfget binary.|string|


<a name="taskupdaterequest"></a>
### TaskUpdateRequest
request used to update task attributes.


|Name|Description|Schema|
|---|---|---|
|**peerID**  <br>*optional*|ID of the peer which has finished to download the whole task.|string|





