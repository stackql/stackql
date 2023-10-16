package tsm_physio //nolint:revive,stylecheck // prefer this nomenclature

var (
	_ LockManager = (*lockManager)(nil)
)

type LockManager interface {
	//
}

type lockManager struct{}
