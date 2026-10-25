package dto

type MarshalledBody interface {
	GetBytes() []byte
	GetError() (error, bool)
}

type standardMarshalledBody struct {
	bytes []byte
	err   error
}

func NewMarshalledBody(b []byte, e error) MarshalledBody {
	return &standardMarshalledBody{
		bytes: b,
		err:   e,
	}
}

func (mb *standardMarshalledBody) GetBytes() []byte {
	return mb.bytes
}

func (mb *standardMarshalledBody) GetError() (error, bool) {
	return mb.err, mb.err != nil
}
