package company

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// Store provides JSONL-based company persistence with company-centric directory.
type Store struct {
	baseDir string // "data/companys"
	mu      sync.Mutex
}

// NewStore creates a company store.
// baseDir should be "data/companys"
func NewStore(baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &Store{baseDir: baseDir}, nil
}

// companyDir returns the directory for a specific company.
func (s *Store) companyDir(companyID string) string {
	return filepath.Join(s.baseDir, companyID)
}

// Save writes a company to companys/<id>/company.jsonl.
func (s *Store) Save(company *Company) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.companyDir(company.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "company.jsonl")
	data, err := json.Marshal(company)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

// Get retrieves a company by ID from companys/<id>/company.jsonl.
func (s *Store) Get(id string) (*Company, error) {
	path := filepath.Join(s.companyDir(id), "company.jsonl")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Company
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// List returns all companies by scanning company directories.
func (s *Store) List() ([]*Company, error) {
	dirs, err := ioutil.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}
	var companies []*Company
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		c, err := s.Get(d.Name())
		if err != nil {
			continue
		}
		companies = append(companies, c)
	}
	return companies, nil
}

// Delete removes a company directory entirely.
func (s *Store) Delete(id string) error {
	dir := s.companyDir(id)
	return os.RemoveAll(dir)
}

// ListByOwner returns companies owned by a user.
func (s *Store) ListByOwner(ownerID string) ([]*Company, error) {
	companies, err := s.List()
	if err != nil {
		return nil, err
	}
	var result []*Company
	for _, c := range companies {
		if c.OwnerID == ownerID {
			result = append(result, c)
		}
	}
	return result, nil
}