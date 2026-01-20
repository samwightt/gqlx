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
