package dto

type ContextKey string

const (
	ContextPrefixStackqlRequest string = "stackql/request/"
)

const (
	ContextKeyCreationDate ContextKey = "stackql/request/created_date"
)
