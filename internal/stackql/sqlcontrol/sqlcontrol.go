package sqlcontrol

import (
	"strings"
)

var (
	_ ControlAttributes = &standardControlAttributes{}
)

const (
	gen_id_col_name         string = "iql_generation_id"
	ssn_id_col_name         string = "iql_session_id"
	txn_id_col_name         string = "iql_txn_id"
	max_txn_id_col_name     string = "iql_max_txn_id"
	ins_id_col_name         string = "iql_insert_id"
	insert_endoded_col_name string = "iql_insert_encoded"
	latest_update_col_name  string = "iql_last_modified"
	gc_status_col_name      string = "iql_gc_status"
)

type ControlAttributes interface {
	GetControlGCStatusColumnName() string
	GetControlGenIdColumnName() string
	GetControlInsIdColumnName() string
	GetControlInsertEncodedIdColumnName() string
	GetControlLatestUpdateColumnName() string
	GetControlMaxTxnColumnName() string
	GetControlSsnIdColumnName() string
	GetControlTxnIdColumnName() string
}

func GetControlAttributes(attrType string) ControlAttributes {
	return getControlAttributes(attrType)
}

func getControlAttributes(attrType string) ControlAttributes {
	switch strings.ToLower(attrType) {
	case "standard":
		return getStandardControlAttributes()
	default:
		return getStandardControlAttributes()
	}
}

func getStandardControlAttributes() ControlAttributes {
	return &standardControlAttributes{}
}

type standardControlAttributes struct{}

func (ca *standardControlAttributes) GetControlGenIdColumnName() string {
	return gen_id_col_name
}

func (ca *standardControlAttributes) GetControlSsnIdColumnName() string {
	return ssn_id_col_name
}

func (ca *standardControlAttributes) GetControlTxnIdColumnName() string {
	return txn_id_col_name
}

func (ca *standardControlAttributes) GetControlInsIdColumnName() string {
	return ins_id_col_name
}

func (ca *standardControlAttributes) GetControlInsertEncodedIdColumnName() string {
	return insert_endoded_col_name
}

func (ca *standardControlAttributes) GetControlMaxTxnColumnName() string {
	return max_txn_id_col_name
}

func (ca *standardControlAttributes) GetControlLatestUpdateColumnName() string {
	return latest_update_col_name
}

func (ca *standardControlAttributes) GetControlGCStatusColumnName() string {
	return gc_status_col_name
}
