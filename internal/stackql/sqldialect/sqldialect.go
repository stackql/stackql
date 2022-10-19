package sqldialect

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
)

type SQLDialect interface {
	// GCCollect() will collect unmarked (from input list) condemned records, from:
	//   - canonical tables.
	//   - cache.
	GCCollect([]int, []int) error
	// GCCollectAll() will collect **all** condemned / expired records, from both canonical tables and cache.
	GCCollectAll() error
	// GCCollectFromCache() will collect unmarked (from input list), expired cache records.
	GCCollectFromCache([]int) error
	// GCPurgeCache() will completely wipe the cache.
	GCPurgeCache() error
}

func NewSQLDialect(sqlEngine sqlengine.SQLEngine, namespaces tablenamespace.TableNamespaceCollection, name string) (SQLDialect, error) {
	switch strings.ToLower(name) {
	case "sqlite":
		return newSQLiteDialct(sqlEngine, namespaces)
	default:
		return nil, fmt.Errorf("cannot accomodate sql dialect '%s'", name)
	}
}

func newSQLiteDialct(sqlEngine sqlengine.SQLEngine, namespaces tablenamespace.TableNamespaceCollection) (SQLDialect, error) {
	return &sqLiteDialect{
		namespaces: namespaces,
		sqlEngine:  sqlEngine,
	}, nil
}

type sqLiteDialect struct {
	namespaces tablenamespace.TableNamespaceCollection
	sqlEngine  sqlengine.SQLEngine
}

func (sl *sqLiteDialect) GCCollectAll() error {
	return sl.gcCollectAll()
}

func (sl *sqLiteDialect) GCCollect(transactionIDs, cacheTransactionIDs []int) error {
	return sl.gcCollect(transactionIDs, cacheTransactionIDs)
}

func (sl *sqLiteDialect) GCCollectFromCache(transactionIDs []int) error {
	return sl.gcCollectFromCache(transactionIDs)
}

func (sl *sqLiteDialect) gcCollectAll() error {
	s, err := sl.getGCCollectAllTemplate()
	if err != nil {
		return err
	}
	s2, err := sl.getGCCollectCacheTemplate(nil)
	if err != nil {
		return err
	}
	s = append(s, s2...)
	err = sl.sqlEngine.ExecInTxn(s)
	return err
}

func (sl *sqLiteDialect) gcCollect(transactionIDs, cacheTransactionIDs []int) error {
	s, err := sl.getGCCollectTemplate(transactionIDs)
	if err != nil {
		return err
	}
	s2, err := sl.getGCCollectCacheTemplate(cacheTransactionIDs)
	if err != nil {
		return err
	}
	s = append(s, s2...)
	err = sl.sqlEngine.ExecInTxn(s)
	return err
}

func (sl *sqLiteDialect) gcCollectFromCache(transactionIDs []int) error {
	s, err := sl.getGCCollectCacheTemplate(transactionIDs)
	if err != nil {
		return err
	}
	err = sl.sqlEngine.ExecInTxn(s)
	return err
}

func (sl *sqLiteDialect) GCPurgeCache() error {
	return sl.gcPurgeCache()
}

func (sl *sqLiteDialect) gcPurgeCache() error {
	s, err := sl.getGCPurgeCacheTemplate()
	if err != nil {
		return err
	}
	err = sl.sqlEngine.ExecInTxn(s)
	return err
}

func (sl *sqLiteDialect) getGCPurgeCacheTemplate() ([]string, error) {
	query := `select distinct name from sqlite_schema where type = 'table' and name like ? ;`
	rows, err := sl.sqlEngine.Query(query, sl.namespaces.GetAnalyticsCacheTableNamespaceConfigurator().GetLikeString())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rv []string
	for {
		hasNext := rows.Next()
		if !hasNext {
			break
		}
		var s string
		err := rows.Scan(&s)
		if err != nil {
			return nil, err
		}
		rv = append(rv, fmt.Sprintf(`DROP TABLE IF EXISTS "%s" CASCADE; `, s))
	}
	return rv, nil
}

func (sl *sqLiteDialect) getGCCollectAllTemplate() ([]string, error) {
	query := `SELECT DISTINCT table_name FROM "__iql__.control.gc.txn_table_x_ref" ;`
	rows, err := sl.sqlEngine.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rv []string
	for {
		hasNext := rows.Next()
		if !hasNext {
			break
		}
		var s string
		err := rows.Scan(&s)
		if err != nil {
			return nil, err
		}
		rv = append(rv, fmt.Sprintf("delete from %s; ", s))
	}
	return rv, nil
}

func (sl *sqLiteDialect) getGCCollectTemplate(transactionIDs []int) ([]string, error) {
	var transactionIDStrings []string
	for _, txn := range transactionIDs {
		transactionIDStrings = append(transactionIDStrings, fmt.Sprintf("%d", txn))
	}
	var inBuilder strings.Builder
	inBuilder.WriteString("( ")
	inBuilder.WriteString(strings.Join(transactionIDStrings, ", "))
	inBuilder.WriteString(" )")
	query := fmt.Sprintf(`SELECT DISTINCT table_name, iql_transaction_id FROM "__iql__.control.gc.txn_table_x_ref" WHERE iql_transaction_id NOT IN %s AND table_name NOT LIKE ? ;`, inBuilder.String())
	rows, err := sl.sqlEngine.Query(query, sl.namespaces.GetAnalyticsCacheTableNamespaceConfigurator().GetLikeString())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rv []string
	for {
		hasNext := rows.Next()
		if !hasNext {
			break
		}
		var s string
		var i int
		err = rows.Scan(&s, &i)
		if err != nil {
			return nil, err
		}
		rv = append(rv, fmt.Sprintf(`delete from "%s" where iql_transaction_id  = %d ; `, s, i))
	}
	return rv, nil
}

func (sl *sqLiteDialect) getGCCollectCacheTemplate(transactionIDs []int) ([]string, error) {
	var transactionIDStrings []string
	for _, txn := range transactionIDs {
		transactionIDStrings = append(transactionIDStrings, fmt.Sprintf("%d", txn))
	}
	var inBuilder strings.Builder
	inBuilder.WriteString("( ")
	inBuilder.WriteString(strings.Join(transactionIDStrings, ", "))
	inBuilder.WriteString(" )")
	query := fmt.Sprintf(`SELECT DISTINCT table_name, iql_transaction_id FROM "__iql__.control.gc.txn_table_x_ref" WHERE iql_transaction_id NOT IN %s AND table_name LIKE ? ;`, inBuilder.String())
	rows, err := sl.sqlEngine.Query(query, sl.namespaces.GetAnalyticsCacheTableNamespaceConfigurator().GetLikeString())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rv []string
	for {
		hasNext := rows.Next()
		if !hasNext {
			break
		}
		var s string
		var i int
		err = rows.Scan(&s, &i)
		if err != nil {
			return nil, err
		}
		if sl.namespaces.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(s) {
			rv = append(rv, fmt.Sprintf(`delete from "%s" where iql_transaction_id  = %d and iql_latest_update <= datetime('now','-%d second'); `, s, i, sl.namespaces.GetAnalyticsCacheTableNamespaceConfigurator().GetTTL()))
		}
	}
	return rv, nil
}
