package internaldto

type RawMap map[int]map[int]interface{}

type RawResult interface {
	GetMap() (RawMap, error)
}

type simpleRawResult struct {
	m RawMap
}

func (rr *simpleRawResult) GetMap() (RawMap, error) {
	return rr.m, nil
}

func createSimpleRawResult(m RawMap) RawResult {
	return &simpleRawResult{
		m: m,
	}
}

func createSimpleRawResultStream(m RawMap) IRawResultStream {
	return &SimpleRawResultStream{
		rr: createSimpleRawResult(m),
	}
}

type IRawResultStream interface {
	Read() (RawResult, error)
	IsNil() bool
}

type SimpleRawResultStream struct {
	rr RawResult
}

func (sr *SimpleRawResultStream) Read() (RawResult, error) {
	return sr.rr, nil
}

func (sr *SimpleRawResultStream) IsNil() bool {
	rm, err := sr.rr.GetMap()
	if err != nil {
		return true
	}
	return len(rm) < 1
}
