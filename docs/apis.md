# Dragonfly SuperNode API


<a name="overview"></a>
## Overview
API is an HTTP API served by Dragonfly's SuperNode. It is the API dfget or Harbor uses to communicate
with the supernode.


### Version information
*Version* : 0.1


### URI scheme
*BasePath* : /v1.24  
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
when dfget lasts in five minutes without downloading or uploading task, dfget
automatically sends a DELETE /peers/{id} request.


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


<a name="pieces-id-get"></a>
### Get a piece
```
GET /pieces/{id}
```


#### Description
Get detailed information of a piece in supernode.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of piece|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|< [PieceInfo](#pieceinfo) > array|
|**404**|no such task|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`


<a name="pieces-id-put"></a>
### Update a piece
```
PUT /pieces/{id}
```


#### Description
Update some information of piece, like status of piece.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of piece|string|
|**Body**|**PieceUpdateRequest**  <br>*optional*|request body which contains task update information|[PieceUpdateRequest](#pieceupdaterequest)|


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
update information of a task.


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


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of task|string|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**204**|no error|No Content|
|**404**|no such peer|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


<a name="tasks-id-pieces-get"></a>
### Get pieces in task
```
GET /tasks/{id}/pieces
```


#### Description
Get fixed number of pieces in a task. The number is set in query.


#### Parameters

|Type|Name|Description|Schema|
|---|---|---|---|
|**Path**|**id**  <br>*required*|ID of task|string|
|**Query**|**num**  <br>*optional*|Request number of pieces of task. If request number is larger than the total pieces in supernode,<br>supernode returns the total pieces of task. If not set, supernode will set 4 by default.|integer (int64)|


#### Responses

|HTTP Code|Description|Schema|
|---|---|---|
|**200**|no error|< [PieceInfo](#pieceinfo) > array|
|**404**|no such task|[4ErrorResponse](#4errorresponse)|
|**500**|An unexpected server error occurred.|[Error](#error)|


#### Produces

* `application/json`




<a name="definitions"></a>
## Definitions

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
|**ID**  <br>*optional*|Peer ID of dfget client. Every peer has a unique ID among peer network.|string|
|**IP**  <br>*optional*|IP address which peer client carries|string (ipv4)|
|**callSystem**  <br>*optional*|This field is for debugging. When caller of dfget is using it to files, he can pass callSystem<br>name to dfget. When this field is passing to supernode, supernode has ability to filter them via <br>some black/white list to guarantee security, or some other purposes.  <br>**Minimum length** : `1`|string|
|**dfdaemon**  <br>*optional*|tells whether it is a call from dfdaemon.|boolean|
|**hostName**  <br>*optional*|host name of peer client node, as a valid RFC 1123 hostname.  <br>**Minimum length** : `1`|string (hostname)|
|**path**  <br>*optional*|This is actually an HTTP URLPATH of dfget. <br>Other peers can access the source file via this PATH.|string|
|**port**  <br>*optional*|when registering, dfget will setup one uploader process. <br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.  <br>**Minimum value** : `30000`  <br>**Maximum value** : `65535`|integer (int64)|
|**version**  <br>*optional*|version number of dfget binary|string|


<a name="peercreateresponse"></a>
### PeerCreateResponse
ID of created peer.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of created peer.|string|


<a name="peerinfo"></a>
### PeerInfo
The detailed information of a peer in supernode.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of peer|string|
|**IP**  <br>*optional*|IP address which peer client carries|string (ipv4)|
|**callSystem**  <br>*optional*|This field is for debugging. When caller of dfget is using it to files, he can pass callSystem<br>name to dfget. When this field is passing to supernode, supernode has ability to filter them via <br>some black/white list to guarantee security, or some other purposes.  <br>**Minimum length** : `1`|string|
|**dfdaemon**  <br>*optional*|tells whether it is a call from dfdaemon.|boolean|
|**hostName**  <br>*optional*|host name of peer client node, as a valid RFC 1123 hostname.  <br>**Minimum length** : `1`|string (hostname)|
|**path**  <br>*optional*|This is actually an HTTP URLPATH of dfget. <br>Other peers can access the source file via this PATH.|string|
|**port**  <br>*optional*|when registering, dfget will setup one uploader process. <br>This one acts as a server for peer pulling tasks.<br>This port is which this server listens on.  <br>**Minimum value** : `30000`  <br>**Maximum value** : `65535`|integer (int64)|
|**version**  <br>*optional*|version number of dfget binary|string|


<a name="pieceinfo"></a>
### PieceInfo
Peer's detailed information in supernode.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of the peer|string|


<a name="pieceupdaterequest"></a>
### PieceUpdateRequest

|Name|Schema|
|---|---|
|**ID**  <br>*optional*|string|


<a name="preheatcreaterequest"></a>
### PreheatCreateRequest
Request option of creating a preheat task in supernode.


|Name|Description|Schema|
|---|---|---|
|**filter**  <br>*optional*|URL may contains some changeful query parameters such as authentication parameters. Dragonfly will <br>filter these parameter via 'filter'. The usage of it is that different URL may generate the same <br>download taskID.|string|
|**headers**  <br>*optional*|If there is any authentication step of the remote server, the headers should contains authenticated information.<br>Dragonfly will sent request taking the headers to remote server.|< string, string > map|
|**identifier**  <br>*optional*|This field is used for generating new downloading taskID to indetify different downloading task of remote URL.|string|
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
|**headers**  <br>*optional*|extra HTTP headers sent to the rawURL.<br>This field is carried with the request to supernode. <br>Supernode will extract these HTTP headers, and set them in HTTP downloading requests<br>from source server as user's wish.|< string, string > map|
|**identifier**  <br>*optional*|special attribute of remote source file. This field is used with taskURL to generate new taskID to<br>indetify different downloading task of remote source file. For example, if user A and user B uses<br>the same taskURL and taskID to download file, A and B will share the same peer network to distribute files.<br>If user A additionally adds an indentifier with taskURL, while user B still carries only taskURL, then A's<br>generated taskID is different from B, and the result is that two users use different peer networks.|string|
|**md5**  <br>*optional*|md5 checksum for the resource to distribute. dfget catches this parameter from dfget's CLI<br>and passes it to supernode. When supernode finishes downloading file/image from the source location,<br>it will validate the source file with this md5 value to check whether this is a valid file.|string|
|**rawURL**  <br>*optional*|The is the resource's URL which user uses dfget to download. The location of URL can be anywhere, LAN or WAN.<br>For image distribution, this is image layer's URL in image registry.<br>The resource url is provided by command line parameter.|string|
|**taskURL**  <br>*optional*|taskURL is generated from rawURL. rawURL may contains some queries or parameter, dfget will filter some queries via<br>--filter parameter of dfget. The usage of it is that different rawURL may generate the same taskID.|string|


<a name="taskcreateresponse"></a>
### TaskCreateResponse
response get from task creation request.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of the created task.|string|


<a name="taskinfo"></a>
### TaskInfo
detailed information about task in supernode.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of the task.|string|


<a name="taskupdaterequest"></a>
### TaskUpdateRequest
request used to update task attributes.


|Name|Description|Schema|
|---|---|---|
|**ID**  <br>*optional*|ID of the created task.|string|





