package role

import (
	"errors"
	"sync"
)

// Pool manages available roles.
type Pool struct {
	roles map[string]*Role
	mu    sync.RWMutex
}

// NewPool creates an empty role pool.
func NewPool() *Pool {
	return &Pool{roles: make(map[string]*Role)}
}

// Add adds a role to the pool.
func (p *Pool) Add(role *Role) error {
	if err := role.Validate(); err != nil {
		return err
	}
	p.mu.Lock()
	p.roles[role.ID] = role
	p.mu.Unlock()
	return nil
}

// Get retrieves a role by ID.
func (p *Pool) Get(id string) (*Role, error) {
	p.mu.RLock()
	role, ok := p.roles[id]
	p.mu.RUnlock()
	if !ok {
		return nil, errors.New("role not found: " + id)
	}
	return role, nil
}

// List returns all roles.
func (p *Pool) List() []*Role {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]*Role, 0, len(p.roles))
	for _, r := range p.roles {
		result = append(result, r)
	}
	return result
}