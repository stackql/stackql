// Package garbagecollector retains explicit PURGE commands. Query tables are
// owned and eagerly released by omnistaging, independently of this package.
package garbagecollector

import "github.com/stackql/stackql/internal/stackql/sql_system"

type GarbageCollector interface {
	Collect() error
	Purge() error
	PurgeCache() error
	PurgeControlTables() error
	PurgeEphemeral() error
}

type standardGarbageCollector struct {
	system sql_system.SQLSystem
}

func NewGarbageCollector(system sql_system.SQLSystem) GarbageCollector {
	return &standardGarbageCollector{system: system}
}

// Collect is a compatibility operation: query owners already release their
// tables. Scanning and dropping query tables here could destroy another query.
func (gc *standardGarbageCollector) Collect() error { return nil }

// PurgeControlTables must not erase catalogue entries or reset query ID
// allocation. No legacy GC control tables are created or maintained anymore.
func (gc *standardGarbageCollector) PurgeControlTables() error { return nil }

func (gc *standardGarbageCollector) Purge() error      { return gc.system.GCPurgeCache() }
func (gc *standardGarbageCollector) PurgeCache() error { return gc.system.GCPurgeCache() }

// PurgeEphemeral retains the legacy command; query tables have their own owner.
func (gc *standardGarbageCollector) PurgeEphemeral() error { return nil }
