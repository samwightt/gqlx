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

func setupRefsTestSchema(t *testing.T, schema string) string {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.graphql")
	err := os.WriteFile(schemaPath, []byte(schema), 0644)
	require.NoError(t, err)
	return schemaPath
}

func TestReferences_FieldReturns(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
			name: String!
		}

		type Query {
			user: User
			users: [User!]!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should include fields that return User
	assert.Contains(t, stdout, "Query.user: User")
	assert.Contains(t, stdout, "Query.users: [User!]!")
}

func TestReferences_ArgumentType(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user(id: ID!): User
			search(userId: ID!, limit: Int): [User!]!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "ID", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should include arguments of type ID
	assert.Contains(t, stdout, "Query.user.id: ID!")
	assert.Contains(t, stdout, "Query.search.userId: ID!")

	// Should include fields that return ID
	assert.Contains(t, stdout, "User.id: ID!")
}

func TestReferences_ListTypeReferences(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
			friends: [User!]!
		}

		type Query {
			users: [User!]!
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should match User even when wrapped in list/non-null
	assert.Contains(t, stdout, "User.friends: [User!]!")
	assert.Contains(t, stdout, "Query.users: [User!]!")
	assert.Contains(t, stdout, "Query.user: User")
}

func TestReferences_KindFilterField(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user(id: ID!): User
			users: [User!]!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "-f", "text", "--kind", "field"})
	require.NoError(t, err)

	// Should include fields
	assert.Contains(t, stdout, "Query.user: User")
	assert.Contains(t, stdout, "Query.users: [User!]!")

	// Should NOT include ID arguments
	assert.NotContains(t, stdout, ".id:")
}

func TestReferences_KindFilterArgument(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user(id: ID!): User
			search(userId: ID!): [User!]!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "ID", "-s", schemaPath, "-f", "text", "--kind", "argument"})
	require.NoError(t, err)

	// Should include arguments
	assert.Contains(t, stdout, "Query.user.id: ID!")
	assert.Contains(t, stdout, "Query.search.userId: ID!")

	// Should NOT include field returns
	assert.NotContains(t, stdout, "User.id: ID!")
}

func TestReferences_InTypeFilter(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
			name: String!
		}

		type Post {
			id: ID!
			author: User!
		}

		type Query {
			user: User
			post: Post
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "-f", "text", "--in", "Query"})
	require.NoError(t, err)

	// Should include references from Query
	assert.Contains(t, stdout, "Query.user: User")

	// Should NOT include references from Post
	assert.NotContains(t, stdout, "Post.author")
}

func TestReferences_NonExistentTypeError(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"references", "NonExistent", "-s", schemaPath, "-f", "text"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestReferences_DidYouMeanSuggestion(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"references", "Usr", "-s", schemaPath, "-f", "text"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "User")
}

func TestReferences_NoReferencesFound(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Orphan {
			name: String!
		}

		type Query {
			user: User
		}
	`)

	_, stderr, err := cmd.ExecuteWithArgs([]string{"references", "Orphan", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should print "No references found" to stderr
	assert.Contains(t, stderr, "No references found")
}

func TestReferences_NoReferencesFound_JSON(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Orphan {
			name: String!
		}

		type Query {
			user: User
		}
	`)

	stdout, stderr, err := cmd.ExecuteWithArgs([]string{"references", "Orphan", "-s", schemaPath, "-f", "json"})
	require.NoError(t, err)

	// Should print "No references found" to stderr
	assert.Contains(t, stderr, "No references found")
	// Stdout should have null for JSON (nil slice marshals to null)
	assert.Equal(t, "null\n", stdout)
}

func TestReferences_JSONFormat(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		"A user"
		type User {
			id: ID!
		}

		type Query {
			"Get a user by ID"
			user(id: ID!): User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "-f", "json"})
	require.NoError(t, err)

	var refs []struct {
		Location    string `json:"location"`
		Kind        string `json:"kind"`
		Type        string `json:"type"`
		Description string `json:"description,omitempty"`
	}

	err = json.Unmarshal([]byte(stdout), &refs)
	require.NoError(t, err)

	// Find Query.user reference
	var foundQueryUser bool
	for _, ref := range refs {
		if ref.Location == "Query.user" {
			assert.Equal(t, "field", ref.Kind)
			assert.Equal(t, "User", ref.Type)
			assert.Equal(t, "Get a user by ID", ref.Description)
			foundQueryUser = true
		}
	}
	assert.True(t, foundQueryUser, "Expected to find Query.user reference")
}

func TestReferences_PrettyFormat(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "-f", "pretty"})
	require.NoError(t, err)

	// Pretty format should have table elements
	assert.Contains(t, stdout, "location")
	assert.Contains(t, stdout, "kind")
	assert.Contains(t, stdout, "type")
	assert.Contains(t, stdout, "Query.user")
	assert.Contains(t, stdout, "field")
}

func TestReferences_InvalidKindFilter(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "--kind", "invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--kind must be 'field' or 'argument'")
}

func TestReferences_InTypeFilterNonExistent(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "--in", "NonExistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestReferences_InTypeFilterDidYouMean(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "--in", "Qery"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "Query")
}

func TestReferences_InputTypeReferences(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		input CreateUserInput {
			name: String!
		}

		type Mutation {
			createUser(input: CreateUserInput!): User!
		}

		type Query {
			dummy: String
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "CreateUserInput", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should find argument reference
	assert.Contains(t, stdout, "Mutation.createUser.input: CreateUserInput!")
}

func TestReferences_CombinedFilters(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Post {
			id: ID!
		}

		type Query {
			user(id: ID!): User
			post(id: ID!): Post
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "ID", "-s", schemaPath, "-f", "text", "--kind", "argument", "--in", "Query"})
	require.NoError(t, err)

	// Should include Query arguments
	assert.Contains(t, stdout, "Query.user.id: ID!")
	assert.Contains(t, stdout, "Query.post.id: ID!")

	// Should NOT include field returns
	assert.NotContains(t, stdout, "User.id")
	assert.NotContains(t, stdout, "Post.id")
}

func TestReferences_NestedListTypes(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			userMatrix: [[User!]!]!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "User", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should match User even in nested list
	assert.Contains(t, stdout, "Query.userMatrix: [[User!]!]!")
}

func TestReferences_InterfaceTypeReferences(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		interface Node {
			id: ID!
		}

		type User implements Node {
			id: ID!
			name: String!
		}

		type Query {
			node(id: ID!): Node
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "Node", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should find the interface reference
	assert.Contains(t, stdout, "Query.node: Node")
}

func TestReferences_EnumTypeReferences(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		enum Status {
			ACTIVE
			INACTIVE
		}

		type User {
			id: ID!
			status: Status!
		}

		type Query {
			usersByStatus(status: Status!): [User!]!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"references", "Status", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should find field and argument references
	assert.Contains(t, stdout, "User.status: Status!")
	assert.Contains(t, stdout, "Query.usersByStatus.status: Status!")
}

func TestReferences_RequiresTypeArgument(t *testing.T) {
	schemaPath := setupRefsTestSchema(t, `
		type Query {
			dummy: String
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"references", "-s", schemaPath})
	assert.Error(t, err)
}
