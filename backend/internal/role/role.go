package role

import "errors"

// Role represents an agent role in the virtual company.
type Role struct {
	ID           string   `yaml:"id" json:"id"`
	Name         string   `yaml:"name" json:"name"`
	Description  string   `yaml:"description" json:"description"`
	SystemPrompt string   `yaml:"system_prompt" json:"system_prompt"`
	ToolsAllowed []string `yaml:"tools_allowed" json:"tools_allowed"`
	Skills       []string `yaml:"skills" json:"skills,omitempty"`
	Temperature  float64  `yaml:"temperature" json:"temperature,omitempty"`
}

// Validate checks if role has required fields.
func (r *Role) Validate() error {
	if r.ID == "" {
		return errors.New("role ID is required")
	}
	if r.Name == "" {
		return errors.New("role name is required")
	}
	return nil
}