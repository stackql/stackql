package internaldto

type ExecPayload struct {
	Payload    []byte
	Header     map[string][]string
	PayloadMap map[string]interface{}
}
