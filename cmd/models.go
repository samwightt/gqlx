package cmd

type ArgumentInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type FieldInfo struct {
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
