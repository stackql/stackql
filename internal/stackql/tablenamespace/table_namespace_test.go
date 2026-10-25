package tablenamespace_test

import (
	"testing"
	"testing/synctest"
	"time"

	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
)

type freshnessSystem struct {
	sql_system.SQLSystem
	stamp time.Time
	found bool
}

func (s *freshnessSystem) IsTablePresent(string, string, string) bool { return true }
func (s *freshnessSystem) TableOldestUpdateUTC(string, string, string, string) (time.Time, bool) {
	return s.stamp, s.found
}

func TestCacheExpiryNeedsNoTransactionCounters(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		collection, err := tablenamespace.NewStandardTableNamespaceCollection(map[string]dto.NamespaceCfg{
			"analytics": {TTL: 1, RegexpStr: "^(?P<objectName>.*)$", NamespaceTemplate: "cache_{{ .objectName }}"},
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		system := &freshnessSystem{stamp: time.Now(), found: true}
		if _, err = collection.WithSQLSystem(system); err != nil {
			t.Fatal(err)
		}
		cache := collection.GetAnalyticsCacheTableNamespaceConfigurator()
		if !cache.Match("resource", "request", "cached_at", "request_key") {
			t.Fatal("fresh entry missed")
		}
		time.Sleep(time.Second)
		if cache.Match("resource", "request", "cached_at", "request_key") {
			t.Fatal("expired entry served")
		}
		system.found = false
		if cache.Match("resource", "request", "cached_at", "request_key") {
			t.Fatal("missing timestamp served")
		}
	})
}
