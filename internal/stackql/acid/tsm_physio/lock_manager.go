package tsm_physio //nolint:stylecheck,revive // prefer this nomenclature

var (
	_ LockManager = (*lockManager)(nil)
)

type LockManager interface {
	//
}

type lockManager struct{}
