package internaldto

type RowsDTO struct {
	RowMap      map[string]map[string]interface{}
	ColumnOrder []string
	Err         error
	RowSort     func(map[string]map[string]interface{}) []string
}
