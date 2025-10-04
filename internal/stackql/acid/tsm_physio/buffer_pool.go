package tsm_physio //nolint:stylecheck,revive // prefer this nomenclature

var (
	_ BufferPool = (*bufferPool)(nil)
)

type BufferPool interface {
	//
}

type bufferPool struct{}
