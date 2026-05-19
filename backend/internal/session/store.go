package session

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// Store provides JSONL-based session persistence with company isolation.
type Store struct {
	baseDir string // "data/companys"
	mu      sync.Mutex
}

// NewStore creates a session store.
// baseDir should be "data/companys"
func NewStore(baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &Store{baseDir: baseDir}, nil
}

// sessionsDir returns company-specific session directory: companys/<id>/sessions.
func (s *Store) sessionsDir(companyID string) string {
	return filepath.Join(s.baseDir, companyID, "sessions")
}

// Save writes a session to companys/<id>/sessions/<sid>.jsonl.
func (s *Store) Save(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.sessionsDir(session.CompanyID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, session.ID+".jsonl")
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

// Get retrieves a session by company ID and session ID.
func (s *Store) Get(companyID, sessionID string) (*Session, error) {
	path := filepath.Join(s.sessionsDir(companyID), sessionID+".jsonl")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// List returns all sessions for a company.
func (s *Store) List(companyID string) ([]*Session, error) {
	dir := s.sessionsDir(companyID)
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var sessions []*Session
	for _, f := range files {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			continue
		}
		var sess Session
		if json.Unmarshal(data, &sess) == nil {
			sessions = append(sessions, &sess)
		}
	}
	return sessions, nil
}

// Delete removes a session.
func (s *Store) Delete(companyID, sessionID string) error {
	path := filepath.Join(s.sessionsDir(companyID), sessionID+".jsonl")
	return os.Remove(path)
}