package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/samwightt/gqlx/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const typesTestSchema = `
"A user in the system"
type User {
  id: ID!
  name: String!
}

"Input for creating a user"
input CreateUserInput {
  name: String!
}

"Possible statuses"
enum Status {
  ACTIVE
  INACTIVE
}

"A node interface"
interface Node {
  id: ID!
}

"Search result union"
union SearchResult = User

type Query {
  user(id: ID!): User
}

type Mutation {
  createUser(input: CreateUserInput!): User!
}
`

func setupTypesTestSchema(t *testing.T) string {
	t.Helper()
	return writeTypesTestSchema(t, typesTestSchema)
}

func writeTypesTestSchema(t *testing.T, schema string) string {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.graphql")
	err := os.WriteFile(schemaPath, []byte(schema), 0644)
	require.NoError(t, err)
	return schemaPath
}

func TestTypes_TextFormat(t *testing.T) {
	schemaPath := setupTypesTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Check our custom types are present with their kinds
	assert.Contains(t, stdout, "type User")
	assert.Contains(t, stdout, "input CreateUserInput")
	assert.Contains(t, stdout, "enum Status")
	assert.Contains(t, stdout, "interface Node")
	assert.Contains(t, stdout, "union SearchResult")
	assert.Contains(t, stdout, "type Query")
	assert.Contains(t, stdout, "type Mutation")

	// Check descriptions are included
	assert.Contains(t, stdout, "# A user in the system")
	assert.Contains(t, stdout, "# Input for creating a user")
}

func TestTypes_JSONFormat(t *testing.T) {
	schemaPath := setupTypesTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "json"})
	require.NoError(t, err)

	var types []struct {
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Description string `json:"description,omitempty"`
	}

	err = json.Unmarshal([]byte(stdout), &types)
	require.NoError(t, err)

	// Build a map for easier assertions
	typeMap := make(map[string]struct {
		Kind        string
		Description string
	})
	for _, t := range types {
		typeMap[t.Name] = struct {
			Kind        string
			Description string
		}{t.Kind, t.Description}
	}

	// Check our custom types
	assert.Equal(t, "OBJECT", typeMap["User"].Kind)
	assert.Equal(t, "A user in the system", typeMap["User"].Description)

	assert.Equal(t, "INPUT_OBJECT", typeMap["CreateUserInput"].Kind)
	assert.Equal(t, "Input for creating a user", typeMap["CreateUserInput"].Description)

	assert.Equal(t, "ENUM", typeMap["Status"].Kind)
	assert.Equal(t, "Possible statuses", typeMap["Status"].Description)

	assert.Equal(t, "INTERFACE", typeMap["Node"].Kind)
	assert.Equal(t, "A node interface", typeMap["Node"].Description)

	assert.Equal(t, "UNION", typeMap["SearchResult"].Kind)
	assert.Equal(t, "Search result union", typeMap["SearchResult"].Description)

	assert.Equal(t, "OBJECT", typeMap["Query"].Kind)
	assert.Equal(t, "OBJECT", typeMap["Mutation"].Kind)
}

func TestTypes_PrettyFormat(t *testing.T) {
	schemaPath := setupTypesTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "pretty"})
	require.NoError(t, err)

	// Pretty format should have table borders
	assert.Contains(t, stdout, "─")
	assert.Contains(t, stdout, "│")

	// Should have headers
	assert.Contains(t, stdout, "kind")
	assert.Contains(t, stdout, "name")
	assert.Contains(t, stdout, "description")

	// Should have data
	assert.Contains(t, stdout, "type")
	assert.Contains(t, stdout, "User")
	assert.Contains(t, stdout, "input")
	assert.Contains(t, stdout, "enum")
	assert.Contains(t, stdout, "interface")
	assert.Contains(t, stdout, "union")
}

func TestTypes_IncludesBuiltInTypes(t *testing.T) {
	schemaPath := setupTypesTestSchema(t)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should include built-in scalar types
	assert.Contains(t, stdout, "scalar String")
	assert.Contains(t, stdout, "scalar Int")
	assert.Contains(t, stdout, "scalar Boolean")
	assert.Contains(t, stdout, "scalar ID")
}

func TestTypes_NonExistentSchema(t *testing.T) {
	_, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", "/nonexistent/schema.graphql", "-f", "text"})
	assert.Error(t, err)
}

