package parser_test

import (
	"errors"
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	. "github.com/stackql/stackql/internal/stackql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// SQLParserMock is a mock implementation of the SQLParser interface.
type SQLParserMock struct {
	mock.Mock
}

// Parse is a mock implementation of the Parse method.
func (m *SQLParserMock) Parse(cmd string) (sqlparser.Statement, error) {
	args := m.Called(cmd)
	// If the first argument is not nil, return it as the statement
	if args.Get(0) != nil {
		return args.Get(0).(sqlparser.Statement), args.Error(1)
	}
	// Otherwise, return nil as the statement
	return nil, args.Error(1)
}

func TestNewParser(t *testing.T) {
	t.Run("NewParser", func(t *testing.T) {
		p, err := NewParser()
		assert.NotNil(t, p, "Parser should not be nil")
		assert.NoError(t, err, "Expected no error for NewParser")
	})
}

func TestParseQuery(t *testing.T) {
	// Test case for a valid SQL query
	t.Run("Valid SQL query", func(t *testing.T) {
		parserMock := new(SQLParserMock)
		validQuery := "SELECT * FROM table_name;"
		expectedStatement, _ := sqlparser.Parse("SELECT * FROM table_name;")
		parserMock.On("Parse", validQuery).Return(expectedStatement, nil)

		parser, err := NewParser()
		assert.NoError(t, err, "Expected no error for NewParser")
		statement, err := parser.ParseQuery(validQuery)

		assert.NoError(t, err, "Expected no error for valid SQL query: %v", t)
		assert.NotNil(t, statement, "Expected statement to be returned for valid SQL query")
		assert.Equal(t, expectedStatement, statement, "Expected statement to match the parsed statement")
	})

	// Test case for an invalid SQL query
	t.Run("Invalid SQL query", func(t *testing.T) {
		mockParser := new(SQLParserMock)
		invalidQuery := "SELECTTT * FROM table_name WHERE;"
		expectedError := errors.New("error:  You have an error in your stackql syntax; parser error")
		mockParser.On("Parse", invalidQuery).Return(nil, expectedError)

		parser, err := NewParser()
		assert.NoError(t, err, "Expected no error for NewParser")
		statement, err := parser.ParseQuery(invalidQuery)

		assert.Error(t, err, "Expected an error for invalid SQL query")
		assert.Nil(t, statement, "Expected no statement for invalid SQL query")
	})
}
