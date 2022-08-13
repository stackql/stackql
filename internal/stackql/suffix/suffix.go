package suffix

import (
	"github.com/stackql/go-openapistackql/openapistackql"

	"github.com/stackql/go-suffix-map/pkg/suffixmap"
)

type ParameterSuffixMap struct {
	sm suffixmap.SuffixMap
}

func NewParameterSuffixMap() *ParameterSuffixMap {
	return &ParameterSuffixMap{
		sm: suffixmap.NewSuffixMap(nil),
	}
}

func (psm *ParameterSuffixMap) Get(k string) (openapistackql.Addressable, bool) {
	rv, ok := psm.sm.Get(k)
	if !ok {
		return nil, false
	}
	crv, ok := rv.(openapistackql.Addressable)
	return crv, ok
}

func (psm *ParameterSuffixMap) GetAll() map[string]openapistackql.Addressable {
	m := psm.sm.GetAll()
	rv := make(map[string]openapistackql.Addressable)
	for k, v := range m {
		p, ok := v.(openapistackql.Addressable)
		if ok {
			rv[k] = p
		}
	}
	return rv
}

func (psm *ParameterSuffixMap) Put(k string, v openapistackql.Addressable) {
	psm.sm.Put(k, v)
}

func (psm *ParameterSuffixMap) Delete(k string) bool {
	return psm.sm.Delete(k)
}

func (psm *ParameterSuffixMap) Size() int {
	return psm.sm.Size()
}