func TestTypes_ImplementsFilter(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		interface Node {
			id: ID!
		}

		type User implements Node {
			id: ID!
			name: String!
		}

		type Post implements Node {
			id: ID!
			title: String!
		}

		type Comment {
			id: ID!
			text: String!
		}

		type Query {
			node(id: ID!): Node
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--implements", "Node"})
	require.NoError(t, err)

	// Should include types that implement Node
	assert.Contains(t, stdout, "type User")
	assert.Contains(t, stdout, "type Post")

	// Should NOT include types that don't implement Node
	assert.NotContains(t, stdout, "type Comment")
	assert.NotContains(t, stdout, "type Query")
	assert.NotContains(t, stdout, "interface Node")
}

func TestTypes_ImplementsFilter_JSON(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		interface Node {
			id: ID!
		}

		type User implements Node {
			id: ID!
		}

		type Other {
			id: ID!
		}

		type Query {
			dummy: String
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "json", "--implements", "Node"})
	require.NoError(t, err)

	var types []struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}

	err = json.Unmarshal([]byte(stdout), &types)
	require.NoError(t, err)

	// Should only have User
	assert.Len(t, types, 1)
	assert.Equal(t, "User", types[0].Name)
}

func TestTypes_ImplementsFilter_NoMatches(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		interface Node {
			id: ID!
		}

		type User {
			id: ID!
		}

		type Query {
			dummy: String
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "json", "--implements", "Node"})
	require.NoError(t, err)

	var types []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &types)
	require.NoError(t, err)

	assert.Len(t, types, 0)
}

func TestTypes_ImplementsFilter_InterfaceNotFound(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			dummy: String
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--implements", "Node"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestTypes_ImplementsFilter_DidYouMean(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		interface Node {
			id: ID!
		}

		type User implements Node {
			id: ID!
		}

		type Query {
			dummy: String
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--implements", "Nod"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "Node")
}

func TestTypes_ImplementsFilter_NotAnInterface(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			dummy: String
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--implements", "User"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an interface")
}

func TestTypes_KindFilter_Single(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
		}

		enum Status {
			ACTIVE
			INACTIVE
		}

		interface Node {
			id: ID!
		}

		input CreateUserInput {
			name: String!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--kind", "enum"})
	require.NoError(t, err)

	// Should only include enums
	assert.Contains(t, stdout, "enum Status")

	// Should NOT include other kinds
	assert.NotContains(t, stdout, "type User")
	assert.NotContains(t, stdout, "interface Node")
	assert.NotContains(t, stdout, "input CreateUserInput")
}

func TestTypes_KindFilter_Multiple(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
		}

		enum Status {
			ACTIVE
		}

		interface Node {
			id: ID!
		}

		input CreateUserInput {
			name: String!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--kind", "enum", "--kind", "interface"})
	require.NoError(t, err)

	// Should include enums and interfaces
	assert.Contains(t, stdout, "enum Status")
	assert.Contains(t, stdout, "interface Node")

	// Should NOT include types or inputs
	assert.NotContains(t, stdout, "type User")
	assert.NotContains(t, stdout, "input CreateUserInput")
}

func TestTypes_KindFilter_Type(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
		}

		type Post {
			id: ID!
		}

		enum Status {
			ACTIVE
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--kind", "type"})
	require.NoError(t, err)

	// Should include object types
	assert.Contains(t, stdout, "type User")
	assert.Contains(t, stdout, "type Post")
	assert.Contains(t, stdout, "type Query")

	// Should NOT include enums
	assert.NotContains(t, stdout, "enum Status")
}

func TestTypes_KindFilter_Input(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
		}

		input CreateUserInput {
			name: String!
		}

		input UpdateUserInput {
			name: String
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--kind", "input"})
	require.NoError(t, err)

	// Should include input types
	assert.Contains(t, stdout, "input CreateUserInput")
	assert.Contains(t, stdout, "input UpdateUserInput")

	// Should NOT include object types
	assert.NotContains(t, stdout, "type User")
	assert.NotContains(t, stdout, "type Query")
}

func TestTypes_KindFilter_CombinedWithHasField(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
			name: String!
		}

		type Post {
			id: ID!
			title: String!
		}

		interface Node {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--kind", "type", "--has-field", "name"})
	require.NoError(t, err)

	// Should only include types with "name" field
	assert.Contains(t, stdout, "type User")

	// Should NOT include types without "name" or non-object types
	assert.NotContains(t, stdout, "type Post")
	assert.NotContains(t, stdout, "type Query")
	assert.NotContains(t, stdout, "interface Node")
}

func TestTypes_KindFilter_JSON(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
		}

		enum Status {
			ACTIVE
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "json", "--kind", "enum"})
	require.NoError(t, err)

	var types []struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}

	err = json.Unmarshal([]byte(stdout), &types)
	require.NoError(t, err)

	// Should include our Status enum (plus built-in enums)
	var foundStatus bool
	for _, typ := range types {
		assert.Equal(t, "ENUM", typ.Kind)
		if typ.Name == "Status" {
			foundStatus = true
		}
	}
	assert.True(t, foundStatus, "Expected to find Status enum")
}

