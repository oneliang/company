package user

import (
	"time"

	"github.com/oneliang/company/internal/common"
)

// User represents a CEO user who manages multiple companies.
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`     // "ceo"
	Companies []string  `json:"companies"` // company IDs managed
	CreatedAt time.Time `json:"created_at"`
}

// New creates a new user.
func New(name, email string) *User {
	return &User{
		ID:        common.ShortID8(),
		Name:      name,
		Email:     email,
		Role:      "ceo",
		Companies: []string{},
		CreatedAt: time.Now(),
	}
}

// AddCompany adds a company to user's management list.
func (u *User) AddCompany(companyID string) {
	u.Companies = append(u.Companies, companyID)
}

// RemoveCompany removes a company from user's management list.
func (u *User) RemoveCompany(companyID string) {
	for i, id := range u.Companies {
		if id == companyID {
			u.Companies = append(u.Companies[:i], u.Companies[i+1:]...)
			return
		}
	}
}

// HasCompany checks if user manages a specific company.
func (u *User) HasCompany(companyID string) bool {
	for _, id := range u.Companies {
		if id == companyID {
			return true
		}
	}
	return false
}