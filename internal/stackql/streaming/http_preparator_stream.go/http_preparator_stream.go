//nolint:revive,stylecheck // permissable deviation from norm
package http_preparator_stream

import "github.com/stackql/any-sdk/anysdk"

var (
	_ HttpPreparatorStream = &httpPreparatorStream{}
)

type HttpPreparatorStream interface {
	Write(anysdk.HTTPPreparator) error
	Next() (anysdk.HTTPPreparator, bool)
}

type httpPreparatorStream struct {
	sl []anysdk.HTTPPreparator
}

func NewHttpPreparatorStream() HttpPreparatorStream {
	return &httpPreparatorStream{}
}

func (s *httpPreparatorStream) Write(p anysdk.HTTPPreparator) error {
	s.sl = append(s.sl, p)
	return nil
}

func (s *httpPreparatorStream) Next() (anysdk.HTTPPreparator, bool) {
	if len(s.sl) < 1 {
		return nil, false
	}
	p := s.sl[0]
	s.sl = s.sl[1:]
	return p, true
}
