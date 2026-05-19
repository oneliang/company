package session

import (
	"time"

	"github.com/oneliang/company/internal/common"
	"github.com/oneliang/company/internal/workflow"
)

// Status represents session state.
type Status string

const (
	StatusDraft     Status = "draft"     // Workflow generated, waiting for CEO approval
	StatusApproved  Status = "approved"  // CEO approved, ready for execution
	StatusPending   Status = "pending"
	StatusPlanning  Status = "planning"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Session represents a company work session.
type Session struct {
	ID           string             `json:"id"`
	CompanyID    string             `json:"company_id"`    // Company association
	Goal         string             `json:"goal"`
	Status       Status             `json:"status"`
	CurrentStep  string             `json:"current_step"`  // 当前执行的步骤 ID（用于前端显示）
	Workflow     *workflow.Workflow `json:"workflow,omitempty"`
	WorkspaceDir string             `json:"workspace_dir"` // Session workspace directory name
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// New creates a new session for a company.
func New(companyID, goal string) *Session {
	now := time.Now()
	id := common.ShortID12()
	return &Session{
		ID:           id,
		CompanyID:    companyID,
		Goal:         goal,
		Status:       StatusPending,
		WorkspaceDir: id, // Session workspace named by session_id
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetStatus updates session status.
func (s *Session) SetStatus(status Status) {
	s.Status = status
	s.UpdatedAt = time.Now()
}