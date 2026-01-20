package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samwightt/gqlx/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSchema = `
type User {
  "The unique identifier"
  id: ID!
  "The user's name"
  name: String!
  "The user's email address"
  email: String
}

input CreateUserInput {
  "The user's name"
  name: String!
  "The user's email address"
  email: String!
  "The user's age"
  age: Int = 18
}

type Query {
  "Fetch a user by ID"
  user(id: ID!): User
  "Search users by name"
  users(query: String!, limit: Int, offset: Int): [User!]!
}

type Mutation {
  "Create a new user"
  createUser(input: CreateUserInput!): User!
}
`

func setupTestSchema(t *testing.T) string {
	t.Helper()
	return writeTestSchema(t, testSchema)
}

func writeTestSchema(t *testing.T, schema string) string {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.graphql")
	err := os.WriteFile(schemaPath, []byte(schema), 0644)
	require.NoError(t, err)
	return schemaPath
}

func TestFields_TextFormat(t *testing.T) {
	schemaPath := setupTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "User"})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	assert.Len(t, lines, 3)

	assert.Contains(t, stdout, "id: ID!")
	assert.Contains(t, stdout, "name: String!")
	assert.Contains(t, stdout, "email: String")
	assert.Contains(t, stdout, "# The unique identifier")
	assert.Contains(t, stdout, "# The user's name")
}

func TestFields_TextFormat_WithArguments(t *testing.T) {
	schemaPath := setupTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "Query"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "user(id: ID!): User")
	assert.Contains(t, stdout, "users(query: String!, limit: Int, offset: Int): [User!]!")
}

func TestFields_JSONFormat(t *testing.T) {
	schemaPath := setupTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "json", "User"})
	require.NoError(t, err)

	var fields []struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description,omitempty"`
		Arguments   []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"arguments,omitempty"`
	}

	err = json.Unmarshal([]byte(stdout), &fields)
	require.NoError(t, err)

	assert.Len(t, fields, 3)

	idField := fields[0]
	assert.Equal(t, "id", idField.Name)
	assert.Equal(t, "ID!", idField.Type)
	assert.Equal(t, "The unique identifier", idField.Description)
}

func TestFields_JSONFormat_WithArguments(t *testing.T) {
	schemaPath := setupTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "json", "Query"})
	require.NoError(t, err)

	var fields []struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Arguments []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"arguments,omitempty"`
	}

	err = json.Unmarshal([]byte(stdout), &fields)
	require.NoError(t, err)

	// Find the users field
	var usersField struct {
		Name      string `json:"name"`
		Arguments []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"arguments,omitempty"`
	}
	for _, f := range fields {
		if f.Name == "users" {
			usersField.Name = f.Name
			usersField.Arguments = f.Arguments
			break
		}
	}

	assert.Equal(t, "users", usersField.Name)
	assert.Len(t, usersField.Arguments, 3)
	assert.Equal(t, "query", usersField.Arguments[0].Name)
	assert.Equal(t, "String!", usersField.Arguments[0].Type)
}

func TestFields_PrettyFormat(t *testing.T) {
	schemaPath := setupTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "pretty", "User"})
	require.NoError(t, err)

	// Pretty format should have table borders
	assert.Contains(t, stdout, "─")
	assert.Contains(t, stdout, "│")

	// Should have headers
	assert.Contains(t, stdout, "field")
	assert.Contains(t, stdout, "type")
	assert.Contains(t, stdout, "description")

	// Should have data
	assert.Contains(t, stdout, "id")
	assert.Contains(t, stdout, "ID!")
}

func TestFields_NonExistentType(t *testing.T) {
	schemaPath := setupTestSchema(t)

	_, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "NonExistent"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestFields_NonExistentType_DidYouMean(t *testing.T) {
	schemaPath := setupTestSchema(t)

	_, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "Usr"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "User")
}

func TestFields_InvalidFormat(t *testing.T) {
	schemaPath := setupTestSchema(t)

	_, stderr, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "invalid", "User"})

	assert.Error(t, err)
	assert.Contains(t, stderr, "invalid format")
}

func TestFields_NonExistentSchema(t *testing.T) {
	_, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", "/nonexistent/schema.graphql", "-f", "text", "User"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestFields_InvalidGraphQLSchema(t *testing.T) {
	schemaPath := writeTestSchema(t, `this is not valid graphql {{{`)

	_, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "User"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing error")
}

func TestFields_InputObject_Text(t *testing.T) {
	schemaPath := setupTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "CreateUserInput"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "name: String!")
	assert.Contains(t, stdout, "email: String!")
	assert.Contains(t, stdout, "age: Int = 18")
	assert.Contains(t, stdout, "# The user's name")
	assert.Contains(t, stdout, "# The user's email address")
	assert.Contains(t, stdout, "# The user's age")
}

func TestFields_PrettyFormat_WithArguments(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			search(query: String!, limit: Int, offset: Int): [String!]!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "pretty", "Query"})
	require.NoError(t, err)

	// Pretty format should show arguments on multiple lines
	assert.Contains(t, stdout, "search(")
	assert.Contains(t, stdout, "query: String!")
	assert.Contains(t, stdout, "limit: Int")
	assert.Contains(t, stdout, "offset: Int")
}

func TestFields_InputObject_JSON(t *testing.T) {
	schemaPath := setupTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "json", "CreateUserInput"})
	require.NoError(t, err)

	var fields []struct {
		Name         string `json:"name"`
		Type         string `json:"type"`
		DefaultValue string `json:"defaultValue,omitempty"`
		Description  string `json:"description,omitempty"`
	}

	err = json.Unmarshal([]byte(stdout), &fields)
	require.NoError(t, err)

	assert.Len(t, fields, 3)

	// Check we have the expected fields
	fieldMap := make(map[string]struct {
		Type         string
		DefaultValue string
	})
	for _, f := range fields {
		fieldMap[f.Name] = struct {
			Type         string
			DefaultValue string
		}{f.Type, f.DefaultValue}
	}

	assert.Equal(t, "String!", fieldMap["name"].Type)
	assert.Equal(t, "", fieldMap["name"].DefaultValue)

	assert.Equal(t, "String!", fieldMap["email"].Type)
	assert.Equal(t, "", fieldMap["email"].DefaultValue)

	assert.Equal(t, "Int", fieldMap["age"].Type)
	assert.Equal(t, "18", fieldMap["age"].DefaultValue)
}
