package cmd_test

import (
	"encoding/json"
	"testing"

	"github.com/samwightt/gqlx/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgs_SpecificField_Text(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!, includeDeleted: Boolean): User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "Query.user"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "id: ID!")
	assert.Contains(t, stdout, "includeDeleted: Boolean")
}

func TestArgs_SpecificField_JSON(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			users(limit: Int = 10, offset: Int): [User!]!
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "json", "Query.users"})
	require.NoError(t, err)

	var args []struct {
		Name         string `json:"name"`
		Type         string `json:"type"`
		DefaultValue string `json:"defaultValue,omitempty"`
	}

	err = json.Unmarshal([]byte(stdout), &args)
	require.NoError(t, err)

	assert.Len(t, args, 2)

	argMap := make(map[string]struct {
		Type         string
		DefaultValue string
	})
	for _, a := range args {
		argMap[a.Name] = struct {
			Type         string
			DefaultValue string
		}{a.Type, a.DefaultValue}
	}

	assert.Equal(t, "Int", argMap["limit"].Type)
	assert.Equal(t, "10", argMap["limit"].DefaultValue)
	assert.Equal(t, "Int", argMap["offset"].Type)
	assert.Equal(t, "", argMap["offset"].DefaultValue)
}

func TestArgs_SpecificField_Pretty(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "pretty", "Query.user"})
	require.NoError(t, err)

	// Pretty format should have table elements
	assert.Contains(t, stdout, "─")
	assert.Contains(t, stdout, "│")
	assert.Contains(t, stdout, "argument")
	assert.Contains(t, stdout, "type")
	assert.Contains(t, stdout, "id")
	assert.Contains(t, stdout, "ID!")
}

func TestArgs_AllFields_Text(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
			posts(limit: Int): [Post!]!
		}

		type User {
			id: ID!
			friends(first: Int): [User!]!
		}

		type Post {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should include args with Type.field.arg format
	assert.Contains(t, stdout, "Query.user.id: ID!")
	assert.Contains(t, stdout, "Query.posts.limit: Int")
	assert.Contains(t, stdout, "User.friends.first: Int")
}

func TestArgs_AllFields_JSON(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "json"})
	require.NoError(t, err)

	var args []struct {
		TypeName  string `json:"typeName"`
		FieldName string `json:"fieldName"`
		Name      string `json:"name"`
		Type      string `json:"type"`
	}

	err = json.Unmarshal([]byte(stdout), &args)
	require.NoError(t, err)

	// Find the Query.user.id arg
	var found bool
	for _, a := range args {
		if a.TypeName == "Query" && a.FieldName == "user" && a.Name == "id" {
			assert.Equal(t, "ID!", a.Type)
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find Query.user.id argument")
}

func TestArgs_DeprecatedFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(
				id: ID!,
				legacyId: Int @deprecated(reason: "Use id instead")
			): User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "--deprecated", "Query.user"})
	require.NoError(t, err)

	// Should only include deprecated args
	assert.Contains(t, stdout, "legacyId: Int")

	// Should NOT include non-deprecated args
	assert.NotContains(t, stdout, "id: ID!")
}

func TestArgs_DeprecatedFilter_AllFields(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!, oldId: Int @deprecated): User
			post(id: ID!): Post
		}

		type User {
			id: ID!
			friends(first: Int, legacyLimit: Int @deprecated): [User!]!
		}

		type Post {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "--deprecated"})
	require.NoError(t, err)

	// Should include deprecated args with full path
	assert.Contains(t, stdout, "Query.user.oldId: Int")
	assert.Contains(t, stdout, "User.friends.legacyLimit: Int")

	// Should NOT include non-deprecated args
	assert.NotContains(t, stdout, "Query.user.id")
	assert.NotContains(t, stdout, "Query.post.id")
	assert.NotContains(t, stdout, "User.friends.first")
}

func TestArgs_NonExistentType(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "NonExistent.field"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestArgs_NonExistentField(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "Query.nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestArgs_InvalidFieldFormat(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "justafieldname"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Type.field")
}

func TestArgs_FieldWithNoArgs(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "json", "Query.user"})
	require.NoError(t, err)

	var args []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &args)
	require.NoError(t, err)

	assert.Len(t, args, 0)
}

func TestArgs_DidYouMean_Type(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "Queri.user"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "Query")
}

func TestArgs_DidYouMean_Field(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "Query.usr"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "user")
}

func TestArgs_WithDescription(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			"Fetch a user"
			user(
				"The user's unique identifier"
				id: ID!
			): User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "Query.user"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "id: ID!")
	assert.Contains(t, stdout, "# The user's unique identifier")
}

func TestArgs_TypeFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!, name: String): User
			users(limit: Int, offset: Int): [User!]!
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "--type", "Int", "Query.users"})
	require.NoError(t, err)

	// Should include Int arguments
	assert.Contains(t, stdout, "limit: Int")
	assert.Contains(t, stdout, "offset: Int")
}

func TestArgs_TypeFilter_AllFields(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
			search(query: String!, limit: Int): [User!]!
		}

		type User {
			id: ID!
			friends(first: Int): [User!]!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "--type", "Int"})
	require.NoError(t, err)

	// Should include Int arguments with full path
	assert.Contains(t, stdout, "Query.search.limit: Int")
	assert.Contains(t, stdout, "User.friends.first: Int")

	// Should NOT include non-Int arguments
	assert.NotContains(t, stdout, "Query.user.id")
	assert.NotContains(t, stdout, "Query.search.query")
}

func TestArgs_RequiredFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!, includeDeleted: Boolean): User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "--required", "Query.user"})
	require.NoError(t, err)

	// Should include required arguments
	assert.Contains(t, stdout, "id: ID!")

	// Should NOT include nullable arguments
	assert.NotContains(t, stdout, "includeDeleted")
}

func TestArgs_NullableFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!, includeDeleted: Boolean, limit: Int): User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "--nullable", "Query.user"})
	require.NoError(t, err)

	// Should include nullable arguments
	assert.Contains(t, stdout, "includeDeleted: Boolean")
	assert.Contains(t, stdout, "limit: Int")

	// Should NOT include required arguments
	assert.NotContains(t, stdout, "id: ID!")
}

func TestArgs_RequiredFilter_AllFields(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
			search(query: String!, limit: Int): [User!]!
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "--required"})
	require.NoError(t, err)

	// Should include required arguments
	assert.Contains(t, stdout, "Query.user.id: ID!")
	assert.Contains(t, stdout, "Query.search.query: String!")

	// Should NOT include nullable arguments
	assert.NotContains(t, stdout, "Query.search.limit")
}

func TestArgs_RequiredAndNullable_MutuallyExclusive(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "--required", "--nullable", "Query.user"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used together")
}

func TestArgs_CombinedFilters(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			search(
				query: String!,
				limit: Int,
				offset: Int!,
				filter: String
			): [User!]!
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"args", "-s", schemaPath, "-f", "text", "--type", "Int", "--required", "Query.search"})
	require.NoError(t, err)

	// Should include only required Int arguments
	assert.Contains(t, stdout, "offset: Int!")

	// Should NOT include nullable Int or non-Int arguments
	assert.NotContains(t, stdout, "limit:")
	assert.NotContains(t, stdout, "query:")
	assert.NotContains(t, stdout, "filter:")
}
