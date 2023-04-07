package primitive_context //nolint:revive,stylecheck // meaning of package name is clear

var (
	_ IPrimitiveCtx = &primitiveCtx{}
)

type IPrimitiveCtx interface {
	IsReadOnly() bool
	SetIsReadOnly(bool)
}

type primitiveCtx struct {
	isReadOnly bool
}

func NewPrimitiveContext() IPrimitiveCtx {
	return &primitiveCtx{}
}

func (pc *primitiveCtx) IsReadOnly() bool {
	return pc.isReadOnly
}

func (pc *primitiveCtx) SetIsReadOnly(isReadOnly bool) {
	pc.isReadOnly = isReadOnly
}
