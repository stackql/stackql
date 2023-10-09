package sqltable_test

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/stackql/stackql/internal/stackql/datasource/sqltable"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"
)

func randString() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	maxLength := 256
	stringLength := r.Intn(maxLength + 1) // Randomly decide the length of the string
	s := make([]byte, stringLength)
	for i := range s {
		s[i] = letters[r.Intn(len(letters))]
	}
	return string(s)
}

func generateRandomColumns(n int) []typing.RelationalColumn {
	columns := make([]typing.RelationalColumn, n)
	for i := range columns {
		// Assuming RelationalColumn is a type like string for simplicity
		columns[i] = typing.NewRelationalColumn(randString(), randString()) // Generate a random string of length 10
	}
	return columns
}

func TestNewStandardSQLTable(t *testing.T) {
	table, err := sqltable.NewStandardSQLTable(generateRandomColumns(10))
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if table == nil {
		t.Fatal("Expected table to be non-nil")
	}

	_, ok := table.(*sqltable.StandardSQLTable)
	if !ok {
		t.Fatal("Expected table to be of type *standardSQLTable")
	}
}

func TestGetSymTab(t *testing.T) {
	columns := generateRandomColumns(10)
	table, _ := sqltable.NewStandardSQLTable(columns)

	// Initialize symTab with different values
	symTab := table.GetSymTab()

	// Set symbols in the symTab
	err := symTab.SetSymbol("testKey", symtab.NewSymTabEntry("testType", "testData", "testIn"))
	if err != nil {
		t.Fatalf("Failed to set symbol: %v", err)
	}

	// Test if the symbol was set correctly
	entry, exists := symTab.GetSymbol("testKey")
	if exists != nil {
		t.Fatalf("Symbol not found in symTab")
	}
	if !reflect.DeepEqual(entry, symtab.NewSymTabEntry("testType", "testData", "testIn")) {
		t.Fatalf("Symbol not set correctly in symTab")
	}

	// Create a new leaf and set symbols in it
	leafSymTab, err := symTab.NewLeaf("testLeafKey")
	if err != nil {
		t.Fatalf("Failed to create new leaf: %v", err)
	}
	err = leafSymTab.SetSymbol("leafKey", symtab.NewSymTabEntry("leafType", "leafData", "leafIn"))
	if err != nil {
		t.Fatalf("Failed to set symbol in leaf: %v", err)
	}

	// Test if the symbol was set correctly in the leaf
	entry, exists = leafSymTab.GetSymbol("leafKey")
	if exists != nil {
		t.Fatalf("Symbol not found in leafSymTab")
	}
	if !reflect.DeepEqual(entry, symtab.NewSymTabEntry("leafType", "leafData", "leafIn")) {
		t.Fatalf("Symbol not set correctly in leafSymTab")
	}
}

func TestGetColumns(t *testing.T) {
	testCases := []struct {
		name       string
		numColumns int
	}{
		{
			name:       "Test with 0 columns",
			numColumns: 0,
		},
		{
			name:       "Test with 5 columns",
			numColumns: 5,
		},
		{
			name:       "Test with 10 columns",
			numColumns: 10,
		},
		{
			name:       "Test with 15 columns",
			numColumns: 15,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputColumns := generateRandomColumns(tc.numColumns)
			table, err := sqltable.NewStandardSQLTable(inputColumns)
			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			returnedColumns := table.GetColumns()
			if !reflect.DeepEqual(returnedColumns, inputColumns) {
				t.Fatalf("Expected columns %v, but got %v", inputColumns, returnedColumns)
			}
		})
	}
}
