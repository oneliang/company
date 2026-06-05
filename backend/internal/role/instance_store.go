package role

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// InstanceStore provides JSONL-based instance persistence.
type InstanceStore struct {
	baseDir string
	mu      sync.Mutex
}

// NewInstanceStore creates an instance store.
func NewInstanceStore(baseDir string) (*InstanceStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &InstanceStore{baseDir: baseDir}, nil
}

// instancesDir returns company-specific instance directory.
func (s *InstanceStore) instancesDir(companyID, sessionID string) string {
	return filepath.Join(s.baseDir, companyID, "sessions", sessionID, "instances")
}

// Save writes an instance to storage.
func (s *InstanceStore) Save(companyID string, instance *Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.instancesDir(companyID, instance.SessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, instance.ID+".jsonl")
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

// Get retrieves an instance by ID.
func (s *InstanceStore) Get(companyID, sessionID, instanceID string) (*Instance, error) {
	path := filepath.Join(s.instancesDir(companyID, sessionID), instanceID+".jsonl")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var instance Instance
	if err := json.Unmarshal(data, &instance); err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetByStep retrieves instance by step ID.
func (s *InstanceStore) GetByStep(companyID, sessionID, stepID string) (*Instance, error) {
	dir := s.instancesDir(companyID, sessionID)
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			continue
		}
		var instance Instance
		if json.Unmarshal(data, &instance) == nil && instance.StepID == stepID {
			return &instance, nil
		}
	}
	return nil, nil
}

// ListBySession returns all instances for a session.
func (s *InstanceStore) ListBySession(companyID, sessionID string) ([]*Instance, error) {
	dir := s.instancesDir(companyID, sessionID)
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var instances []*Instance
	for _, f := range files {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			continue
		}
		var instance Instance
		if json.Unmarshal(data, &instance) == nil {
			instances = append(instances, &instance)
		}
	}
	return instances, nil
}

// ListByRole returns all instances for a role template.
func (s *InstanceStore) ListByRole(companyID, roleID string) ([]*Instance, error) {
	// Search across all sessions in company
	companyDir := filepath.Join(s.baseDir, companyID, "sessions")
	sessionDirs, err := filepath.Glob(filepath.Join(companyDir, "*", "instances"))
	if err != nil {
		return nil, err
	}
	var instances []*Instance
	for _, dir := range sessionDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		if err != nil {
			continue
		}
		for _, f := range files {
			data, err := ioutil.ReadFile(f)
			if err != nil {
				continue
			}
			var instance Instance
			if json.Unmarshal(data, &instance) == nil && instance.RoleID == roleID {
				instances = append(instances, &instance)
			}
		}
	}
	return instances, nil
}

// Delete removes an instance file.
func (s *InstanceStore) Delete(companyID, sessionID, instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.instancesDir(companyID, sessionID), instanceID+".jsonl")
	return os.Remove(path)
}

// DeleteByStep removes instance by step ID.
func (s *InstanceStore) DeleteByStep(companyID, sessionID, stepID string) error {
	instance, err := s.GetByStep(companyID, sessionID, stepID)
	if err != nil || instance == nil {
		return err
	}
	return s.Delete(companyID, sessionID, instance.ID)
}

// DeleteBySession removes all instances for a session.
func (s *InstanceStore) DeleteBySession(companyID, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.instancesDir(companyID, sessionID)
	return os.RemoveAll(dir)
}