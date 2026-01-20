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

func TestFields_AllFields_Text(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			name: String!
		}

		type Post {
			id: ID!
			title: String!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should include fields with type prefix
	assert.Contains(t, stdout, "User.id: ID!")
	assert.Contains(t, stdout, "User.name: String!")
	assert.Contains(t, stdout, "Post.id: ID!")
	assert.Contains(t, stdout, "Post.title: String!")
	assert.Contains(t, stdout, "Query.user: User")
}

func TestFields_AllFields_JSON(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			name: String!
		}

		type Post {
			title: String!
		}

		type Query {
			dummy: String
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "json"})
	require.NoError(t, err)

	var fields []struct {
		TypeName string `json:"typeName"`
		Name     string `json:"name"`
		Type     string `json:"type"`
	}

	err = json.Unmarshal([]byte(stdout), &fields)
	require.NoError(t, err)

	// Should have fields from User, Post, Query, and built-in types
	assert.Greater(t, len(fields), 3)

	// Check that typeName is populated
	fieldMap := make(map[string]string)
	for _, f := range fields {
		key := f.TypeName + "." + f.Name
		fieldMap[key] = f.Type
	}

	assert.Equal(t, "ID!", fieldMap["User.id"])
	assert.Equal(t, "String!", fieldMap["User.name"])
	assert.Equal(t, "String!", fieldMap["Post.title"])
}

func TestFields_AllFields_Pretty(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			name: String!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "pretty"})
	require.NoError(t, err)

	// Pretty format should have table elements
	assert.Contains(t, stdout, "─")
	assert.Contains(t, stdout, "│")

	// Should include fields with type prefix
	assert.Contains(t, stdout, "User.id")
	assert.Contains(t, stdout, "User.name")
}

func TestFields_DeprecatedFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			name: String!
			oldField: String @deprecated(reason: "Use newField instead")
			newField: String!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--deprecated", "User"})
	require.NoError(t, err)

	// Should only include deprecated fields
	assert.Contains(t, stdout, "oldField: String")

	// Should NOT include non-deprecated fields
	assert.NotContains(t, stdout, "id:")
	assert.NotContains(t, stdout, "name:")
	assert.NotContains(t, stdout, "newField:")
}

func TestFields_DeprecatedFilter_AllTypes(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			legacyId: Int @deprecated
		}

		type Post {
			id: ID!
			oldTitle: String @deprecated(reason: "Use title instead")
			title: String!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--deprecated"})
	require.NoError(t, err)

	// Should include deprecated fields with type prefix
	assert.Contains(t, stdout, "User.legacyId: Int")
	assert.Contains(t, stdout, "Post.oldTitle: String")

	// Should NOT include non-deprecated fields
	assert.NotContains(t, stdout, "User.id:")
	assert.NotContains(t, stdout, "Post.id:")
	assert.NotContains(t, stdout, "Post.title:")
}

func TestFields_DeprecatedFilter_NoMatches(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			name: String!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "json", "--deprecated", "User"})
	require.NoError(t, err)

	var fields []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &fields)
	require.NoError(t, err)

	assert.Len(t, fields, 0)
}

func TestFields_HasArgFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
			users(limit: Int, offset: Int): [User!]!
			allUsers: [User!]!
		}

		type User {
			id: ID!
			name: String!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--has-arg", "id", "Query"})
	require.NoError(t, err)

	// Should include fields with "id" argument
	assert.Contains(t, stdout, "user(id: ID!): User")

	// Should NOT include fields without "id" argument
	assert.NotContains(t, stdout, "users(")
	assert.NotContains(t, stdout, "allUsers")
}

func TestFields_HasArgFilter_MultipleArgs(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			search(query: String!, limit: Int): [Result!]!
			paginate(limit: Int, offset: Int): [Result!]!
			findAll(query: String!, limit: Int, offset: Int): [Result!]!
		}

		type Result {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--has-arg", "query", "--has-arg", "limit", "Query"})
	require.NoError(t, err)

	// Should include fields with BOTH "query" AND "limit" arguments
	assert.Contains(t, stdout, "search(")
	assert.Contains(t, stdout, "findAll(")

	// Should NOT include fields missing either argument
	assert.NotContains(t, stdout, "paginate(")
}

