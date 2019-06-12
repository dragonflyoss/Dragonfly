# Decentralized Plugin

Today, dfget talks to supernode to discover peers information and publish its own
status. While it is working great, it imposes certain operation overhead to run
high available supernode cluster. In order to simplify the deployment, we propose
a new way of exchanging status between dfget clients.

## dfdaemon

Today, dfdaemon is a long-running proxy service for docker and it calls dfget
internally to roundtrip the http requests. This is not necessary given that dfget
is also written in Go. A direct library call seems to be more straightforward than
running an `exec.Command`:

## dfget

Once the existing dfget functionality is moved to dfdaemon, we can make dfget
much simpler. The new dfget is just a cli tool which talks with the local daemon
and all the heavy work is done at the daemon side.

## architecture

Previously, df-client talks with supernode directly. It is straightforward, but
since new alternatives can arise, and even supernode itself might have its own
big changes where compatibilities are hard to maintain. A new interface has to
be introduced.

```
+-----------+        +--------+
| supernode |        | gossip |
+-----+-----+        +----+---+
      ^                   ^
      |                   |
      |                   |
+-----+-------------------+---+
|      Tracker Interface      |
+--------------+--------------+
               ^
               |
               |
        +------+------+
        |  df-client  |
        +-------------+
```

In the new architecture, the relationship between df-client and supernode are
decoupled, they all follow the provided `Tracker` interface. At the same time,
new implementations of tracker can be plugged in to replace supernode entirely.

### tracker

To start with, we can rename `SupernodeAPI` to `Tracker`, this is a minimal change
we can start with. Then we can implement a new tracker with gossip.

```go
type Tracker interface {
	Register(node string, req *types.RegisterRequest) (resp *types.RegisterResponse, e error)
	PullPieceTask(node string, req *types.PullPieceTaskRequest) (resp *types.PullPieceTaskResponse, e error)
	ReportPiece(node string, req *types.ReportPieceRequest) (resp *types.BaseResponse, e error)
	ServiceDown(node string, taskID string, cid string) (resp *types.BaseResponse, e error)
	ReportClientError(node string, req *types.ClientErrorRequest) (resp *types.BaseResponse, e error)
}
```

## bootstrap

When the dfdaemon(including the dfget today) is starting, it runs a discovery to
find its peers.

```
$ curl -X GET https://discovery.example.com/peers?region=foo&availabilityZone=bar
{"seed": "x.x.x.x"}
```

Once that step is done, the dfdaemon should know all its peers list now. There are
other alternatives:

1. supernode can take this role for bootstrapping.
1. we can point all the peers to a static ip address(dns name is prefered).
1. or specifically in kubernetes deployment, they can find peers via Nodes.

Here is the interface for peers discovery:

```go
package peer

type Interface interface {
    Peers() []string
}
```

At eBay, we have three implementations for this interface:

### IP

This is the most trivial implementation: we pass the initial peer address via flags:

```go
package fixed

type provider struct {
	peers []string
}

func (p *provider) Peers() []string {
	return p.peers
}
```

### Nodes

In kubernetes environment, mostly, dfdaemon is deployed as daemonsets and they are
going to run on every node, so we can leverage this attribute to do bootstraping:

```go
package nodes

type provider struct {
	clientset *kubernetes.Clientset
}

func (p *provider) Peers() []string {
	peers := []string{}
	nodes, err := p.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return peers
	}
	for _, node := range nodes.Items {
		peers = append(peers, node.Name)
	}
	return peers
}
```

### Endpoints

At eBay, we start with a smaller deployment per cluster, for example 10~20 peers
in a 2k nodes cluster, in that scenario, we run deployment instead of daemonset,
and since we combine the registry proxy together with the p2p cache peer, they
share the same pod selector, and their addresses appear in its Endpoint object:

```go
package endpoints

type endpointsGetter interface {
	Get(string, metav1.GetOptions) (*v1.Endpoints, error)
}

type provider struct {
	getter    endpointsGetter
	namespace string
	name      string
}

func (p *provider) Peers() []string {
	peers := []string{}
	ep, err := p.getter.Get(p.name, metav1.GetOptions{})
	if err != nil {
		return peers
	}
	for _, set := range ep.Subsets {
		for _, addr := range set.Addresses {
			peers = append(peers, addr.IP)
		}
	}
	return peers
}
```

## publish information

There is an awesome tool which is named [serf][2] which runs gossip protocol, and
it adds an event layer based on the underlying protocol.

So whenever a peer want to publish information, we can make it simple by firing
an event, and then we can assume that the message is going to be distributed by all
the peers.

### Layer Event

Layer event publishes layer related information:

```go
type LayerEvent struct {
	// Status is StatusStarted or StatusEnded
	Status int `json:"status"`
	// Digest is the layer sha256 checksum digest.
	Digest string `json:"digest"`
	// Address is the peer address.
	Address string `json:"address"`
	// Blocks is the number of blocks that it has fetched.
	Blocks int64 `json:"blocks"`
}
```

There are multiple scenarios when the peer needs to send this event:

#### Download Initialized

This happens when peer starts downloading one of image layer:

```go
// StartLayer broadcasts a message to cluster that it can accept new requests for
// that layer. It optionally accepts a duration parameter which indicates the time
// that it will expire after.
func (s *serf) StartLayer(digest string) {
	s.startLayer(digest, 0)
}

func (s *serf) startLayer(digest string, blocks int64) {
	layerEvent := &LayerEvent{
		Status:  StatusStarted,
		Digest:  digest,
		Address: s.address,
		Blocks:  blocks,
	}
	data, err := json.Marshal(layerEvent)
	if err != nil {
		glog.Errorf("failed to marshal start event for %s: %s", digest, err)
		return
	}
	s.agent.UserEvent(EventLayer, data, false)
}
```

