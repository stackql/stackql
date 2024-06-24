package taxonomy

type AnnotationCtxSplitMap interface {
	Put(AnnotationCtx, AnnotationCtx)
	Get(AnnotationCtx) ([]AnnotationCtx, bool)
	Len() int
}

type annotationCtxSplitMap struct {
	m map[AnnotationCtx][]AnnotationCtx
}

func NewAnnotationCtxSplitMap() AnnotationCtxSplitMap {
	return &annotationCtxSplitMap{
		m: make(map[AnnotationCtx][]AnnotationCtx),
	}
}

func (am *annotationCtxSplitMap) Put(k, v AnnotationCtx) {
	_, ok := am.m[k]
	if ok {
		am.m[k] = append(am.m[k], v)
		return
	}
	am.m[k] = []AnnotationCtx{v}
}

func (am *annotationCtxSplitMap) Get(k AnnotationCtx) ([]AnnotationCtx, bool) {
	rv, ok := am.m[k]
	return rv, ok
}

func (am *annotationCtxSplitMap) Len() int {
	return len(am.m)
}
