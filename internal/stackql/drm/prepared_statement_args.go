package drm

import (
	"fmt"
	"sort"
)

var (
	_ PreparedStatementArgs = &standardPreparedStatementArgs{}
)

type PreparedStatementArgs interface {
	Analyze() error
	compose() (childQueryComposition, error)
	GetArgs() []interface{}
	GetChild(int) PreparedStatementArgs
	GetChildren() map[int]PreparedStatementArgs
	GetExpandedArgs() []interface{}
	GetExpandedQuery() string
	GetQuery() string
	SetArgs([]interface{})
	SetChild(int, PreparedStatementArgs)
}

type standardPreparedStatementArgs struct {
	query         string
	args          []interface{}
	children      map[int]PreparedStatementArgs
	expandedQuery string
	expandedArgs  []interface{}
}

func NewPreparedStatementArgs(query string) PreparedStatementArgs {
	return &standardPreparedStatementArgs{
		query:    query,
		children: make(map[int]PreparedStatementArgs),
	}
}

func (ca *standardPreparedStatementArgs) GetChild(k int) PreparedStatementArgs {
	return ca.children[k]
}

func (ca *standardPreparedStatementArgs) GetChildren() map[int]PreparedStatementArgs {
	return ca.children
}

func (ca *standardPreparedStatementArgs) GetArgs() []interface{} {
	return ca.args
}

func (ca *standardPreparedStatementArgs) GetExpandedArgs() []interface{} {
	return ca.expandedArgs
}

func (ca *standardPreparedStatementArgs) GetQuery() string {
	return ca.query
}

func (ca *standardPreparedStatementArgs) GetExpandedQuery() string {
	return ca.expandedQuery
}

func (ca *standardPreparedStatementArgs) SetChild(i int, a PreparedStatementArgs) {
	ca.children[i] = a
}

func (ca *standardPreparedStatementArgs) SetArgs(args []interface{}) {
	ca.args = args
}

func (ca *standardPreparedStatementArgs) Analyze() error {
	composition, err := ca.compose()
	if err != nil {
		return err
	}
	ca.expandedQuery = composition.GetQueryString()
	ca.expandedArgs = composition.GetVarArgs()
	return nil
}

func (ca *standardPreparedStatementArgs) compose() (childQueryComposition, error) {
	var varArgs []interface{}
	j := 0
	query := ca.GetQuery()
	var childQueryStrings []interface{} // dunno why
	var keys []int
	for i := range ca.GetChildren() {
		keys = append(keys, i)
	}
	sort.Ints(keys)
	for _, k := range keys {
		cp := ca.GetChild(k)
		childResponse, err := cp.compose()
		if err != nil {
			return nil, err
		}
		childQueryStrings = append(childQueryStrings, childResponse.GetQueryString())
		varArgs = append(varArgs, childResponse.GetVarArgs()...)
		j = k
	}
	if len(childQueryStrings) > 0 {
		query = fmt.Sprintf(ca.GetQuery(), childQueryStrings...)
	}
	if len(ca.GetArgs()) >= j {
		varArgs = append(varArgs, ca.GetArgs()[j:]...)
	}
	return newChildQueryComposition(query, varArgs), nil
}

type childQueryComposition interface {
	GetQueryString() string
	GetChildQueryStrings() []interface{}
	GetVarArgs() []interface{}
}

func newChildQueryComposition(query string, varArgs []interface{}) childQueryComposition {
	return &standardChildQueryComposition{
		query: query,
		// childQueryComposition: childQueryComposition,
		varArgs: varArgs,
	}
}

type standardChildQueryComposition struct {
	query                          string
	childQueryComposition, varArgs []interface{}
}

func (cc *standardChildQueryComposition) GetQueryString() string {
	return cc.query
}

func (cc *standardChildQueryComposition) GetChildQueryStrings() []interface{} {
	return cc.childQueryComposition
}

func (cc *standardChildQueryComposition) GetVarArgs() []interface{} {
	return cc.varArgs
}
