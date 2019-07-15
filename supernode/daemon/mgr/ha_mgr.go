package mgr

//HaMgr is the interface to implement supernode Ha.
type HaMgr interface {
	//ElectDaemonthe is the daemon progress to implement active/standby switch.
	ElectDaemon(change chan int)

	//StandbyToActive changes supernode's status from standby to active after TryStandbyToActive successful.
	StandbyToActive()

	//ActiveToStandby changes supernode's status from active to standby after TryStandbyToActive failed.
	ActiveToStandby()

	//TryStandbyToActive try to change its status from standby to active.
	TryStandbyToActive(change chan int)

	//WatchActive keeps notice on active supernode's status change.
	WatchActive(messageChannel chan string)

	//HagetSupernodeState get supernode's status.
	GetSupernodeStatus() int

	//HaSetSupernodeState compare and set supernode's status.
	CompareAndSetSupernodeStatus(preStatus int, nowStatus int) bool

	//CloseHaManager close the tool use to implement supernode ha
	CloseHaManager() error

	//GiveUpActiveStatus give up its active status because of unhealthy
	GiveUpActiveStatus() bool
}
