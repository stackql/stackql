package primitive_context //nolint:revive,stylecheck // meaning of package name is clear

var (
	_ IPrimitiveCtx = &primitiveCtx{}
)

type IPrimitiveCtx interface {
	IsNotMutating() bool
	SetIsNotMutating(bool)
}

type primitiveCtx struct {
	isNotMutating bool
}

func NewPrimitiveContext() IPrimitiveCtx {
	return &primitiveCtx{}
}

func (pc *primitiveCtx) IsNotMutating() bool {
	return pc.isNotMutating
}

func (pc *primitiveCtx) SetIsNotMutating(isNotMutating bool) {
	pc.isNotMutating = isNotMutating
}
