package user

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// Store provides JSONL-based user persistence.
type Store struct {
	dir string
	mu  sync.Mutex
}

// NewStore creates a user store.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Save writes a user to JSONL file.
func (s *Store) Save(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, user.ID+".jsonl")
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

// Get retrieves a user by ID.
func (s *Store) Get(id string) (*User, error) {
	path := filepath.Join(s.dir, id+".jsonl")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var u User
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// List returns all users.
func (s *Store) List() ([]*User, error) {
	files, err := filepath.Glob(filepath.Join(s.dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var users []*User
	for _, f := range files {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			continue
		}
		var u User
		if json.Unmarshal(data, &u) == nil {
			users = append(users, &u)
		}
	}
	return users, nil
}