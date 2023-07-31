//nolint:revive,stylecheck // permissable deviation from norm
package http_preparator_stream

import (
	"github.com/stackql/go-openapistackql/openapistackql"
)

var (
	_ HttpPreparatorStream = &httpPreparatorStream{}
)

type HttpPreparatorStream interface {
	Write(openapistackql.HTTPPreparator) error
	Next() (openapistackql.HTTPPreparator, bool)
}

type httpPreparatorStream struct {
	sl []openapistackql.HTTPPreparator
}

func NewHttpPreparatorStream() HttpPreparatorStream {
	return &httpPreparatorStream{}
}

func (s *httpPreparatorStream) Write(p openapistackql.HTTPPreparator) error {
	s.sl = append(s.sl, p)
	return nil
}

func (s *httpPreparatorStream) Next() (openapistackql.HTTPPreparator, bool) {
	if len(s.sl) < 1 {
		return nil, false
	}
	p := s.sl[0]
	s.sl = s.sl[1:]
	return p, true
}
