package config

// Config contains all configuration of supernode.
type Config struct {
	// ListenPort is the port supernode server listens on.
	// default:
	ListenPort int

	// HomeDir is working directory of supernode.
	// default: /home/admin/supernode
	HomeDir string

	// the core pool size of ScheduledExecutorService.
	// When a request to start a download task, supernode will construct a thread concurrent pool
	// to download pieces of source file and write to specified storage.
	// Note: source file downloading is into pieces via range attribute set in HTTP header.
	// default: 10
	SchedulerCorePoolSize int

	// DownloadPath specifies the path where to store downloaded files from source address.
	// This path can be set beyond BaseDir, such as taking advantage of a different disk from BaseDir's.
	// default: $BaseDir/downloads
	DownloadPath string

	// PeerUpLimit is the upload limit of a peer. When dfget starts to play a role of peer,
	// it can only stand PeerUpLimit upload tasks from other peers.
	// default: 5
	PeerUpLimit int

	// PeerDownLimit is the download limit of a peer. When a peer starts to download a file/image,
	// it will download file/image in the form of pieces. PeerDownLimit mean that a peer can only
	// stand starting PeerDownLimit concurrent downloading tasks.
	// default: 4
	PeerDownLimit int

	// When dfget node starts to play a role of peer, it will provide services for other peers
	// to pull pieces. If it runs into an issue when providing services for a peer, its self failure
	// increases by 1. When the failure limit reaches EliminationLimit, the peer will isolate itself
	// as a unhealthy state. Then this dfget will be no longer called by other peers.
	// default: 5
	EliminationLimit int

	// FailureCountLimit is the failure count limit set in supernode for dfget client.
	// When a dfget client takes part in the peer network constructed by supernode,
	// supernode will command the peer to start distribution task.
	// When dfget client fails to finish distribution task, the failure count of client
	// increases by 1. When failure count of client reaches to FailureCountLimit(default 5),
	// dfget client will be moved to blacklist of supernode to stop playing as a peer.
	// default: 5
	FailureCountLimit int

	// LinkLimit is set for supernode to limit every piece download network speed (unit: MB/s).
	// default: 20
	LinkLimit int

	// SystemReservedBindwidth is the network bandwidth reserved for system software.
	// unit: MB/s
	// default: 20
	SystemReservedBindwidth int

	// MaxBindwidth is the network bandwidth that supernode can use.
	// unit: MB/s
	// default: 200
	MaxBindwidth int

	// Whether to enable profiler
	// default: true
	EnableProfiler bool

	// Whether to open DEBUG level
	// default: false
	Debug bool
}
