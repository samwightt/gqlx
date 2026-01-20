package cmd_test

import (
	"encoding/json"
	"testing"

	"github.com/samwightt/gqlx/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaths_DirectPath(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
		}

		type User {
			id: ID!
			name: String!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "User"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "Query.user(...) -> User")
}

func TestPaths_DirectPathNoArgs(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			viewer: User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "User"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "Query.viewer -> User")
}

func TestPaths_MultiplePaths(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
			users: [User!]!
			viewer: Viewer
		}

		type User {
			id: ID!
		}

		type Viewer {
			me: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "User"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "Query.user(...) -> User")
	assert.Contains(t, stdout, "Query.users -> User")
	assert.Contains(t, stdout, "Query.viewer -> Viewer.me -> User")
}

func TestPaths_NestedPath(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			media(id: ID!): Media
		}

		type Media {
			id: ID!
			author: User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "User"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "Query.media(...) -> Media.author -> User")
}

func TestPaths_JSON(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
			viewer: Viewer
		}

		type User {
			id: ID!
		}

		type Viewer {
			me: User
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "json", "User"})
	require.NoError(t, err)

	var paths []struct {
		Path string `json:"path"`
	}

	err = json.Unmarshal([]byte(stdout), &paths)
	require.NoError(t, err)

	assert.Len(t, paths, 2)

	pathStrings := make([]string, len(paths))
	for i, p := range paths {
		pathStrings[i] = p.Path
	}

	assert.Contains(t, pathStrings, "Query.user(...) -> User")
	assert.Contains(t, pathStrings, "Query.viewer -> Viewer.me -> User")
}

func TestPaths_Pretty(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user(id: ID!): User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "pretty", "User"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "─")
	assert.Contains(t, stdout, "│")
	assert.Contains(t, stdout, "path")
	assert.Contains(t, stdout, "Query.user(...) -> User")
}

func TestPaths_NoPathsFound(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			viewer: Viewer
		}

		type Viewer {
			id: ID!
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "json", "User"})
	require.NoError(t, err)

	var paths []struct {
		Path string `json:"path"`
	}

	err = json.Unmarshal([]byte(stdout), &paths)
	require.NoError(t, err)

	assert.Len(t, paths, 0)
}

func TestPaths_NonExistentType(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "NonExistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestPaths_DidYouMean(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "Usr"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "User")
}

func TestPaths_MaxDepth(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			a: TypeA
		}

		type TypeA {
			b: TypeB
		}

		type TypeB {
			c: TypeC
		}

		type TypeC {
			d: TypeD
		}

		type TypeD {
			target: Target
		}

		type Target {
			id: ID!
		}
	`)

	// With default depth of 5, should find the path
	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "Target"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "Target")

	// With depth of 2, should NOT find the path
	stdout, _, err = cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "json", "--max-depth", "2", "Target"})
	require.NoError(t, err)

	var paths []struct {
		Path string `json:"path"`
	}
	err = json.Unmarshal([]byte(stdout), &paths)
	require.NoError(t, err)
	assert.Len(t, paths, 0)
}

func TestPaths_ListTypes(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			users: [User!]!
		}

		type User {
			id: ID!
			friends(first: Int): [User!]!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "User"})
	require.NoError(t, err)

	// Should find User through the list
	assert.Contains(t, stdout, "Query.users -> User")
}

func TestPaths_FieldWithArgsShowsEllipsis(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			search(query: String!, limit: Int): SearchResult
		}

		type SearchResult {
			users(filter: UserFilter): [User!]!
		}

		type User {
			id: ID!
		}

		input UserFilter {
			name: String
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "User"})
	require.NoError(t, err)

	// Both fields have args, so both should show (...)
	assert.Contains(t, stdout, "Query.search(...) -> SearchResult.users(...) -> User")
}

func TestPaths_ShortestFlag(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
			viewer: Viewer
		}

		type Viewer {
			profile: Profile
		}

		type Profile {
			owner: User
		}

		type User {
			id: ID!
		}
	`)

	// Without --shortest, should show all paths
	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "User"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "Query.user -> User")
	assert.Contains(t, stdout, "Query.viewer -> Viewer.profile -> Profile.owner -> User")

	// With --shortest, should only show the shortest path
	stdout, _, err = cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--shortest", "User"})
	require.NoError(t, err)
	assert.Contains(t, stdout, "Query.user -> User")
	assert.NotContains(t, stdout, "Viewer.profile")
}

