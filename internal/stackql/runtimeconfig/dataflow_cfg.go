package dto

var (
	_ DataFlowCfg = &dataFlowCfg{}
)

type DataFlowCfg interface {
	GetMaxDependencies() int
	GetMaxComponents() int
}

type dataFlowCfg struct {
	maxDependencies int
	maxComponents   int
}

func NewDataFlowCfg(maxDependencies int, maxComponents int) DataFlowCfg {
	return &dataFlowCfg{
		maxDependencies: maxDependencies,
		maxComponents:   maxComponents,
	}
}

func (d *dataFlowCfg) GetMaxDependencies() int {
	return d.maxDependencies
}

func (d *dataFlowCfg) GetMaxComponents() int {
	return d.maxComponents
}