func TestFields_HasArgFilter_AllTypes(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
			post(id: ID!): Post
		}

		type User {
			id: ID!
			posts(limit: Int): [Post!]!
		}

		type Post {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--has-arg", "id"})
	require.NoError(t, err)

	// Should include fields with "id" argument with type prefix
	assert.Contains(t, stdout, "Query.user(id: ID!): User")
	assert.Contains(t, stdout, "Query.post(id: ID!): Post")

	// Should NOT include fields without "id" argument
	assert.NotContains(t, stdout, "User.posts")
	assert.NotContains(t, stdout, "User.id")
	assert.NotContains(t, stdout, "Post.id")
}

func TestFields_HasArgFilter_NoMatches(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			users: [User!]!
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "json", "--has-arg", "id", "Query"})
	require.NoError(t, err)

	var fields []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &fields)
	require.NoError(t, err)

	assert.Len(t, fields, 0)
}

func TestFields_ReturnsFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
			users: [User!]!
			post(id: ID!): Post
		}

		type User {
			id: ID!
			name: String!
		}

		type Post {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--returns", "User", "Query"})
	require.NoError(t, err)

	// Should include fields returning User (including lists)
	assert.Contains(t, stdout, "user(id: ID!): User")
	assert.Contains(t, stdout, "users: [User!]!")

	// Should NOT include fields returning other types
	assert.NotContains(t, stdout, "post(")
}

func TestFields_ReturnsFilter_AllTypes(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
			post: Post
		}

		type User {
			id: ID!
			friends: [User!]!
		}

		type Post {
			id: ID!
			author: User!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--returns", "User"})
	require.NoError(t, err)

	// Should include all fields returning User with type prefix
	assert.Contains(t, stdout, "Query.user: User")
	assert.Contains(t, stdout, "User.friends: [User!]!")
	assert.Contains(t, stdout, "Post.author: User!")

	// Should NOT include fields returning other types
	assert.NotContains(t, stdout, "Query.post:")
	assert.NotContains(t, stdout, "User.id:")
	assert.NotContains(t, stdout, "Post.id:")
}

func TestFields_ReturnsFilter_Scalars(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			name: String!
			email: String
			age: Int
		}

		type Query {
			dummy: String
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--returns", "String", "User"})
	require.NoError(t, err)

	// Should include fields returning String
	assert.Contains(t, stdout, "name: String!")
	assert.Contains(t, stdout, "email: String")

	// Should NOT include fields returning other types
	assert.NotContains(t, stdout, "id:")
	assert.NotContains(t, stdout, "age:")
}

func TestFields_ReturnsFilter_NoMatches(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "json", "--returns", "Post", "Query"})
	require.NoError(t, err)

	var fields []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &fields)
	require.NoError(t, err)

	assert.Len(t, fields, 0)
}

func TestFields_RequiredFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			name: String!
			email: String
			age: Int
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--required", "User"})
	require.NoError(t, err)

	// Should include required (non-null) fields
	assert.Contains(t, stdout, "id: ID!")
	assert.Contains(t, stdout, "name: String!")

	// Should NOT include nullable fields
	assert.NotContains(t, stdout, "email:")
	assert.NotContains(t, stdout, "age:")
}

func TestFields_NullableFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			name: String!
			email: String
			age: Int
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--nullable", "User"})
	require.NoError(t, err)

	// Should include nullable fields
	assert.Contains(t, stdout, "email: String")
	assert.Contains(t, stdout, "age: Int")

	// Should NOT include required fields
	assert.NotContains(t, stdout, "id:")
	assert.NotContains(t, stdout, "name:")
}

func TestFields_RequiredFilter_AllTypes(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
			nickname: String
		}

		type Post {
			title: String!
			subtitle: String
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--required"})
	require.NoError(t, err)

	// Should include required fields with type prefix
	assert.Contains(t, stdout, "User.id: ID!")
	assert.Contains(t, stdout, "Post.title: String!")

	// Should NOT include nullable fields
	assert.NotContains(t, stdout, "User.nickname")
	assert.NotContains(t, stdout, "Post.subtitle")
}

func TestFields_RequiredAndNullable_MutuallyExclusive(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"fields", "-s", schemaPath, "-f", "text", "--required", "--nullable", "User"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used together")
}
