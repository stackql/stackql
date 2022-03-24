package symtab

import (
	"fmt"

	"vitess.io/vitess/go/vt/sqlparser"

	log "github.com/sirupsen/logrus"
	"github.com/stackql/go-suffix-map/pkg/suffixmap"
)

type SymTabEntry struct {
	Type string
	In   string
	Data interface{}
}

func NewSymTabEntry(t string, data interface{}, in string) SymTabEntry {
	return SymTabEntry{
		Type: t,
		Data: data,
		In:   in,
	}
}

type SymTab interface {
	GetSymbol(interface{}) (SymTabEntry, error)
	NewLeaf(k interface{}) (SymTab, error)
	SetSymbol(interface{}, SymTabEntry) error
}

type HashMapTreeSymTab struct {
	tab    map[interface{}]SymTabEntry
	leaves map[interface{}]SymTab
}

func NewHashMapTreeSymTab() *HashMapTreeSymTab {
	return &HashMapTreeSymTab{
		tab:    make(map[interface{}]SymTabEntry),
		leaves: make(map[interface{}]SymTab),
	}
}

func (st *HashMapTreeSymTab) GetSymbol(k interface{}) (SymTabEntry, error) {
	switch k := k.(type) {
	case *sqlparser.ColName:
		log.Infoln(fmt.Sprintf("reading from symbol table using ColIdent %v", k))
		return st.GetSymbol(k.Name.GetRawVal())
	}
	v, ok := st.tab[k]
	if ok {
		return v, nil
	}
	switch key := k.(type) {
	case string:
		for ki, vi := range st.tab {
			switch ki := ki.(type) {
			case string:
				if suffixmap.SuffixMatches(ki, key) {
					return vi, nil
				}
			}
		}
	}
	for _, v := range st.leaves {
		lv, err := v.GetSymbol(k)
		if err == nil {
			return lv, nil
		}
	}
	return SymTabEntry{}, fmt.Errorf("could not locate symbol %v", k)
}

func (st *HashMapTreeSymTab) SetSymbol(k interface{}, v SymTabEntry) error {
	_, ok := st.tab[k]
	if ok {
		return fmt.Errorf("symbol %v already present in symtab", k)
	}
	st.tab[k] = v
	return nil
}

func (st *HashMapTreeSymTab) NewLeaf(k interface{}) (SymTab, error) {
	_, ok := st.leaves[k]
	if ok {
		return nil, fmt.Errorf("leaf symtab %v already present in symtab", k)
	}
	st.leaves[k] = NewHashMapTreeSymTab()
	return st.leaves[k], nil
}
