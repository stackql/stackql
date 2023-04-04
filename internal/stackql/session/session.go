package session

type Session interface {
	// ExecuteQuery(query string) (interface{}, error)
	ExecuteQuery(query string) (interface{}, error)
}
