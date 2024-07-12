package streaming

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/stackql/stackql/pkg/maths"
)

type MapStreamCollection interface {
	MapStream
	Push(MapStream)
	Len() int
}

func NewStandardMapStreamCollection() MapStreamCollection {
	return &standardMapStreamCollection{}
}

type standardMapStreamCollection struct {
	store   []map[string]interface{}
	streams []MapStream
}

func (sc *standardMapStreamCollection) Push(stream MapStream) {
	sc.streams = append(sc.streams, stream)
}

func (sc *standardMapStreamCollection) Write(input []map[string]interface{}) error {
	// sc.store = append(sc.store, input...)
	var errSlice []error
	for _, stream := range sc.streams {
		if err := stream.Write(input); err != nil {
			errSlice = append(errSlice, err)
		}
	}
	if len(errSlice) > 0 {
		var sb strings.Builder
		for _, err := range errSlice {
			sb.WriteString(err.Error())
			sb.WriteString(";")
		}
		return fmt.Errorf(sb.String())
	}
	return nil
}

func (sc *standardMapStreamCollection) Len() int {
	streamLen := len(sc.streams)
	storeLen := len(sc.store)
	if streamLen > storeLen {
		return streamLen
	}
	return storeLen
}

func (sc *standardMapStreamCollection) Read() ([]map[string]interface{}, error) {
	var allOutputs [][]map[string]interface{}
	maxLength := 0
	// var allLengths []int
	storeLen := len(sc.store)
	if storeLen > 0 {
		// allLengths = append(allLengths, len(sc.store))
		maxLength = len(sc.store)
	}
	for _, stream := range sc.streams {
		output, err := stream.Read()
		if !errors.Is(err, io.EOF) {
			return output, err
		}
		thisLen := len(output)
		if thisLen == 0 {
			continue
		}
		allOutputs = append(allOutputs, output)
		// allLengths = append(allLengths, thisLen)
		if thisLen > maxLength {
			maxLength = thisLen
		}
	}
	if maxLength == 0 {
		return nil, io.EOF
	}
	// lcm := maths.LcmMultiple(allLengths...)
	rv := maths.CartesianProduct(allOutputs...)
	return rv, io.EOF
}
