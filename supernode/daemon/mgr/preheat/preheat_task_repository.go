package preheat

import (
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

const(
	EXPIRED_TIME = 7 * 24 * 3600 * 1000;
)

type PreheatTaskRepository struct {
	preheatTasks *sync.Map
	loaded *sync.Once
}

func NewPreheatTaskRepository() *PreheatTaskRepository {
	r := &PreheatTaskRepository{
		preheatTasks: new(sync.Map),
		loaded: new(sync.Once),
	}
	r.load()
	return r
}

func(r *PreheatTaskRepository) Get(id string) *mgr.PreheatTask {
	t, ok := r.preheatTasks.Load(id)
	if ok {
		return t.(*mgr.PreheatTask)
	}
	return nil
}

func(r *PreheatTaskRepository) GetAll() []*mgr.PreheatTask {
	list := make([]*mgr.PreheatTask, 0)
	r.preheatTasks.Range(func(key, value interface{}) bool{
		list = append(list, value.(*mgr.PreheatTask))
		return true
	})
	return list
}

func(r *PreheatTaskRepository) GetAllIds() []string {
	list := make([]string, 0)
	r.preheatTasks.Range(func(key, value interface{}) bool{
		list = append(list, key.(string))
		return true
	})
	return list
}

func(r *PreheatTaskRepository) Add(task *mgr.PreheatTask) (*mgr.PreheatTask, error) {
	t, _ := r.preheatTasks.LoadOrStore(task.ID, task)
	return t.(*mgr.PreheatTask), nil
}

func(r *PreheatTaskRepository) Update(id string, task *mgr.PreheatTask) bool {
	v, _ := r.preheatTasks.Load(id)
	t, _ := v.(*mgr.PreheatTask)
	if t != nil {
		if task.ParentID != "" {
			t.ParentID = task.ParentID
		}
		if task.Children != nil {
			t.Children = task.Children
		}
		if task.Status != "" {
			t.Status = task.Status
		}
		if task.StartTime > 0 {
			t.StartTime = task.StartTime
		}
		if task.FinishTime > 0 {
			t.FinishTime = task.FinishTime
		}
		return true
	}
	return false
}

func(r *PreheatTaskRepository) Delete(id string) bool {
	_, existed := r.preheatTasks.Load(id)
	r.preheatTasks.Delete(id)
	return existed
}

func(r *PreheatTaskRepository) IsExpired(id string) bool {
	t, _ := r.preheatTasks.Load(id)
	if t == nil {
		return false
	}
	return t != nil && r.expired(t.(*mgr.PreheatTask).StartTime)
}

func (r *PreheatTaskRepository) expired(timestamp int64) bool {
	return time.Now().UnixNano()/int64(time.Millisecond) > timestamp+EXPIRED_TIME
}

func(r *PreheatTaskRepository) load() {
	r.loaded.Do(func(){

	})
}

