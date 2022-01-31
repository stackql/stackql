package cache

import (
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultMarshallerKey string = "default_marshaller"
	RootMarshallerKey    string = "root_marshaller"
	ServiceMarshallerKey string = "service_marshaller"
)

type IKeyValCache interface {
	Len() int
	Get(string, IMarshaller) interface{}
	Put(string, interface{}, IMarshaller)
}

type Item struct {
	Value         interface{}     `json:"-"`
	RawValue      json.RawMessage `json:"raw_value"`
	LastAccess    int64
	Marshaller    IMarshaller `json:"-"`
	MarshallerKey string
	Tablespace    string
	TablespaceID  int
}

type TTLMap struct {
	m               map[string]*Item
	l               sync.Mutex
	cacheName       string
	cacheFileSuffix string
	runtimeCtx      dto.RuntimeCtx
	dbEngine        sqlengine.SQLEngine
}

func (m *TTLMap) persistToFile() {

	for k, v := range m.m {
		err := v.Marshaller.Marshal(v)
		if err != nil {
			log.Infoln(fmt.Sprintf("persist to file Marshal error = %s", err.Error()))
			continue
		}
		log.Debugln(fmt.Sprintf("persisting to cache: k = '%s', v = %v", k, v))
		blob, jsonErr := json.Marshal(v)
		if jsonErr != nil {
			log.Infoln(fmt.Sprintf("persist to file final Marshal error = %s", jsonErr.Error()))
			continue
		}
		log.Debugln(fmt.Sprintf("persisting to cache: k = '%s', blob = %s", k, string(blob)))
		m.dbEngine.CacheStorePut(k, blob, v.Tablespace, v.TablespaceID)
	}
}

func (m *TTLMap) getCacheDir(relativePath string) string {
	return path.Join(m.runtimeCtx.ProviderRootPath, relativePath)
}

func sanitisePath(p string) string {
	return p
}

func (m *TTLMap) getCacheFileName() string {
	if m.cacheFileSuffix == "" {
		return m.cacheName
	}
	return m.cacheName + "." + m.cacheFileSuffix
}

func (m *TTLMap) restoreFromFile() error {
	im := make(map[string]*Item)
	cachedKVs, err := m.dbEngine.CacheStoreGetAll()
	log.Infoln(fmt.Sprintf("len(cachedKVs) = %v", len(cachedKVs)))
	if err != nil {
		return err
	} else {
		for _, kv := range cachedKVs {
			val := &Item{}
			jErr := json.Unmarshal(kv.V, val)
			if jErr != nil {
				return fmt.Errorf("error unmarshaling kv raw, key = '%s': %v", kv.K, jErr)
			}
			marshaller, e := GetMarshaller(val.MarshallerKey)
			if e != nil {
				return e
			}
			val.Marshaller = marshaller
			jsonErr := val.Marshaller.Unmarshal(val)
			if jsonErr != nil {
				return fmt.Errorf("error unmarshaling kv item, key = '%s': %v", kv.K, jsonErr)
			}
			im[kv.K] = val
		}
	}
	m.m = im
	return nil
}

func NewTTLMap(
	dbEngine sqlengine.SQLEngine,
	runtimeCtx dto.RuntimeCtx,
	cacheName string,
	initSize int,
	maxTTL int,
	marshaller IMarshaller) IKeyValCache {
	log.Infoln(fmt.Sprintf("cache op: created new cache"))
	m := &TTLMap{
		m:               make(map[string]*Item, initSize),
		cacheName:       cacheName,
		cacheFileSuffix: constants.JsonStr,
		runtimeCtx:      runtimeCtx,
		dbEngine:        dbEngine,
	}
	restorErr := m.restoreFromFile()
	if restorErr != nil {
		log.Infoln(restorErr.Error())
	}
	go func() {
		for now := range time.Tick(time.Second) {
			m.l.Lock()
			for k, v := range m.m {
				if (maxTTL > 0) && ((now.Unix() - v.LastAccess) > int64(maxTTL)) {
					delete(m.m, k)
					log.Infoln(fmt.Sprintf("cache op: deleted %s", k))
				}
			}
			m.l.Unlock()
		}
	}()
	return m
}

func (m *TTLMap) Len() int {
	return len(m.m)
}

func (m *TTLMap) Put(k string, v interface{}, marshaller IMarshaller) {
	if v == nil {
		log.Infoln("attempting to add nil to cache")
		return
	}
	m.l.Lock()
	defer m.l.Unlock()
	log.Debugln(fmt.Sprintf("TTLMap.Put() called for k = %v, v = %v", k, v))
	it := &Item{
		Value:         v,
		Marshaller:    marshaller,
		MarshallerKey: marshaller.GetKey(),
	}
	m.m[k] = it
	log.Infoln(fmt.Sprintf("cache op: added %s", k))
	it.LastAccess = time.Now().Unix()
	log.Infoln(fmt.Sprintf("type of interface for Put = %T", v))
	m.persistToFile()
}

func (m *TTLMap) Get(k string, marshaller IMarshaller) (v interface{}) {
	m.l.Lock()
	i := 0
	for keyStr, v := range m.m {
		log.Infoln(fmt.Sprintf("key[%d] = %s, vale type = %T", i, keyStr, v))
		i++
	}
	if it, ok := m.m[k]; ok {
		v = it.Value
		log.Infoln(fmt.Sprintf("cache op: succeeded in retrieving %s", k))
		it.LastAccess = time.Now().Unix()
	} else {
		log.Infoln(fmt.Sprintf("cache op: failed to retrieve %s", k))
	}
	m.l.Unlock()
	return
}