Layer event always starts with block 0.

#### Download Progressed

This happens when the peer gets more blocks:

```go
// InProgress broadcasts a message to cluster that it already has several blocks
// for the provided digest, this piece of information is useful, since we now can
// get the file by blocks with semantics like range.
func (s *serf) InProgress(digest string, num int64) {
	layerEvent := &LayerEvent{
		Status:  StatusProgressed,
		Digest:  digest,
		Address: s.address,
		Blocks:  num,
	}
	data, err := json.Marshal(layerEvent)
	if err != nil {
		glog.Errorf("failed to marshal blocks event for %s: %s", digest, err)
		return
	}
	s.agent.UserEvent(EventLayer, data, false)
}
```

This event basically wants its peers to realize that there are new blocks available
for this specific layer.

#### Layer Removed

This happens when peers discard this layer because of reasons like garbage
collection.

```go
// EndLayer broadcast a message to cluster that no more new requests can reach to
// this host for layer downloading.
func (s *serf) EndLayer(digest string) {
	layerEvent := &LayerEvent{
		Status:  StatusEnded,
		Digest:  digest,
		Address: s.address,
	}
	data, err := json.Marshal(layerEvent)
	if err != nil {
		glog.Errorf("failed to marshal end event for %s: %s", digest, err)
		return
	}
	// delete from local immediately.
	s.delNode(digest, s.address)
	// propagate to other peers
	s.agent.UserEvent(EventLayer, data, false)
}
```

### Length Event

Length event shares the file length information for peers.

```go
type LengthEvent struct {
	Digest string `json:"digest"`
	Length int64  `json:"length"`
}
```

### Sync Event

Sync event happens when peer restarts and asks for peer states again.

```go
type SyncEvent struct {
	Address string `json:"address"`
}
```

## listen events

Similarly, [serf][2] provides an interface for you to react on the events received.
It is pretty convenient, and fully customizable and extensible.

## events

There are multiple events which we care about for a http resource request:

1. The Content-Length for a given resource.
1. The number of blocks for a given resource which are present on peers.
1. The checksum for a given resource.

Each peer can either publish the events as mentioned above or react on the events.

## local state

Once we have the event mechanisms ready, each peer can maintain its own memory
state which describes the distribution status of different resources. This state
can help dfdaemon to choose peers when downloading layers.

In this proposal, we track the following states:

```go
type serf struct {
	// other fields

	// progressMap tracks each layer along with their progress
	progressMap map[string]*LayerProgress
	// lengthMap tracks each layer along with their content length
	lengthMap map[string]int64
	// inflights tracks whether a layer is currently being downloaded.
	inflights map[string]bool
}

type LayerProgress struct {
	// Address is the connection information of peer
	Address string
	// Source indicates whether this layer is coming from original source. (non peers)
	Source  bool
	// Blocks is the number blocks available.
	Blocks  int64
}
```

## Request Flow

Let's say we have the states tracked in local memory, and now we have a request
coming, the following is the pseudo code on how it is handled.

```go
func (p *Puller) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	// path should be in multihash form, for exaple: /sha256:<digest>
	path := req.URL.Path

	// case one: client is requesting a specific block.
	block := req.URL.Query().Get("block")
	iblock, err := strconv.ParseInt(block, 10, 64)
	if err == nil {
		localBlocks := p.state.LocalProgress(path)
		if localBlocks < iblock {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if err := p.ServeLocal(writer, path, iblock, 0); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// source is a required query string parameter
	qsSrc := req.URL.Query().Get("source")
	source, err := url.QueryUnescape(qsSrc)
	// fetch its content-length information:
	qsLen := req.URL.Query().Get("len")
	// 1. the request itself provides this information
	// 2. query from local state
	// 3. send a HEAD request to source.
	length, err := p.resolveLength(path, qsLen, source)

	// optionally, client can specify when this layer data should expire. This
	// overrides the default value.
	qsExpire := req.URL.Query().Get("expire")
	expireSeconds, err := strconv.ParseInt(qsExpire, 10, 64)

	// if Inflight call returns false, it means there is another inflight request
	// which is already downloading this layer locally, in this case, we serve
	// this request by pulling data from peers.
	if p.state.Inflight(path) == false {
		p.ReadPeer(writer, path, length, expireSeconds)
		return
	}

	// Now this is the first request on this layer, we can either request the data
	// from remote or peers, and in the same time, cache the data locally.

	// FileWriter encapsulates the writer interface, it writes back to response
	// as well as to local file. The callback object handles the following events:
	//   OnProgress(string, int64)
	//   OnSuccess(string)
	//   OnError(string, error)
	fw, err := FileWriter(writer, p.localDir, path, length, NewCallback(p.state, p.cache, p.recorder, expireSeconds))

	// check how many nodes are currently holding this file
	nodes := p.state.Endpoints(path)
	if len(nodes) >= THRESHOLD {
		// tell others that this peer is also downloading
		p.state.StartLayer(path)
		p.ReadPeer(fw, path, length, expireSeconds)
			return
	}

	p.state.StartLayer(path)
	// otherwise fetch from source
	p.ReadSource(fw, path, source, length)
}
```

[1]: https://www.wikiwand.com/en/Gossip_protocol
[2]: https://github.com/hashicorp/serf
