package tsm_physio //nolint:revive,stylecheck // prefer this nomenclature

var (
	_ BufferPool = (*bufferPool)(nil)
)

type BufferPool interface {
	//
}

type bufferPool struct{}
