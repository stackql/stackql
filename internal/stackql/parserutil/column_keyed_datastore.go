package parserutil

type ColumnKeyedDatastore interface {
	Contains(ColumnarReference) bool
	ContainsString(string) bool
	Delete(ColumnarReference) bool
	DeleteByString(string) bool
	GetStringified() map[string]interface{}
	AndStringMap(map[string]interface{}) ColumnKeyedDatastore
	DeleteStringMap(map[string]interface{}) ColumnKeyedDatastore
}
