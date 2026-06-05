package workflow

// StepStatus represents step execution state.
type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
	StepBlocked   StepStatus = "blocked"
)

// Step represents a workflow step.
type Step struct {
	ID              string     `json:"id"`
	Role            string     `json:"role"`
	Action          string     `json:"action"`
	Description     string     `json:"description"`
	Status          StepStatus `json:"status"`
	DependsOn       []string   `json:"depends_on,omitempty"`
	IsDecisionPoint bool       `json:"is_decision_point"`
	Output          string     `json:"output,omitempty"`
	Error           string     `json:"error,omitempty"` // Error message if step failed
}