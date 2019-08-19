package ha

import (
	"context"
	"fmt"
	"net/rpc"
	"strconv"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/go-openapi/strfmt"
	"github.com/sirupsen/logrus"

	"go.etcd.io/etcd/clientv3"
)

// EtcdMgr is the struct to manager etcd.
type EtcdMgr struct {
	config    *config.Config
	client    *clientv3.Client
	LeaseResp *clientv3.LeaseGrantResponse
}

const (
	// etcdTimeOut is the etcd client's timeout second.
	etcdTimeOut = 10 * time.Second

	// supernodeKeyPrefix is the key prefix of supernode info
	supernodeKeyPrefix = "/standby/supernode/"
)

// NewEtcdMgr produces a etcdmgr object.
func NewEtcdMgr(cfg *config.Config) (*EtcdMgr, error) {
	config := clientv3.Config{
		Endpoints:   cfg.HAConfig,
		DialTimeout: etcdTimeOut,
	}
	// build connection to etcd.
	client, err := clientv3.New(config)
	if err != nil {
		logrus.Errorf("failed to connect to etcd server,err %v", err)
		return nil, err
	}
	return &EtcdMgr{
		config: cfg,
		client: client,
	}, err

}

// SendSupernodesInfo send supernode info to other supernode.
func (etcd *EtcdMgr) SendSupernodesInfo(ctx context.Context, key, ip, pID string, listenPort, downloadPort, rpcPort int, hostName strfmt.Hostname, timeout int64) error {
	var respchan <-chan *clientv3.LeaseKeepAliveResponse
	kv := clientv3.NewKV(etcd.client)
	lease := clientv3.NewLease(etcd.client)
	leaseResp, e := lease.Grant(ctx, timeout)
	value := fmt.Sprintf("%s@%d@%d@%d@%s@%s", ip, listenPort, downloadPort, rpcPort, hostName, pID)
	if _, e = kv.Put(ctx, key, value, clientv3.WithLease(leaseResp.ID)); e != nil {
		logrus.Errorf("failed to put standby supernode's info to etcd as a lease,err %v", e)
		return e
	}
	etcd.LeaseResp = leaseResp
	if respchan, e = lease.KeepAlive(ctx, leaseResp.ID); e != nil {
		logrus.Errorf("failed to send heart beat to etcd to renew the lease %v", e)
		return e
	}
	//deal with the channel full warn
	//TODO(yunfeiyangbuaa):do with this code,because it is useless
	go func() {
		for {
			<-respchan
		}
	}()
	return nil
}

// Close closes the tool used to implement supernode ha.
func (etcd *EtcdMgr) Close(ctx context.Context) error {
	var err error
	if err = etcd.client.Close(); err != nil {
		logrus.Errorf("failed to close etcd client,err %v", err)
		return err
	}
	logrus.Info("success to close a etcd client")
	return nil
}

// WatchSupernodesChange is the progress to watch the etcd,if the value of key prefix changes,supernode will be notified.
func (etcd *EtcdMgr) WatchSupernodesChange(ctx context.Context, key string) error {
	//when start supernode,get supernode info
	if _, err := etcd.getSupenrodesInfo(ctx, key); err != nil {
		logrus.Errorf("failed to get standby supernode info,err: %v", err)
		return err
	}
	//etcd.registerOtherSupernodesAsPeer(ctx)
	watcher := clientv3.NewWatcher(etcd.client)
	watchChan := watcher.Watch(ctx, key, clientv3.WithPrefix())

	//after supernode start,if other supernode changes,do with it
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			logrus.Infof("success to notice supernodes changes,code(1:supernode add,0:supernode delete) %d", int(event.Type))
			if _, err := etcd.getSupenrodesInfo(ctx, key); err != nil {
				logrus.Errorf("failed to get standby supernode info,err: %v", err)
				return err
			}
		}
	}
	return nil
}

// GetSupenrodesInfo gets supernode info from etcd
func (etcd *EtcdMgr) getSupenrodesInfo(ctx context.Context, key string) ([]config.SupernodeInfo, error) {
	var (
		nodes  []config.SupernodeInfo
		getRes *clientv3.GetResponse
		e      error
	)
	kv := clientv3.NewKV(etcd.client)
	if getRes, e = kv.Get(ctx, key, clientv3.WithPrefix()); e != nil {
		logrus.Errorf("failed to get other supernode's information,err %v", e)
		return nil, e
	}
	for _, v := range getRes.Kvs {
		splits := strings.Split(string(v.Value), "@")
		// if the supernode is itself,skip
		if splits[5] == etcd.config.GetSuperPID() {
			continue
		}
		lPort, _ := strconv.Atoi(splits[1])
		dPort, _ := strconv.Atoi(splits[2])
		rPort, _ := strconv.Atoi(splits[3])
		rpcAddress := fmt.Sprintf("%s:%d", splits[0], rPort)
		conn, err := rpc.DialHTTP("tcp", rpcAddress)
		if err != nil {
			logrus.Errorf("failed to connect to the rpc port %s,err: %v", rpcAddress, err)
			return nil, err
		}
		nodes = append(nodes, config.SupernodeInfo{
			IP:           splits[0],
			ListenPort:   lPort,
			DownloadPort: dPort,
			RPCPort:      rPort,
			HostName:     strfmt.Hostname(splits[4]),
			PID:          splits[5],
			RPCClient:    conn,
		})
	}
	etcd.config.SetOtherSupernodeInfo(nodes)
	return nodes, nil
}
