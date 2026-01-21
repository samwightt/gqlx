package cmd

type ArgumentInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ArgInfo struct {
	TypeName     string `json:"typeName,omitempty"`
	FieldName    string `json:"fieldName,omitempty"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Description  string `json:"description,omitempty"`
}

type FieldInfo struct {
	TypeName     string         `json:"typeName,omitempty"`
	Name         string         `json:"name"`
	Arguments    []ArgumentInfo `json:"arguments,omitempty"`
	Type         string         `json:"type"`
	DefaultValue string         `json:"defaultValue,omitempty"`
	Description  string         `json:"description,omitempty"`
}

type TypeInfo struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description,omitempty"`
}

type ReferenceInfo struct {
	Location    string `json:"location"`              // e.g., "Query.user" or "Query.users.id"
	Kind        string `json:"kind"`                  // "field" or "argument"
	Type        string `json:"type"`                  // The full type string e.g., "User!" or "[User!]!"
	Description string `json:"description,omitempty"` // Description of the field or argument
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type ValidationError struct {
	Message   string     `json:"message"`
	Locations []Location `json:"locations,omitempty"`
	Rule      string     `json:"rule,omitempty"` // e.g., "FieldsOnCorrectType"
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}
