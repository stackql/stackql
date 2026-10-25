package sqlcontrol

import (
	"strings"
)

var (
	_ ControlAttributes = &standardControlAttributes{}
)

const (
	genIDColName         string = "iql_generation_id"
	ssnIDColName         string = "iql_session_id"
	txnIDColName         string = "iql_txn_id"
	maxTxnIDColName      string = "iql_max_txn_id"
	insIDColName         string = "iql_insert_id"
	insertEndodedColName string = "iql_insert_encoded"
	latestUpdateColName  string = "iql_last_modified"
	gcStatusColName      string = "iql_gc_status"
)

type ControlAttributes interface {
	GetControlGCStatusColumnName() string
	GetControlGenIDColumnName() string
	GetControlInsIDColumnName() string
	GetControlInsertEncodedIDColumnName() string
	GetControlLatestUpdateColumnName() string
	GetControlMaxTxnColumnName() string
	GetControlSsnIDColumnName() string
	GetControlTxnIDColumnName() string
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

func (ca *standardControlAttributes) GetControlGenIDColumnName() string {
	return genIDColName
}

func (ca *standardControlAttributes) GetControlSsnIDColumnName() string {
	return ssnIDColName
}

func (ca *standardControlAttributes) GetControlTxnIDColumnName() string {
	return txnIDColName
}

func (ca *standardControlAttributes) GetControlInsIDColumnName() string {
	return insIDColName
}

func (ca *standardControlAttributes) GetControlInsertEncodedIDColumnName() string {
	return insertEndodedColName
}

func (ca *standardControlAttributes) GetControlMaxTxnColumnName() string {
	return maxTxnIDColName
}

func (ca *standardControlAttributes) GetControlLatestUpdateColumnName() string {
	return latestUpdateColName
}

func (ca *standardControlAttributes) GetControlGCStatusColumnName() string {
	return gcStatusColName
}
