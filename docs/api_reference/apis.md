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


<a name="peers-post"></a>
### register dfget in Supernode as a peer node
```
POST /peers
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


<a name="peers-get"></a>
### get all peers
```
GET /peers
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
|**201**|no error|< [PeerInfo](#peerinfo) > array|
|**400**|bad parameter|[Error](#error)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="peers-id-get"></a>
### get a peer in supernode
```
GET /peers/{id}
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


<a name="peers-id-delete"></a>
### delete a peer in supernode
```
DELETE /peers/{id}
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


<a name="preheats-post"></a>
### Create a Preheat Task
```
POST /preheats
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
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="preheats-get"></a>
### List Preheat Tasks
```
GET /preheats
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


<a name="preheats-id-get"></a>
### Get a preheat task
```
GET /preheats/{id}
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


<a name="tasks-post"></a>
### create a task
```
POST /tasks
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


<a name="tasks-id-get"></a>
### get a task
```
GET /tasks/{id}
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


<a name="tasks-id-put"></a>
### update a task
```
PUT /tasks/{id}
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


<a name="tasks-id-delete"></a>
### delete a task
```
DELETE /tasks/{id}
```


#### Description
delete a peer-to-peer task in supernode.
This endpoint is mainly for operation usage. When the peer network or peer
meet some load issues, operation team can delete a task directly to ease
the situation.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of task|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**204**|no error|No Content|
|**404**|no such task|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="tasks-id-pieces-get"></a>
### Get pieces in task
```
GET /tasks/{id}/pieces
```


#### Description
When dfget starts to download pieces of a task, it should get fixed
number of pieces in a task and the use pieces information to download
the pirces. The request piece number is set in query.


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


<a name="tasks-id-pieces-piecerange-put"></a>
### Update a piece
```
PUT /tasks/{id}/pieces/{pieceRange}
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




<a name="definitions"></a>
## Definitions

<a name="dfgettask"></a>
### DfGetTask
A download process initiated by dfget or other clients.


|Name|Description|Schema|
|---|---|---|
|**cID**  <br>*optional*|CID means the client ID. It maps to the specific dfget process. <br>When user wishes to download an image/file, user would start a dfget process to do this. <br>This dfget is treated a client and carries a client ID. <br>Thus, multiple dfget processes on the same peer have different CIDs.|string|
|**path**  <br>*optional*|path is used in one peer A for uploading functionality. When peer B hopes<br>to get piece C from peer A, B must provide a URL for piece C.<br>Then when creating a task in supernode, peer A must provide this URL in request.|string|
|**peerID**  <br>*optional*|PeerID uniquely identifies a peer, and the cID uniquely identifies a <br>download task belonging to a peer. One peer can initiate multiple download tasks, <br>which means that one peer corresponds to multiple cIDs.|string|
|**pieceSize**  <br>*optional*|The size of pieces which is calculated as per the following strategy<br>1. If file's total size is less than 200MB, then the piece size is 4MB by default.<br>2. Otherwise, it equals to the smaller value between totalSize/100MB + 2 MB and 15MB.|integer (int32)|
|**status**  <br>*optional*|The status of Dfget download process.|enum (WAITING, RUNNING, FAILED, SUCCESS)|
|**taskId**  <br>*optional*||string|


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


<a name="peercreaterequest"></a>
### PeerCreateRequest
PeerCreateRequest is used to create a peer instance in supernode.
Usually, when dfget is going to register in supernode as a peer,
it will send PeerCreateRequest to supernode.


|Name|Description|Schema|
|---|---|---|
|**IP**  <br>*optional*|IP address which peer client carries|string (ipv4)|
|**hostName**  <br>*optional*|host name of peer client node, as a valid RFC 1123 hostname.  <br>**Minimum length** : `1`|string (hostname)|
|**port**  <br>*optional*|when registering, dfget will setup one uploader process. <br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.  <br>**Minimum value** : `15000`  <br>**Maximum value** : `65000`|integer (int32)|
|**version**  <br>*optional*|version number of dfget binary.|string|


<a name="peercreateresponse"></a>
### PeerCreateResponse
ID of created peer.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|Peer ID of the node which dfget locates on. <br>Every peer has a unique ID among peer network.<br>It is generated via host's hostname and IP address.|string|


<a name="peerinfo"></a>
### PeerInfo
The detailed information of a peer in supernode.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of peer|string|
|**IP**  <br>*optional*|IP address which peer client carries.<br>(TODO) make IP field contain more information, for example<br>WAN/LAN IP address for supernode to recognize.|string (ipv4)|
|**created**  <br>*optional*|the time to join the P2P network|string (date-time)|
|**hostName**  <br>*optional*|host name of peer client node, as a valid RFC 1123 hostname.  <br>**Minimum length** : `1`|string (hostname)|
|**port**  <br>*optional*|when registering, dfget will setup one uploader process. <br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.  <br>**Minimum value** : `15000`  <br>**Maximum value** : `65000`|integer (int32)|
|**version**  <br>*optional*|version number of dfget binary|string|


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
|**pieceResult**  <br>*optional*|pieceResult It indicates whether the dfgetTask successfully download the piece. <br>It's only useful when `status` is `RUNNING`.|enum (FAILED, SUCCESS, INVALID, SEMISUC)|


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
|**filter**  <br>*optional*|URL may contains some changeful query parameters such as authentication parameters. Dragonfly will <br>filter these parameter via 'filter'. The usage of it is that different URL may generate the same <br>download taskID.|string|
|**headers**  <br>*optional*|If there is any authentication step of the remote server, the headers should contains authenticated information.<br>Dragonfly will sent request taking the headers to remote server.|< string, string > map|
|**identifier**  <br>*optional*|This field is used for generating new downloading taskID to identify different downloading task of remote URL.|string|
|**type**  <br>*optional*|this must be image or file|string|
|**url**  <br>*optional*|the image or file location|string|


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
|**status**  <br>*optional*|The status of preheat task.<br>  WAITING -----> RUNNING -----> SUCCESS<br>                           \|--> FAILED<br>The initial status of a created preheat task is WAITING.<br>It's finished when a preheat task's status is FAILED or SUCCESS.<br>A finished preheat task's information can be queried within 24 hours.|enum (WAITING, RUNNING, FAILED, SUCCESS)|


<a name="taskcreaterequest"></a>
### TaskCreateRequest

|Name|Description|Schema|
|---|---|---|
|**cID**  <br>*optional*|CID means the client ID. It maps to the specific dfget process. <br>When user wishes to download an image/file, user would start a dfget process to do this. <br>This dfget is treated a client and carries a client ID. <br>Thus, multiple dfget processes on the same peer have different CIDs.|string|
|**callSystem**  <br>*optional*|This field is for debugging. When caller of dfget is using it to files, he can pass callSystem<br>name to dfget. When this field is passing to supernode, supernode has ability to filter them via <br>some black/white list to guarantee security, or some other purposes.  <br>**Minimum length** : `1`|string|
|**dfdaemon**  <br>*optional*|tells whether it is a call from dfdaemon. dfdaemon is a long running<br>process which works for container engines. It translates the image<br>pulling request into raw requests into those dfget recognizes.|boolean|
|**filter**  <br>*optional*|filter is used to filter request queries in URL.<br>For example, when a user wants to start to download a task which has a remote URL of <br>a.b.com/fileA?user=xxx&auth=yyy, user can add a filter parameter ["user", "auth"]<br>to filter the url to a.b.com/fileA. Then this parameter can potentially avoid repeatable<br>downloads, if there is already a task a.b.com/fileA.|< string > array|
|**headers**  <br>*optional*|extra HTTP headers sent to the rawURL.<br>This field is carried with the request to supernode. <br>Supernode will extract these HTTP headers, and set them in HTTP downloading requests<br>from source server as user's wish.|< string, string > map|
|**identifier**  <br>*optional*|special attribute of remote source file. This field is used with taskURL to generate new taskID to<br>identify different downloading task of remote source file. For example, if user A and user B uses<br>the same taskURL and taskID to download file, A and B will share the same peer network to distribute files.<br>If user A additionally adds an identifier with taskURL, while user B still carries only taskURL, then A's<br>generated taskID is different from B, and the result is that two users use different peer networks.|string|
|**md5**  <br>*optional*|md5 checksum for the resource to distribute. dfget catches this parameter from dfget's CLI<br>and passes it to supernode. When supernode finishes downloading file/image from the source location,<br>it will validate the source file with this md5 value to check whether this is a valid file.|string|
|**path**  <br>*optional*|path is used in one peer A for uploading functionality. When peer B hopes<br>to get piece C from peer A, B must provide a URL for piece C.<br>Then when creating a task in supernode, peer A must provide this URL in request.|string|
|**rawURL**  <br>*optional*|The is the resource's URL which user uses dfget to download. The location of URL can be anywhere, LAN or WAN.<br>For image distribution, this is image layer's URL in image registry.<br>The resource url is provided by command line parameter.|string|


<a name="taskcreateresponse"></a>
### TaskCreateResponse
response get from task creation request.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of the created task.|string|
|**fileLength**  <br>*optional*|The length of the file dfget requests to download in bytes.|integer (int64)|
|**pieceSize**  <br>*optional*|The size of pieces which is calculated as per the following strategy<br>1. If file's total size is less than 200MB, then the piece size is 4MB by default.<br>2. Otherwise, it equals to the smaller value between totalSize/100MB + 2 MB and 15MB.|integer (int32)|


<a name="taskinfo"></a>
### TaskInfo
detailed information about task in supernode.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of the task.|string|
|**callSystem**  <br>*optional*|This field is for debugging. When caller of dfget is using it to files, he can pass callSystem<br>name to dfget. When this field is passing to supernode, supernode has ability to filter them via <br>some black/white list to guarantee security, or some other purposes.  <br>**Minimum length** : `1`|string|
|**cdnStatus**  <br>*optional*|The status of the created task related to CDN functionality.|enum (WAITING, RUNNING, FAILED, SUCCESS, SOURCE_ERROR)|
|**dfdaemon**  <br>*optional*|tells whether it is a call from dfdaemon. dfdaemon is a long running<br>process which works for container engines. It translates the image<br>pulling request into raw requests into those dfget recganises.|boolean|
|**fileLength**  <br>*optional*|The length of the file dfget requests to download in bytes<br>which including the header and the trailer of each piece.|integer (int64)|
|**headers**  <br>*optional*|extra HTTP headers sent to the rawURL.<br>This field is carried with the request to supernode. <br>Supernode will extract these HTTP headers, and set them in HTTP downloading requests<br>from source server as user's wish.|< string, string > map|
|**httpFileLength**  <br>*optional*|The length of the source file in bytes.|integer (int64)|
|**identifier**  <br>*optional*|special attribute of remote source file. This field is used with taskURL to generate new taskID to<br>identify different downloading task of remote source file. For example, if user A and user B uses<br>the same taskURL and taskID to download file, A and B will share the same peer network to distribute files.<br>If user A additionally adds an identifier with taskURL, while user B still carries only taskURL, then A's<br>generated taskID is different from B, and the result is that two users use different peer networks.|string|
|**md5**  <br>*optional*|md5 checksum for the resource to distribute. dfget catches this parameter from dfget's CLI<br>and passes it to supernode. When supernode finishes downloading file/image from the source location,<br>it will validate the source file with this md5 value to check whether this is a valid file.|string|
|**path**  <br>*optional*|path is used in one peer A for uploading functionality. When peer B hopes<br>to get piece C from peer A, B must provide a URL for piece C.<br>Then when creating a task in supernode, peer A must provide this URL in request.|string|
|**pieceSize**  <br>*optional*|The size of pieces which is calculated as per the following strategy<br>1. If file's total size is less than 200MB, then the piece size is 4MB by default.<br>2. Otherwise, it equals to the smaller value between totalSize/100MB + 2 MB and 15MB.|integer (int32)|
|**pieceTotal**  <br>*optional*||integer (int32)|
|**rawURL**  <br>*optional*|The is the resource's URL which user uses dfget to download. The location of URL can be anywhere, LAN or WAN.<br>For image distribution, this is image layer's URL in image registry.<br>The resource url is provided by command line parameter.|string|
|**realMd5**  <br>*optional*|when supernode finishes downloading file/image from the source location,<br>the md5 sum of the source file will be calculated as the value of the realMd5.<br>And it will be used to compare with md5 value to check whether this is a valid file.|string|
|**taskURL**  <br>*optional*|taskURL is generated from rawURL. rawURL may contains some queries or parameter, dfget will filter some queries via<br>--filter parameter of dfget. The usage of it is that different rawURL may generate the same taskID.|string|


<a name="taskupdaterequest"></a>
### TaskUpdateRequest
request used to update task attributes.


|Name|Description|Schema|
|---|---|---|
|**peerID**  <br>*optional*|ID of the peer which has finished to download the whole task.|string|





