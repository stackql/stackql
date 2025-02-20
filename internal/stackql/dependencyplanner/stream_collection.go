package dependencyplanner

import (
	"github.com/stackql/any-sdk/pkg/streaming"
)

type StreamDependecyCollection interface {
	// Add adds a stream to the collection
	// departingID is the ID of the stream that is departing
	// arrivingID is the ID of the stream that is arriving
	// stream is the stream that is being added
	Add(departingID int64, arrivingID int64, stream streaming.MapStream)
	GetArriving(int64) streaming.MapStreamCollection
	GetDeparting(int64) streaming.MapStreamCollection
}

type streamDependencyCollection struct {
	departingStreams map[int64][]streaming.MapStream
	arrivingStreams  map[int64][]streaming.MapStream
}

func NewStreamDependecyCollection() StreamDependecyCollection {
	return &streamDependencyCollection{
		departingStreams: make(map[int64][]streaming.MapStream),
		arrivingStreams:  make(map[int64][]streaming.MapStream),
	}
}

func (sdc *streamDependencyCollection) Add(departingID int64, arrivingID int64, stream streaming.MapStream) {
	sdc.departingStreams[departingID] = append(sdc.departingStreams[departingID], stream)
	sdc.arrivingStreams[arrivingID] = append(sdc.arrivingStreams[arrivingID], stream)
}

func (sdc *streamDependencyCollection) GetArriving(id int64) streaming.MapStreamCollection {
	rv := streaming.NewStandardMapStreamCollection()
	for _, stream := range sdc.arrivingStreams[id] {
		rv.Push(stream)
	}
	if rv.Len() == 0 {
		rv.Push(streaming.NewStandardMapStream())
	}
	return rv
}

func (sdc *streamDependencyCollection) GetDeparting(id int64) streaming.MapStreamCollection {
	rv := streaming.NewStandardMapStreamCollection()
	for _, stream := range sdc.departingStreams[id] {
		rv.Push(stream)
	}
	if rv.Len() == 0 {
		rv.Push(streaming.NewNopMapStream())
	}
	return rv
}
