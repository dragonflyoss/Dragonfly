package ha

import (
	"fmt"
	"github.com/dragonflyoss/Dragonfly/common/constants"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
)

//Manager is the struct to manager supernode ha.
type Manager struct {
	advertiseIP string
	useHa       bool
	nodeStatus  int
	tool        Tool
}

//NewManager produce the Manager object.
func NewManager(cfg *config.Config) (*Manager, error) {
	toolMgr, err := NewEtcdMgr(cfg)
	if err != nil {
		return nil, err
	}
	return &Manager{
		advertiseIP: cfg.AdvertiseIP,
		useHa:       cfg.UseHA,
		nodeStatus:  constants.SupernodeUseHaInit,
		tool:        toolMgr,
	}, nil
}

//TODO(yunfeiyangbuaa): handle these log,errors and exceptions in the future

//ElectDaemon is the main progress to implement active/standby switch.
func (ha *Manager) ElectDaemon(change chan int) {
	messageChannel := make(chan string)
	go ha.WatchActive(messageChannel)
	go ha.TryStandbyToActive(change)
	//go func(){
	//    time.Sleep(20*time.Second)
	//    ha.GiveUpActiveStatus()
	//}()
	for {

		if activeIP, ok := <-messageChannel; ok {
			//the active node is off.
			if activeIP == ActiveSupernodeOFF {
				//if the previous active supernode is itself.
				if ha.nodeStatus == constants.SupernodeUseHaActive {
					ha.ActiveToStandby()
					change <- constants.SupernodeUsehakill
				} else {
					ha.TryStandbyToActive(change)
				}
			}
		}
	}

}

//GetSupernodeStatus get supernode's status.
func (ha *Manager) GetSupernodeStatus() int {
	if ha.useHa == false {
		return constants.SupernodeUseHaFalse
	}
	return ha.nodeStatus
}

//CompareAndSetSupernodeStatus set supernode's status.
func (ha *Manager) CompareAndSetSupernodeStatus(preStatus int, nowStatus int) bool {
	if ha.nodeStatus == preStatus {
		ha.nodeStatus = nowStatus
		return true
	}
	return false
}

//StandbyToActive change the status from standby to active.
func (ha *Manager) StandbyToActive() {
	if ha.nodeStatus == constants.SupernodeUseHaStandby {
		ha.nodeStatus = constants.SupernodeUseHaActive
	} else {
		fmt.Println(ha.advertiseIP, "is already active")
	}
}

//ActiveToStandby  change the status from active to standby.
func (ha *Manager) ActiveToStandby() {
	if ha.nodeStatus == constants.SupernodeUseHaActive {
		ha.nodeStatus = constants.SupernodeUseHaStandby

	} else {
		fmt.Println(ha.advertiseIP, "is already standby")
	}
}

//TryStandbyToActive try to change the status from standby to active.
func (ha *Manager) TryStandbyToActive(change chan int) {
	finished := make(chan bool, 10)
	is, ip, _ := ha.tool.TryBeActive(finished)
	if is == true {
		ha.StandbyToActive()
		fmt.Println(ha.advertiseIP, "are active supernode,active ip is:", ip)
		change <- constants.SupernodeUseHaActive
		go ha.tool.ActiveResureItsStatus(finished)
		<-finished
		ha.ActiveToStandby()
		fmt.Println(ha.advertiseIP, "active status is finished")
	} else {
		fmt.Println(ha.advertiseIP, "are standby supernode,active ip is:", ip)
		change <- constants.SupernodeUseHaStandby
	}
}

//WatchActive keep watch whether the active supernode is off.
func (ha *Manager) WatchActive(messageChannel chan string) {
	ha.tool.WatchActiveChange(messageChannel)
}

//CloseHaManager close the tool use to implement supernode ha
func (ha *Manager) CloseHaManager() error {
	return ha.tool.CloseTool()
}

//GiveUpActiveStatus give up its active status because of unhealthy
func (ha *Manager) GiveUpActiveStatus() bool {
	return ha.tool.ActiveKillItself()
}
