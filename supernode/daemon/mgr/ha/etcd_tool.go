package ha

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"go.etcd.io/etcd/clientv3"
)

//EtcdMgr is the struct to manager etcd.
type EtcdMgr struct {
	config            clientv3.Config
	client            *clientv3.Client
	leaseTTL          int64
	leaseKeepAliveRsp <-chan *clientv3.LeaseKeepAliveResponse
	hostIP            string
	leaseResp         *clientv3.LeaseGrantResponse
}

const ActiveSupernodeOFF = ""

//NewEtcdMgr produce a etcdmgr object.
func NewEtcdMgr(cfg *config.Config) (*EtcdMgr, error) {
	config := clientv3.Config{
		Endpoints:   cfg.HAConfig,
		DialTimeout: 10 * time.Second,
	}
	// build connection to etcd.
	client, err := clientv3.New(config)
	return &EtcdMgr{
		hostIP: cfg.AdvertiseIP,
		config: config,
		client: client,
	}, err
}

//TODO(yunfeiyangbuaa): handle these log,errors and exceptions in the future

//WatchActiveChange is the progress to watch the etcd,if the value of key /lock/active changes,supernode will be notified.
func (etcd *EtcdMgr) WatchActiveChange(messageChannel chan string) {
	var watchStartRevision int64
	watcher := clientv3.NewWatcher(etcd.client)
	watchChan := watcher.Watch(context.TODO(), "/lock/active", clientv3.WithRev(watchStartRevision))
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			switch event.Type {
			case 0:
				//mvccpb.PUT means someone becomes a new active supernode.
				//fmt.Println("change to:", string(event.Kv.Value), "Revision:", event.Kv.CreateRevision, event.Kv.ModRevision)
				messageChannel <- "string(event.Kv.Value)"
			case 1:
				//mvccpb.DELETE means the active supernode is off.
				//fmt.Println("deleted, ", "Revision:", event.Kv.ModRevision)
				messageChannel <- ActiveSupernodeOFF
			}
		}
	}
}

//ObtainActiveInfo obtain the active supernode's information from etcd.
func (etcd *EtcdMgr) ObtainActiveInfo(key string) (string, error) {
	kv := clientv3.NewKV(etcd.client)
	var (
		getRes *clientv3.GetResponse
		err    error
	)
	if getRes, err = kv.Get(context.TODO(), key); err != nil {
		log.Fatal(err)
	}
	var value string
	for _, v := range getRes.Kvs {
		value = string(v.Value)
	}
	return value, err
}

//ActiveResureItsStatus keep look on the lease's renew response.
func (etcd *EtcdMgr) ActiveResureItsStatus(finished chan bool) bool {
	for {
		select {
		case keepResp := <-etcd.leaseKeepAliveRsp:
			if keepResp == nil {
				fmt.Println("lease renew fail ")
				finished <- true
				return false
			}
		}
	}
}

//TryBeActive try to change the supernode's status from standby to active.
func (etcd *EtcdMgr) TryBeActive(finished chan bool) (bool, string, error) {
	kv := clientv3.NewKV(etcd.client)
	//make a lease to obtain a lock
	lease := clientv3.NewLease(etcd.client)
	leaseResp, err := lease.Grant(context.TODO(), etcd.leaseTTL)
	keepRespChan, kaerr := lease.KeepAlive(context.TODO(), leaseResp.ID)
	etcd.leaseKeepAliveRsp = keepRespChan
	if kaerr != nil {
		log.Fatal(kaerr)
	}
	etcd.leaseResp = leaseResp
	//if the lock is available,get the lock.
	//else read the lock
	txn := kv.Txn(context.TODO())
	txn.If(clientv3.Compare(clientv3.CreateRevision("/lock/active"), "=", 0)).
		Then(clientv3.OpPut("/lock/active", etcd.hostIP, clientv3.WithLease(leaseResp.ID))).
		Else(clientv3.OpGet("/lock/active"))
	txnResp, err := txn.Commit()
	if err != nil {
		log.Fatal(err)
	}
	if !txnResp.Succeeded {
		//fmt.Println("this supernode get lock unsuccessfully")
		_, err = lease.Revoke(context.TODO(), leaseResp.ID)
		return false, string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value), err
	}
	//fmt.Println("this supernode get lock successfully")
	return true, etcd.hostIP, err
}

//ActiveKillItself cancels the renew of lease.
func (etcd *EtcdMgr) ActiveKillItself() bool {
	_, err := etcd.client.Revoke(context.TODO(), etcd.leaseResp.ID)
	if err != nil {
		fmt.Printf("cancel lease fail\n")
		return false
	}
	fmt.Printf("cancel lease success\n")
	return true
}

//CloseTool close the tool used to implement supernode ha
func (etcd *EtcdMgr) CloseTool() error {
	fmt.Println("close the tool")
	return etcd.client.Close()
}
