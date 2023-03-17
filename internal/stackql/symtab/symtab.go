package symtab

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"

	"github.com/stackql/go-suffix-map/pkg/suffixmap"
	"github.com/stackql/stackql/internal/stackql/logging"
)

type Entry struct {
	Type string
	In   string
	Data interface{}
}

func NewSymTabEntry(t string, data interface{}, in string) Entry {
	return Entry{
		Type: t,
		Data: data,
		In:   in,
	}
}

type SymTab interface {
	GetSymbol(interface{}) (Entry, error)
	NewLeaf(k interface{}) (SymTab, error)
	SetSymbol(interface{}, Entry) error
}

type HashMapTreeSymTab struct {
	tab    map[interface{}]Entry
	leaves map[interface{}]SymTab
}

func NewHashMapTreeSymTab() *HashMapTreeSymTab {
	return &HashMapTreeSymTab{
		tab:    make(map[interface{}]Entry),
		leaves: make(map[interface{}]SymTab),
	}
}

func (st *HashMapTreeSymTab) GetSymbol(k interface{}) (Entry, error) {
	//nolint:gocritic // this is a type switch and may well expand in the future
	switch k := k.(type) {
	case *sqlparser.ColName:
		logging.GetLogger().Infoln(fmt.Sprintf("reading from symbol table using ColIdent %v", k))
		return st.GetSymbol(k.Name.GetRawVal())
	}
	v, ok := st.tab[k]
	if ok {
		return v, nil
	}
	//nolint:gocritic // this is a type switch and may well expand in the future
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
	return Entry{}, fmt.Errorf("could not locate symbol %v", k)
}

func (st *HashMapTreeSymTab) SetSymbol(k interface{}, v Entry) error {
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
