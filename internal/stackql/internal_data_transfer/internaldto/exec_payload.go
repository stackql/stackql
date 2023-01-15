package internaldto

var (
	_ ExecPayload = &standardExecPayload{}
)

type ExecPayload interface {
	GetHeader() map[string][]string
	GetPayload() []byte
	GetPayloadMap() map[string]interface{}
	SetHeaderKV(k string, v []string)
}

func NewExecPayload(payload []byte, header map[string][]string, payloadMap map[string]interface{}) ExecPayload {
	return &standardExecPayload{
		payload:    payload,
		header:     header,
		payloadMap: payloadMap,
	}
}

type standardExecPayload struct {
	payload    []byte
	header     map[string][]string
	payloadMap map[string]interface{}
}

func (ep *standardExecPayload) SetHeaderKV(k string, v []string) {
	ep.header[k] = v
}

func (ep *standardExecPayload) GetPayload() []byte {
	return ep.payload
}

func (ep *standardExecPayload) GetHeader() map[string][]string {
	return ep.header
}

func (ep *standardExecPayload) GetPayloadMap() map[string]interface{} {
	return ep.payloadMap
}
