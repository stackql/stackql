package planbuilderinput

import (
	"regexp"

	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql/internal/stackql/nativedb"
	"github.com/stackql/stackql/internal/stackql/sqlstream"
)

var (
	multipleWhitespaceRegexp     *regexp.Regexp = regexp.MustCompile(`\s+`)
	getOidsRegexp                *regexp.Regexp = regexp.MustCompile(`(?i)select\s+t\.oid,\s+(?:NULL|typarray)\s+from.*pg_type`) //nolint:lll // long string
	selectPGCatalogVersionRegexp *regexp.Regexp = regexp.MustCompile(`(?i)select\s+pg_catalog\.version\(\)`)
	selectCurrentSchemaRegexp    *regexp.Regexp = regexp.MustCompile(`(?i)select\s+current_schema\(\)`)
	showTxnIsolationLevelRegexp  *regexp.Regexp = regexp.MustCompile(`(?i)show\s+transaction\s+isolation\s+level`)
)

// Deprecated
// TODO: Get rid ASAP
func IsPGSetupQuery(pbi PlanBuilderInput) (nativedb.Select, bool) {
	handlerCtx := pbi.GetHandlerCtx()
	routeType, canRoute := handlerCtx.GetDBMSInternalRouter().CanRoute(pbi.GetStatement())
	logging.GetLogger().Debugf("canRoute = %t, routeType = %v\n", canRoute, routeType)
	qStripped := multipleWhitespaceRegexp.ReplaceAllString(pbi.GetRawQuery(), " ")
	//nolint:lll // long string
	if qStripped == "select relname, nspname, relkind from pg_catalog.pg_class c, pg_catalog.pg_namespace n where relkind in ('r', 'v', 'm', 'f') and nspname not in ('pg_catalog', 'information_schema', 'pg_toast', 'pg_temp_1') and n.oid = relnamespace order by nspname, relname" {
		return nil, true
	}
	if qStripped == "select oid, typbasetype from pg_type where typname = 'lo'" {
		return nil, true
	}
	if getOidsRegexp.MatchString(qStripped) {
		var colz []nativedb.Column
		colz = append(colz, nativedb.NewColumn("oid", "oid"))
		colz = append(colz, nativedb.NewColumn("typarray", "oid"))
		return nativedb.NewSelect(colz), true
	}
	if selectPGCatalogVersionRegexp.MatchString(qStripped) {
		var colz []nativedb.Column
		colz = append(colz, nativedb.NewColumn("version", "text"))
		return nativedb.NewSelectWithRows(colz, sqlstream.NewStaticMapStream([]map[string]interface{}{
			//nolint:lll // long string
			{"version": "PostgreSQL 14.5 on x86_64-apple-darwin20.6.0, compiled by Apple clang version 13.0.0 (clang-1300.0.29.30), 64-bit"},
		})), true
	}
	if showTxnIsolationLevelRegexp.MatchString(qStripped) {
		var colz []nativedb.Column
		colz = append(colz, nativedb.NewColumn("transaction_isolation", "text"))
		return nativedb.NewSelectWithRows(colz, sqlstream.NewStaticMapStream([]map[string]interface{}{
			{"transaction_isolation": "read committed"},
		})), true
	}
	if selectCurrentSchemaRegexp.MatchString(qStripped) {
		var colz []nativedb.Column
		colz = append(colz, nativedb.NewColumn("current_schema", "text"))
		return nativedb.NewSelectWithRows(colz, sqlstream.NewStaticMapStream([]map[string]interface{}{
			{"current_schema": "public"},
		})), true
	}
	return nil, false
}
