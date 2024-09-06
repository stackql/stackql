package sqlcontrol_test

import (
	"testing"

	. "github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stretchr/testify/assert"
)

// TestGetControlAttributes tests the GetControlAttributes function.
func TestGetControlAttributes(t *testing.T) {
	t.Run("standard type", func(t *testing.T) {
		attrType := "standard"
		controlAttrs := GetControlAttributes(attrType)

		assert.NotNil(t, controlAttrs, "Expected non-nil ControlAttributes for standard attrType")
	})

	t.Run("non-standard type", func(t *testing.T) {
		attrType := "non-standard"
		controlAttrs := GetControlAttributes(attrType)

		assert.NotNil(t, controlAttrs, "Expected non-nil ControlAttributes for non-standard attrType")
	})
}

// TestStandardControlAttributes tests the standardControlAttributes' methods.
func TestStandardControlAttributes(t *testing.T) {
	ca := GetControlAttributes("standard")

	assert.Equal(t, "iql_generation_id", ca.GetControlGenIDColumnName(), "Expected iql_generation_id")
	assert.Equal(t, "iql_session_id", ca.GetControlSsnIDColumnName(), "Expected iql_session_id")
	assert.Equal(t, "iql_txn_id", ca.GetControlTxnIDColumnName(), "Expected iql_txn_id")
	assert.Equal(t, "iql_insert_id", ca.GetControlInsIDColumnName(), "Expected iql_insert_id")
	assert.Equal(t, "iql_insert_encoded", ca.GetControlInsertEncodedIDColumnName(), "Expected iql_insert_encoded")
	assert.Equal(t, "iql_max_txn_id", ca.GetControlMaxTxnColumnName(), "Expected iql_max_txn_id")
	assert.Equal(t, "iql_last_modified", ca.GetControlLatestUpdateColumnName(), "Expected iql_last_modified")
	assert.Equal(t, "iql_gc_status", ca.GetControlGCStatusColumnName(), "Expected iql_gc_status")
}
