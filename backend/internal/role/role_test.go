package role

import "testing"

func TestRoleCreation(t *testing.T) {
	role := &Role{
		ID:           "dev",
		Name:         "Developer",
		Description:  "Software developer role",
		SystemPrompt: "You are a developer",
		ToolsAllowed: []string{"file_read", "file_write"},
	}
	if role.ID != "dev" {
		t.Errorf("expected dev, got %s", role.ID)
	}
	if len(role.ToolsAllowed) != 2 {
		t.Errorf("expected 2 tools, got %d", len(role.ToolsAllowed))
	}
}

func TestRoleValidation(t *testing.T) {
	role := &Role{ID: ""}
	if err := role.Validate(); err == nil {
		t.Error("expected error for empty ID")
	}

	role2 := &Role{ID: "pm", Name: ""}
	if err := role2.Validate(); err == nil {
		t.Error("expected error for empty Name")
	}

	role3 := &Role{ID: "pm", Name: "PM"}
	if err := role3.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}