func TestTypes_KindFilter_CaseInsensitive(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		enum Status {
			ACTIVE
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--kind", "ENUM"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "enum Status")
}

func TestTypes_UsedByFilter(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
			name: String!
			posts: [Post!]!
		}

		type Post {
			id: ID!
			title: String!
		}

		type Comment {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--used-by", "User"})
	require.NoError(t, err)

	// Should include types used by User (in fields)
	assert.Contains(t, stdout, "scalar ID")
	assert.Contains(t, stdout, "scalar String")
	assert.Contains(t, stdout, "type Post")

	// Should NOT include types not used by User
	assert.NotContains(t, stdout, "type Comment")
	assert.NotContains(t, stdout, "type Query")
}

func TestTypes_UsedByFilter_WithArguments(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
			posts(limit: Int, filter: PostFilter): [Post!]!
		}

		type Post {
			id: ID!
		}

		input PostFilter {
			title: String
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--used-by", "User"})
	require.NoError(t, err)

	// Should include types used in field arguments
	assert.Contains(t, stdout, "scalar Int")
	assert.Contains(t, stdout, "input PostFilter")

	// Should include types used in field return types
	assert.Contains(t, stdout, "type Post")
	assert.Contains(t, stdout, "scalar ID")
}

func TestTypes_UsedByFilter_CombinedWithKind(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
			name: String!
			status: Status
		}

		enum Status {
			ACTIVE
			INACTIVE
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--used-by", "User", "--kind", "enum"})
	require.NoError(t, err)

	// Should only include enums used by User
	assert.Contains(t, stdout, "enum Status")

	// Should NOT include scalars even though they're used by User
	assert.NotContains(t, stdout, "scalar ID")
	assert.NotContains(t, stdout, "scalar String")
}

func TestTypes_UsedByFilter_NonExistentType(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--used-by", "NonExistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestTypes_UsedByFilter_DidYouMean(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--used-by", "Usr"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "User")
}

func TestTypes_UsedByFilter_JSON(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
			friend: User
		}

		type Post {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "json", "--used-by", "User"})
	require.NoError(t, err)

	var types []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &types)
	require.NoError(t, err)

	typeNames := make(map[string]bool)
	for _, typ := range types {
		typeNames[typ.Name] = true
	}

	// Should include ID and User (self-reference)
	assert.True(t, typeNames["ID"], "Expected ID to be used by User")
	assert.True(t, typeNames["User"], "Expected User to be used by User (self-reference)")

	// Should NOT include Post
	assert.False(t, typeNames["Post"], "Post should not be used by User")
}

func TestTypes_HasFieldFilter(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
			name: String!
			email: String!
		}

		type Post {
			id: ID!
			title: String!
		}

		type Comment {
			text: String!
			author: User!
		}

		type Query {
			user(id: ID!): User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--has-field", "id"})
	require.NoError(t, err)

	// Should include types that have "id" field
	assert.Contains(t, stdout, "type User")
	assert.Contains(t, stdout, "type Post")

	// Should NOT include types that don't have "id" field
	assert.NotContains(t, stdout, "type Comment")
	assert.NotContains(t, stdout, "type Query")
}

func TestTypes_HasFieldFilter_MultipleFields(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
			name: String!
			email: String!
		}

		type Post {
			id: ID!
			name: String!
			title: String!
		}

		type Comment {
			id: ID!
			text: String!
		}

		type Query {
			dummy: String
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--has-field", "id", "--has-field", "name"})
	require.NoError(t, err)

	// Should include types that have BOTH "id" AND "name" fields
	assert.Contains(t, stdout, "type User")
	assert.Contains(t, stdout, "type Post")

	// Should NOT include types that don't have both fields
	assert.NotContains(t, stdout, "type Comment")
	assert.NotContains(t, stdout, "type Query")
}

func TestTypes_HasFieldFilter_JSON(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
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

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "json", "--has-field", "id"})
	require.NoError(t, err)

	var types []struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}

	err = json.Unmarshal([]byte(stdout), &types)
	require.NoError(t, err)

	// Should only have User
	assert.Len(t, types, 1)
	assert.Equal(t, "User", types[0].Name)
}

func TestTypes_HasFieldFilter_NoMatches(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		type User {
			id: ID!
			name: String!
		}

		type Query {
			dummy: String
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "json", "--has-field", "nonexistent"})
	require.NoError(t, err)

	var types []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &types)
	require.NoError(t, err)

	assert.Len(t, types, 0)
}

func TestTypes_HasFieldFilter_CombinedWithImplements(t *testing.T) {
	schemaPath := writeTypesTestSchema(t, `
		interface Node {
			id: ID!
		}

		type User implements Node {
			id: ID!
			name: String!
			email: String!
		}

		type Post implements Node {
			id: ID!
			title: String!
		}

		type Admin implements Node {
			id: ID!
			email: String!
		}

		type Query {
			email: String!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"types", "-s", schemaPath, "-f", "text", "--implements", "Node", "--has-field", "email"})
	require.NoError(t, err)

	// Should include types that implement Node AND have "email" field
	assert.Contains(t, stdout, "type User")
	assert.Contains(t, stdout, "type Admin")

	// Should NOT include types that don't match both filters
	assert.NotContains(t, stdout, "type Post")
	assert.NotContains(t, stdout, "type Query")
}
