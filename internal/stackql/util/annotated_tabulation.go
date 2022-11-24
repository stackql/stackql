package util

import (
	"github.com/stackql/stackql/internal/stackql/dto"

	"github.com/stackql/go-openapistackql/openapistackql"
)

type AnnotatedTabulation struct {
	tab            *openapistackql.Tabulation
	hIds           *dto.HeirarchyIdentifiers
	inputTableName string
	alias          string
}

func NewAnnotatedTabulation(tab *openapistackql.Tabulation, hIds *dto.HeirarchyIdentifiers, inputTableName string, alias string) AnnotatedTabulation {
	return AnnotatedTabulation{
		tab:            tab,
		hIds:           hIds,
		inputTableName: inputTableName,
		alias:          alias,
	}
}

func (at AnnotatedTabulation) GetTabulation() *openapistackql.Tabulation {
	return at.tab
}

func (at AnnotatedTabulation) GetAlias() string {
	return at.alias
}

func (at AnnotatedTabulation) GetInputTableName() string {
	return at.inputTableName
}

func (at AnnotatedTabulation) GetHeirarchyIdentifiers() *dto.HeirarchyIdentifiers {
	return at.hIds
}
