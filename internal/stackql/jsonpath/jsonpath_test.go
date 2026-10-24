package jsonpath_test

import (
	"testing"

	"gotest.tools/assert"

	"github.com/stackql/stackql/internal/stackql/jsonpath"

	"encoding/json"

	"reflect"
)

func TestGet(t *testing.T) {
	var val interface{}
	jsonErr := json.Unmarshal([]byte(`{"a": {"b": "c"}}`), &val)
	assert.NilError(t, jsonErr)
	val, err := jsonpath.Get("$.a.b", val)
	assert.NilError(t, err)
	assert.Assert(t, true)
	assert.Equal(t, val, "c")
}

type myPrincipalStruct struct {
	Name string `json:"name"`
}

type myStruct struct {
	Principal   *myPrincipalStruct `json:"principal"`
	Description string             `json:"description"`
}

func TestGetStruct(t *testing.T) {
	var val interface{} = myStruct{}
	jsonErr := json.Unmarshal([]byte(`{"principal": {"name": "Joe Blogs"}, "description": "Data point 01 about this dude."}`), &val)
	assert.NilError(t, jsonErr)
	val, err := jsonpath.Get("$.principal.name", val)
	assert.NilError(t, err)
	assert.Assert(t, true)
	assert.Equal(t, val, "Joe Blogs")
}

func TestSet(t *testing.T) {
	initVal := make(map[string]interface{})
	jsonErr := json.Unmarshal([]byte(`{"a": {"b": "c"}}`), &initVal)
	assert.NilError(t, jsonErr)
	err := jsonpath.Set(initVal, "a.b", float64(22))
	assert.NilError(t, err)
	assert.Assert(t, true)
	rhs := make(map[string]interface{})
	rhsErr := json.Unmarshal([]byte(`{"a": {"b": 22}}`), &rhs)
	assert.NilError(t, rhsErr)
	assert.Assert(t, reflect.DeepEqual(initVal, rhs))
}

func TestSetStruct(t *testing.T) {
	ms := myStruct{}
	jsonErr := json.Unmarshal([]byte(`{"principal": {"name": "Joe Blogs"}, "description": "Data point 01 about this dude."}`), &ms)
	assert.NilError(t, jsonErr)
	err := jsonpath.Set(&ms, "principal.name", "Frank Smith")
	assert.NilError(t, err)
	assert.Assert(t, true)
	rhs := make(map[string]interface{})
	rhsErr := json.Unmarshal([]byte(`{"a": {"b": 22}}`), &rhs)
	assert.NilError(t, rhsErr)
	assert.Assert(t, reflect.DeepEqual(ms.Principal.Name, "Frank Smith"))
}

func TestSetStructAll(t *testing.T) {
	ms := myStruct{}
	var myTotalOverwrite interface{}
	jsonErr := json.Unmarshal([]byte(`{"principal": {"name": "Plastic Fantastic"}, "description": "Data point 01 about this dude."}`), &myTotalOverwrite)
	assert.NilError(t, jsonErr)
	err := jsonpath.Set(&ms, "", myTotalOverwrite)
	assert.NilError(t, err)
	assert.Assert(t, true)
	rhs := make(map[string]interface{})
	rhsErr := json.Unmarshal([]byte(`{"a": {"b": 22}}`), &rhs)
	assert.NilError(t, rhsErr)
	assert.Assert(t, reflect.DeepEqual(ms.Principal.Name, "Plastic Fantastic"))
}

func TestSetDollar(t *testing.T) {
	initVal := make(map[string]interface{})
	jsonErr := json.Unmarshal([]byte(`{"a": {"b": "c"}}`), &initVal)
	assert.NilError(t, jsonErr)
	err := jsonpath.Set(initVal, "$.a.b", float64(22))
	assert.NilError(t, err)
	assert.Assert(t, true)
	rhs := make(map[string]interface{})
	rhsErr := json.Unmarshal([]byte(`{"a": {"b": 22}}`), &rhs)
	assert.NilError(t, rhsErr)
	assert.Assert(t, reflect.DeepEqual(initVal, rhs))
}

func TestSplitPath(t *testing.T) {
	val, err := jsonpath.SplitSearchPath("$.a.b")
	assert.NilError(t, err)
	assert.Assert(t, true)
	rhs := []string{"a", "b"}
	assert.Assert(t, reflect.DeepEqual(val, rhs))
}

func TestSplitPathLong(t *testing.T) {
	val, err := jsonpath.SplitSearchPath("$.a.b.c[\"d.e\"]")
	assert.NilError(t, err)
	assert.Assert(t, true)
	rhs := []string{"a", "b", "c", `d.e`}
	assert.Assert(t, reflect.DeepEqual(val, rhs))
}
