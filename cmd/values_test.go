package cmd_test

import (
	"encoding/json"
	"testing"

	"github.com/samwightt/gqlx/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValues_SpecificEnum_Text(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			"Currently active"
			ACTIVE
			"Not active"
			INACTIVE
			PENDING
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text", "Status"})
	require.NoError(t, err)

	assert.Contains(t, stdout, "ACTIVE")
	assert.Contains(t, stdout, "INACTIVE")
	assert.Contains(t, stdout, "PENDING")
	assert.Contains(t, stdout, "# Currently active")
	assert.Contains(t, stdout, "# Not active")
}

func TestValues_SpecificEnum_JSON(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			"Currently active"
			ACTIVE
			INACTIVE
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "json", "Status"})
	require.NoError(t, err)

	var values []struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}

	err = json.Unmarshal([]byte(stdout), &values)
	require.NoError(t, err)

	assert.Len(t, values, 2)

	valueMap := make(map[string]string)
	for _, v := range values {
		valueMap[v.Name] = v.Description
	}

	assert.Equal(t, "Currently active", valueMap["ACTIVE"])
	assert.Equal(t, "", valueMap["INACTIVE"])
}

func TestValues_SpecificEnum_Pretty(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			ACTIVE
			INACTIVE
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "pretty", "Status"})
	require.NoError(t, err)

	// Pretty format should have table elements
	assert.Contains(t, stdout, "─")
	assert.Contains(t, stdout, "│")
	assert.Contains(t, stdout, "value")
	assert.Contains(t, stdout, "description")
	assert.Contains(t, stdout, "ACTIVE")
	assert.Contains(t, stdout, "INACTIVE")
}

func TestValues_AllEnums_Text(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			ACTIVE
			INACTIVE
		}

		enum Role {
			ADMIN
			USER
		}

		type Query {
			status: Status
			role: Role
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text"})
	require.NoError(t, err)

	// Should include values with enum prefix
	assert.Contains(t, stdout, "Status.ACTIVE")
	assert.Contains(t, stdout, "Status.INACTIVE")
	assert.Contains(t, stdout, "Role.ADMIN")
	assert.Contains(t, stdout, "Role.USER")
}

func TestValues_AllEnums_JSON(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			ACTIVE
		}

		enum Role {
			ADMIN
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "json"})
	require.NoError(t, err)

	var values []struct {
		EnumName string `json:"enumName"`
		Name     string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &values)
	require.NoError(t, err)

	// Find our custom enums (ignoring built-in ones)
	var foundStatusActive, foundRoleAdmin bool
	for _, v := range values {
		if v.EnumName == "Status" && v.Name == "ACTIVE" {
			foundStatusActive = true
		}
		if v.EnumName == "Role" && v.Name == "ADMIN" {
			foundRoleAdmin = true
		}
	}
	assert.True(t, foundStatusActive, "Expected to find Status.ACTIVE")
	assert.True(t, foundRoleAdmin, "Expected to find Role.ADMIN")
}

func TestValues_DeprecatedFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			ACTIVE
			INACTIVE
			OLD_STATUS @deprecated(reason: "Use INACTIVE instead")
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text", "--deprecated", "Status"})
	require.NoError(t, err)

	// Should only include deprecated values
	assert.Contains(t, stdout, "OLD_STATUS")

	// Should NOT include non-deprecated values
	assert.NotContains(t, stdout, "ACTIVE")
	assert.NotContains(t, stdout, "INACTIVE")
}

func TestValues_DeprecatedFilter_AllEnums(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			ACTIVE
			OLD_STATUS @deprecated
		}

		enum Role {
			ADMIN
			SUPERUSER @deprecated(reason: "Use ADMIN")
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text", "--deprecated"})
	require.NoError(t, err)

	// Should include deprecated values with enum prefix
	assert.Contains(t, stdout, "Status.OLD_STATUS")
	assert.Contains(t, stdout, "Role.SUPERUSER")

	// Should NOT include non-deprecated values
	assert.NotContains(t, stdout, "Status.ACTIVE")
	assert.NotContains(t, stdout, "Role.ADMIN")
}

func TestValues_NonExistentEnum(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			ACTIVE
		}

		type Query {
			status: Status
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text", "NonExistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestValues_DidYouMean(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			ACTIVE
		}

		type Query {
			status: Status
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text", "Staus"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
	assert.Contains(t, err.Error(), "Status")
}

func TestValues_NotAnEnum(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		type User {
			id: ID!
		}

		type Query {
			user: User
		}
	`)

	_, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text", "User"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an enum")
}

func TestValues_DeprecatedFilter_NoMatches(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			ACTIVE
			INACTIVE
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "json", "--deprecated", "Status"})
	require.NoError(t, err)

	var values []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &values)
	require.NoError(t, err)

	assert.Len(t, values, 0)
}

func TestValues_HasDescriptionFilter(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			"Currently active"
			ACTIVE
			INACTIVE
			"Waiting for review"
			PENDING
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text", "--has-description", "Status"})
	require.NoError(t, err)

	// Should include values with descriptions
	assert.Contains(t, stdout, "ACTIVE")
	assert.Contains(t, stdout, "PENDING")

	// Should NOT include values without descriptions
	assert.NotContains(t, stdout, "INACTIVE")
}

func TestValues_HasDescriptionFilter_AllEnums(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			"Currently active"
			ACTIVE
			INACTIVE
		}

		enum Role {
			ADMIN
			"Regular user"
			USER
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text", "--has-description"})
	require.NoError(t, err)

	// Should include values with descriptions (with enum prefix)
	assert.Contains(t, stdout, "Status.ACTIVE")
	assert.Contains(t, stdout, "Role.USER")

	// Should NOT include values without descriptions
	assert.NotContains(t, stdout, "Status.INACTIVE")
	assert.NotContains(t, stdout, "Role.ADMIN")
}

func TestValues_HasDescriptionFilter_JSON(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			"Currently active"
			ACTIVE
			INACTIVE
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "json", "--has-description", "Status"})
	require.NoError(t, err)

	var values []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	err = json.Unmarshal([]byte(stdout), &values)
	require.NoError(t, err)

	// Should only have ACTIVE
	assert.Len(t, values, 1)
	assert.Equal(t, "ACTIVE", values[0].Name)
	assert.Equal(t, "Currently active", values[0].Description)
}

func TestValues_HasDescriptionFilter_NoMatches(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			ACTIVE
			INACTIVE
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "json", "--has-description", "Status"})
	require.NoError(t, err)

	var values []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal([]byte(stdout), &values)
	require.NoError(t, err)

	assert.Len(t, values, 0)
}

func TestValues_HasDescriptionFilter_CombinedWithDeprecated(t *testing.T) {
	schemaPath := writeTestSchema(t, `
		enum Status {
			"Currently active"
			ACTIVE
			INACTIVE
			"Old status, do not use"
			OLD_STATUS @deprecated(reason: "Use ACTIVE instead")
			LEGACY @deprecated
		}

		type Query {
			status: Status
		}
	`)

	stdout, _, err := cmd.ExecuteWithArgs([]string{"values", "-s", schemaPath, "-f", "text", "--has-description", "--deprecated", "Status"})
	require.NoError(t, err)

	// Should include deprecated values with descriptions
	assert.Contains(t, stdout, "OLD_STATUS")

	// Should NOT include non-deprecated values or deprecated values without descriptions
	assert.NotContains(t, stdout, "ACTIVE")
	assert.NotContains(t, stdout, "INACTIVE")
	assert.NotContains(t, stdout, "LEGACY")
}
