package ceo

import (
	"sync"
	"time"

	"github.com/oneliang/company/internal/common"
)

// DecisionType for CEO decisions.
type DecisionType string

const (
	DecisionApprove  DecisionType = "approve"   // 批准，继续下一个步骤
	DecisionContinue DecisionType = "continue"  // 追加意见，继续执行（保留历史）
	DecisionRestart  DecisionType = "restart"   // 清空历史，重新执行
	DecisionReject   DecisionType = "reject"    // 拒绝，工作流失败
)

// Decision represents a CEO decision at decision points.
type Decision struct {
	ID        string       `json:"id"`
	SessionID string       `json:"session_id"`
	StepID    string       `json:"step_id"`
	Type      DecisionType `json:"type"`
	Content   string       `json:"content"`
	CreatedAt time.Time    `json:"created_at"`
}

// NewDecision creates a decision.
func NewDecision(sessionID, stepID string, typ DecisionType, content string) *Decision {
	return &Decision{
		ID:        common.ShortID12(),
		SessionID: sessionID,
		StepID:    stepID,
		Type:      typ,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// Store manages CEO decisions.
type Store struct {
	decisions map[string][]*Decision
	mu        sync.Mutex
}

// NewDecisionStore creates a decision store.
func NewDecisionStore() *Store {
	return &Store{decisions: make(map[string][]*Decision)}
}

// Add stores a decision.
func (s *Store) Add(d *Decision) {
	s.mu.Lock()
	s.decisions[d.SessionID] = append(s.decisions[d.SessionID], d)
	s.mu.Unlock()
}

// GetBySession retrieves decisions for a session.
func (s *Store) GetBySession(sessionID string) []*Decision {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.decisions[sessionID]
}