func TestPaths_ShortestFlag_MultipleSameLength(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
			viewer: User
			admin: Admin
		}

		type Admin {
			profile: User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--shortest", "User"})
	require.NoError(t, err)

	// Should show both shortest paths (depth 1)
	assert.Contains(t, stdout, "Query.user -> User")
	assert.Contains(t, stdout, "Query.viewer -> User")

	// Should NOT show the longer path
	assert.NotContains(t, stdout, "Admin.profile")
}

func TestPaths_FromFlag(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			viewer: Viewer
		}

		type Viewer {
			friends: [User!]!
			profile: Profile
		}

		type Profile {
			owner: User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--from", "Viewer", "User"})
	require.NoError(t, err)

	// Should show paths from Viewer, not Query
	assert.Contains(t, stdout, "Viewer.friends -> User")
	assert.Contains(t, stdout, "Viewer.profile -> Profile.owner -> User")

	// Should NOT contain Query
	assert.NotContains(t, stdout, "Query")
}

func TestPaths_FromFlag_WithShortest(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			viewer: Viewer
		}

		type Viewer {
			user: User
			profile: Profile
		}

		type Profile {
			owner: User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--from", "Viewer", "--shortest", "User"})
	require.NoError(t, err)

	// Should show only shortest path from Viewer
	assert.Contains(t, stdout, "Viewer.user -> User")
	assert.NotContains(t, stdout, "Profile.owner")
}

func TestPaths_FromFlag_InvalidType(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--from", "NonExistent", "User"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestPaths_FromFlag_DidYouMean(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			viewer: Viewer
		}

		type Viewer {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--from", "Viewr", "User"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "Viewer")
}

func TestPaths_ThroughFlag(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
			media(id: ID!): Media
			viewer: Viewer
		}

		type Media {
			author: User
		}

		type Viewer {
			profile: Profile
		}

		type Profile {
			owner: User
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--through", "Media", "User"})
	require.NoError(t, err)

	// Should only show paths through Media
	assert.Contains(t, stdout, "Query.media(...) -> Media.author -> User")

	// Should NOT show direct path or path through Viewer
	assert.NotContains(t, stdout, "Query.user -> User")
	assert.NotContains(t, stdout, "Viewer.profile")
}

func TestPaths_ThroughFlag_WithShortest(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			viewer: Viewer
		}

		type Viewer {
			media: Media
			content: Content
		}

		type Media {
			author: User
		}

		type Content {
			media: Media
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--through", "Media", "--shortest", "User"})
	require.NoError(t, err)

	// Should show shortest path through Media
	assert.Contains(t, stdout, "Query.viewer -> Viewer.media -> Media.author -> User")

	// Should NOT show longer path through Media
	assert.NotContains(t, stdout, "Content.media")
}

func TestPaths_ThroughFlag_InvalidType(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--through", "NonExistent", "User"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestPaths_ThroughFlag_DidYouMean(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			media: Media
		}

		type Media {
			user: User
		}

		type User {
			id: ID!
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "text", "--through", "Mdia", "User"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "Media")
}

func TestPaths_ThroughFlag_NoMatches(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type Query {
			user: User
		}

		type Media {
			id: ID!
		}

		type User {
			id: ID!
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"paths", "-s", schemaPath, "-f", "json", "--through", "Media", "User"})
	require.NoError(t, err)

	var paths []struct {
		Path string `json:"path"`
	}
	err = json.Unmarshal([]byte(stdout), &paths)
	require.NoError(t, err)

	assert.Len(t, paths, 0)
}
