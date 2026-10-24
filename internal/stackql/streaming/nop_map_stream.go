package streaming

func NewNopMapStream() MapStream {
	return &standardMapStream{}
}
