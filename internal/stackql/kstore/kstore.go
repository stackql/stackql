package kstore

import (
	"sort"
	"sync"

	"github.com/stackql/stackql/internal/stackql/dto"
)

var (
	once             sync.Once //nolint:gochecknoglobals // singleton
	kvStoreSingleton KStore    //nolint:gochecknoglobals // singleton
	_                KStore    = &standardKStore{}
)

type KStore interface {
	Del(k int)
	Min() (int, bool)
	Put(k int)
}

type standardKStore struct {
	m  *sync.Mutex
	ks map[int]struct{}
}

func (kv *standardKStore) Put(k int) {
	kv.m.Lock()
	defer kv.m.Unlock()
	kv.ks[k] = struct{}{}
}

func (kv *standardKStore) Del(k int) {
	kv.m.Lock()
	defer kv.m.Unlock()
	delete(kv.ks, k)
}

func (kv *standardKStore) Min() (int, bool) {
	kv.m.Lock()
	defer kv.m.Unlock()
	var arr []int
	for k := range kv.ks {
		arr = append(arr, k)
	}
	if len(arr) == 0 {
		return 0, false
	}
	sort.Ints(arr)
	return arr[0], true
}

func GetKStore(cfg dto.KStoreCfg) (KStore, error) {
	once.Do(func() {
		kvStoreSingleton = &standardKStore{
			m:  &sync.Mutex{},
			ks: make(map[int]struct{}),
		}
	})
	return kvStoreSingleton, nil
}
