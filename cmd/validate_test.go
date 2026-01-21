package cmd_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/samwightt/gqlx/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isValidationError(err error) bool {
	return err != nil && errors.Is(err, cmd.ErrValidationFailed)
}

const validateTestSchema = `
type User {
  id: ID!
  name: String!
  email: String!
  posts: [Post!]!
}

type Post {
  id: ID!
  title: String!
  author: User!
}

type Query {
  user(id: ID!): User
  users(limit: Int, offset: Int): [User!]!
  post(id: ID!): Post
}

type Mutation {
  createUser(name: String!, email: String!): User!
  updateUser(id: ID!, name: String, email: String): User
  deleteUser(id: ID!): Boolean!
}
`

func setupValidateTestSchema(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.graphql")
	err := os.WriteFile(schemaPath, []byte(validateTestSchema), 0644)
	require.NoError(t, err)
	return schemaPath
}

func writeValidateQuery(t *testing.T, dir string, query string) string {
	t.Helper()
	queryPath := filepath.Join(dir, "query.graphql")
	err := os.WriteFile(queryPath, []byte(query), 0644)
	require.NoError(t, err)
	return queryPath
}

func TestValidate_ValidSimpleQuery(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user(id: "123") {
				id
				name
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "✓ Query is valid")
}

func TestValidate_ValidQueryWithVariables(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query GetUser($userId: ID!) {
			user(id: $userId) {
				id
				name
				email
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "✓ Query is valid")
}

func TestValidate_ValidMutation(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		mutation CreateNewUser($name: String!, $email: String!) {
			createUser(name: $name, email: $email) {
				id
				name
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "✓ Query is valid")
}

func TestValidate_ValidNestedQuery(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user(id: "123") {
				id
				name
				posts {
					id
					title
					author {
						id
						name
					}
				}
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "✓ Query is valid")
}

func TestValidate_InvalidFieldSelection(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user(id: "123") {
				id
				nonexistent
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	assert.True(t, isValidationError(err), "expected validation error")
	assert.Contains(t, stdout, "✗ Query has")
	assert.Contains(t, stdout, "error")
}

func TestValidate_InvalidArgument(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user(id: "123", unknownArg: "test") {
				id
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	assert.True(t, isValidationError(err), "expected validation error")
	assert.Contains(t, stdout, "✗ Query has")
}

func TestValidate_MissingRequiredArgument(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user {
				id
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	assert.True(t, isValidationError(err), "expected validation error")
	assert.Contains(t, stdout, "✗ Query has")
}

func TestValidate_UnknownType(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			nonexistentField {
				id
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	assert.True(t, isValidationError(err), "expected validation error")
	assert.Contains(t, stdout, "✗ Query has")
}

func TestValidate_JSONFormat_Valid(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user(id: "123") {
				id
				name
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "json"})
	require.NoError(t, err)

	var result struct {
		Valid  bool `json:"valid"`
		Errors []struct {
			Message   string `json:"message"`
			Locations []struct {
				Line   int `json:"line"`
				Column int `json:"column"`
			} `json:"locations"`
		} `json:"errors"`
	}

	err = json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidate_JSONFormat_Invalid(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user(id: "123") {
				id
				nonexistent
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "json"})
	assert.True(t, isValidationError(err), "expected validation error")

	var result struct {
		Valid  bool `json:"valid"`
		Errors []struct {
			Message   string `json:"message"`
			Locations []struct {
				Line   int `json:"line"`
				Column int `json:"column"`
			} `json:"locations"`
		} `json:"errors"`
	}

	unmarshalErr := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, unmarshalErr)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "nonexistent")
}

func TestValidate_MultipleErrors(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user(id: "123") {
				id
				field1
				field2
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	assert.True(t, isValidationError(err), "expected validation error")
	assert.Contains(t, stdout, "✗ Query has 2 errors")
}

func TestValidate_SyntaxError(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user(id: "123") {
				id
				name

	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	assert.True(t, isValidationError(err), "expected validation error")
	assert.Contains(t, stdout, "✗ Query has")
}

func TestValidate_Stdin_Valid(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)

	query := `query { user(id: "123") { id name } }`
	stdin := bytes.NewBufferString(query)

	stdout, _, err := cmd.ExecuteWithArgsAndStdin([]string{"validate", "-s", schemaPath, "-f", "text"}, stdin)
	require.NoError(t, err)
	assert.Contains(t, stdout, "✓ Query is valid")
}

func TestValidate_Stdin_Invalid(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)

	query := `query { user(id: "123") { nonexistent } }`
	stdin := bytes.NewBufferString(query)

	stdout, _, err := cmd.ExecuteWithArgsAndStdin([]string{"validate", "-s", schemaPath, "-f", "text"}, stdin)
	assert.True(t, isValidationError(err), "expected validation error")
	assert.Contains(t, stdout, "✗ Query has")
}

func TestValidate_NonExistentFile(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)

	_, _, err := cmd.ExecuteWithArgs([]string{"validate", "/nonexistent/query.graphql", "-s", schemaPath})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read query file")
}

func TestValidate_NonExistentSchema(t *testing.T) {
	dir := os.TempDir()
	queryPath := filepath.Join(dir, "test_query.graphql")
	err := os.WriteFile(queryPath, []byte(`query { user { id } }`), 0644)
	require.NoError(t, err)
	defer os.Remove(queryPath)

	_, _, err = cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", "/nonexistent/schema.graphql"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestValidate_WrongArgumentType(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			users(limit: "not_an_int") {
				id
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	assert.True(t, isValidationError(err), "expected validation error")
	assert.Contains(t, stdout, "✗ Query has")
}

func TestValidate_FragmentSpread(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		fragment UserFields on User {
			id
			name
			email
		}

		query {
			user(id: "123") {
				...UserFields
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "✓ Query is valid")
}

func TestValidate_InvalidFragmentSpread(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		fragment PostFields on Post {
			id
			title
		}

		query {
			user(id: "123") {
				...PostFields
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	assert.True(t, isValidationError(err), "expected validation error")
	assert.Contains(t, stdout, "✗ Query has")
}

func TestValidate_InlineFragment(t *testing.T) {
	schemaPath := setupValidateTestSchema(t)
	dir := filepath.Dir(schemaPath)

	queryPath := writeValidateQuery(t, dir, `
		query {
			user(id: "123") {
				... on User {
					id
					name
				}
			}
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"validate", queryPath, "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "✓ Query is valid")
}
