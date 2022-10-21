package sqldialect

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
)

type SQLDialect interface {
	// GCAdd() will record a Txn as active
	GCAdd(string, dto.TxnControlCounters, dto.TxnControlCounters) error
	// GCCollectAll() will remove all records from data tables.
	GCCollectAll() error
	// GCCollectObsoleted() must be mutex-protected.
	GCCollectObsoleted(minTransactionID int) error
	// GCControlTablesPurge() will remove all data from non ring control tables.
	GCControlTablesPurge() error
	// GCPurgeCache() will completely wipe the cache.
	GCPurgeCache() error
	// GCPurgeCache() will completely wipe the cache.
	GCPurgeEphemeral() error
	// PurgeAll() drops all data tables, does **not** drop control tables.
	PurgeAll() error
}

func NewSQLDialect(sqlEngine sqlengine.SQLEngine, namespaces tablenamespace.TableNamespaceCollection, controlAttributes sqlcontrol.ControlAttributes, name string) (SQLDialect, error) {
	switch strings.ToLower(name) {
	case "sqlite":
		return newSQLiteDialct(sqlEngine, namespaces, controlAttributes)
	default:
		return nil, fmt.Errorf("cannot accomodate sql dialect '%s'", name)
	}
}

func newSQLiteDialct(sqlEngine sqlengine.SQLEngine, namespaces tablenamespace.TableNamespaceCollection, controlAttributes sqlcontrol.ControlAttributes) (SQLDialect, error) {
	rv := &sqLiteDialect{
		controlAttributes: controlAttributes,
		namespaces:        namespaces,
		sqlEngine:         sqlEngine,
	}
	err := rv.initSQLiteEngine()
	return rv, err
}

type sqLiteDialect struct {
	controlAttributes sqlcontrol.ControlAttributes
	namespaces        tablenamespace.TableNamespaceCollection
	sqlEngine         sqlengine.SQLEngine
}

func (eng *sqLiteDialect) initSQLiteEngine() error {
	_, err := eng.sqlEngine.Exec(sqlEngineSetupDDL)
	return err
}

func (sl *sqLiteDialect) GCAdd(tableName string, parentTcc, lockableTcc dto.TxnControlCounters) error {
	maxTxnColName := sl.controlAttributes.GetControlMaxTxnColumnName()
	q := fmt.Sprintf(
		`
		UPDATE "%s" 
		SET "%s" = r.current_value
		FROM (
			SELECT *
			FROM
				"__iql__.control.gc.rings"
		) AS r
		WHERE 
			"%s" = ? 
			AND 
			"%s" = ? 
			AND
			r.ring_name = 'transaction_id'
			AND
			"%s" < CASE 
			   WHEN ("%s" - r.current_offset) < 0
				 THEN CAST(pow(2, r.width_bits) + ("%s" - r.current_offset)  AS int)
				 ELSE "%s" - r.current_offset
				 END
		`,
		tableName,
		maxTxnColName,
		sl.controlAttributes.GetControlTxnIdColumnName(),
		sl.controlAttributes.GetControlInsIdColumnName(),
		maxTxnColName,
		maxTxnColName,
		maxTxnColName,
		maxTxnColName,
	)
	_, err := sl.sqlEngine.Exec(q, lockableTcc.TxnId, lockableTcc.InsertId)
	return err
}

func (sl *sqLiteDialect) GCCollectObsoleted(minTransactionID int) error {
	return sl.gCCollectObsoleted(minTransactionID)
}

func (sl *sqLiteDialect) gCCollectObsoleted(minTransactionID int) error {
	maxTxnColName := sl.controlAttributes.GetControlMaxTxnColumnName()
	obtainQuery := fmt.Sprintf(
		`
		SELECT
			'DELETE FROM "' || name || '" WHERE "%s" < %d ; '
		FROM
			sqlite_master 
		where 
			type = 'table'
		  and
			name not like '__iql__%%' 
			and
			name NOT LIKE 'sqlite_%%' 
		`,
		maxTxnColName,
		minTransactionID,
	)
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *sqLiteDialect) GCCollectAll() error {
	return sl.gCCollectAll()
}

func (sl *sqLiteDialect) gCCollectAll() error {
	obtainQuery := `
		SELECT
			'DELETE FROM "' || name || '"  ; '
		FROM
			sqlite_master 
		where 
			type = 'table'
		  and
			name not like '__iql__%%' 
			and
			name NOT LIKE 'sqlite_%%' 
		`
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *sqLiteDialect) GCControlTablesPurge() error {
	return sl.gcControlTablesPurge()
}

func (sl *sqLiteDialect) gcControlTablesPurge() error {
	obtainQuery := `
		SELECT
		  'DELETE FROM "' || name || '" ; '
		FROM
			sqlite_master 
		where 
			type = 'table'
			and
			name like '__iql__%'
		`
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *sqLiteDialect) GCPurgeEphemeral() error {
	return sl.gcPurgeEphemeral()
}

func (sl *sqLiteDialect) GCPurgeCache() error {
	return sl.gcPurgeCache()
}

func (sl *sqLiteDialect) gcPurgeCache() error {
	query := `
	select distinct 
		'DROP TABLE IF EXISTS "' || name || '" ; ' 
	from sqlite_schema 
	where type = 'table' and name like ?
	`
	rows, err := sl.sqlEngine.Query(query, sl.namespaces.GetAnalyticsCacheTableNamespaceConfigurator().GetLikeString())
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(rows)
}

func (sl *sqLiteDialect) gcPurgeEphemeral() error {
	query := `
	select distinct 
		'DROP TABLE IF EXISTS "' || name || '" ; ' 
	from 
		sqlite_schema 
	where 
		type = 'table' 
		and 
		name NOT like ? 
		and 
		name not like '__iql__%' 
		and
		name NOT LIKE 'sqlite_%' 
	`
	rows, err := sl.sqlEngine.Query(query, sl.namespaces.GetAnalyticsCacheTableNamespaceConfigurator().GetLikeString())
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(rows)
}

func (sl *sqLiteDialect) PurgeAll() error {
	return sl.purgeAll()
}

func (sl *sqLiteDialect) purgeAll() error {
	obtainQuery := `
		SELECT
			'DROP TABLE IF EXISTS "' || name || '" ; '
		FROM
			sqlite_master 
		where 
			type = 'table'
		  AND
			name NOT LIKE '__iql__%'
			and
			name NOT LIKE 'sqlite_%'
		`
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *sqLiteDialect) readExecGeneratedQueries(queryResultSet *sql.Rows) error {
	defer queryResultSet.Close()
	var queries []string
	for {
		hasNext := queryResultSet.Next()
		if !hasNext {
			break
		}
		var s string
		err := queryResultSet.Scan(&s)
		if err != nil {
			return err
		}
		queries = append(queries, s)
	}
	err := sl.sqlEngine.ExecInTxn(queries)
	return err
}
