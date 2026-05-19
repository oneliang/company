package workflow

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// TemplateStep defines a step in the template.
type TemplateStep struct {
	ID              string   `yaml:"id"`
	Role            string   `yaml:"role"`
	Action          string   `yaml:"action"`
	Description     string   `yaml:"description"`
	DependsOn       []string `yaml:"depends_on,omitempty"`
	IsDecisionPoint bool     `yaml:"decision_point,omitempty"`
}

// Template defines a reusable workflow structure.
type Template struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Steps       []TemplateStep `yaml:"steps"`
}

// LoadTemplate loads from YAML file.
func LoadTemplate(path string) (*Template, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// ToWorkflow converts template to an executable workflow.
func (t *Template) ToWorkflow() *Workflow {
	steps := make([]*Step, len(t.Steps))
	for i, ts := range t.Steps {
		steps[i] = &Step{
			ID:              ts.ID,
			Role:            ts.Role,
			Action:          ts.Action,
			Description:     ts.Description,
			Status:          StepPending,
			DependsOn:       ts.DependsOn,
			IsDecisionPoint: ts.IsDecisionPoint,
		}
	}
	return &Workflow{Steps: steps, Status: "pending"}
}