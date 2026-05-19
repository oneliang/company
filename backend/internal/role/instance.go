package role

import (
	"time"

	"github.com/oneliang/company/internal/common"
)

// InstanceStatus represents role instance execution state.
type InstanceStatus string

const (
	InstancePending   InstanceStatus = "pending"
	InstanceRunning   InstanceStatus = "running"
	InstanceCompleted InstanceStatus = "completed"
	InstanceFailed    InstanceStatus = "failed"
)

// Instance represents a runtime instance of a Role.
// Role is the template (class), Instance is the runtime (instance).
type Instance struct {
	ID        string         `json:"id"`
	RoleID    string         `json:"role_id"`
	SessionID string         `json:"session_id"`
	StepID    string         `json:"step_id"`
	Context   string         `json:"context"`   // Context from dependency outputs
	Input     string         `json:"input"`     // Specific task input
	Output    string         `json:"output"`    // Execution output
	Status    InstanceStatus `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// NewInstance creates a new role instance.
func NewInstance(roleID, sessionID, stepID string) *Instance {
	now := time.Now()
	return &Instance{
		ID:        common.ShortID12(),
		RoleID:    roleID,
		SessionID: sessionID,
		StepID:    stepID,
		Status:    InstancePending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetContext sets the context from dependency outputs.
func (i *Instance) SetContext(ctx string) {
	i.Context = ctx
	i.UpdatedAt = time.Now()
}

// SetInput sets the specific task input.
func (i *Instance) SetInput(input string) {
	i.Input = input
	i.UpdatedAt = time.Now()
}

// SetOutput sets the execution output and marks as completed.
func (i *Instance) SetOutput(output string) {
	i.Output = output
	i.Status = InstanceCompleted
	i.UpdatedAt = time.Now()
}

// SetRunning marks instance as running.
func (i *Instance) SetRunning() {
	i.Status = InstanceRunning
	i.UpdatedAt = time.Now()
}

// SetFailed marks instance as failed.
func (i *Instance) SetFailed() {
	i.Status = InstanceFailed
	i.UpdatedAt = time.Now()
}

// GetFullPrompt combines role template prompt with instance context.
func (i *Instance) GetFullPrompt(role *Role) string {
	prompt := role.SystemPrompt
	if i.Context != "" {
		prompt += "\n\n--- Context from Previous Steps ---\n" + i.Context
	}
	if i.Input != "" {
		prompt += "\n\n--- Current Task ---\n" + i.Input
	}
	return prompt
}