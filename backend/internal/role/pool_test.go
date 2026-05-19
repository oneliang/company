package role

import "testing"

func TestRolePoolGet(t *testing.T) {
	pool := NewPool()
	pool.Add(&Role{ID: "pm", Name: "PM"})
	pool.Add(&Role{ID: "dev", Name: "Dev"})

	role, err := pool.Get("pm")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if role.Name != "PM" {
		t.Errorf("expected PM, got %s", role.Name)
	}

	_, err = pool.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent role")
	}
}

func TestRolePoolList(t *testing.T) {
	pool := NewPool()
	pool.Add(&Role{ID: "pm", Name: "PM"})
	pool.Add(&Role{ID: "dev", Name: "Dev"})

	roles := pool.List()
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}

func TestRolePoolAddInvalid(t *testing.T) {
	pool := NewPool()
	err := pool.Add(&Role{ID: "", Name: "Test"})
	if err == nil {
		t.Error("expected error for invalid role")
	}
}