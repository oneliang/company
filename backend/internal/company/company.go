package company

import (
	"time"

	"github.com/oneliang/company/internal/common"
)

// Company represents a virtual company entity.
type Company struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Industry     string    `json:"industry"`      // software, marketing, consulting, etc.
	OwnerID      string    `json:"owner_id"`      // CEO user ID
	WorkspaceDir string    `json:"workspace_dir"` // Company workspace directory name
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// New creates a new company.
func New(name, industry, ownerID string) *Company {
	now := time.Now()
	return &Company{
		ID:           common.ShortID8(),
		Name:         name,
		Industry:     industry,
		OwnerID:      ownerID,
		WorkspaceDir: "workspace", // Default workspace directory name
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetDescription updates company description.
func (c *Company) SetDescription(desc string) {
	c.Description = desc
	c.UpdatedAt = time.Now()
}