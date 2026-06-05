package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/company/internal/agent"
	"github.com/oneliang/company/internal/ceo"
	"github.com/oneliang/company/internal/company"
	"github.com/oneliang/company/internal/role"
	"github.com/oneliang/company/internal/session"
	"github.com/oneliang/company/internal/task"
	"github.com/oneliang/company/internal/workflow"
)

// Handlers provides REST API handlers.
type Handlers struct {
	companyStore  *company.Store
	sessionStore  *session.Store
	decisionStore *ceo.Store
	instanceStore *role.InstanceStore
	executor      *agent.Executor
	configsDir    string
	dataDir       string
	wsHandler     *WebSocketHandler // WebSocket handler for progress notifications
}

// NewHandlers creates handlers with agent executor.
func NewHandlers(companyStore *company.Store, sessionStore *session.Store, configsDir string, dataDir string, wsHandler *WebSocketHandler) *Handlers {
	// Initialize agent executor with session store
	executor, err := agent.NewExecutor(filepath.Join(configsDir, "config.yaml"), dataDir)
	if err != nil {
		slog.Warn("config not found, agent execution disabled", "error", err)
	}

	// Initialize role instance store
	instanceStore, err := role.NewInstanceStore(dataDir)
	if err != nil {
		slog.Warn("instance store not initialized", "error", err)
	}

	return &Handlers{
		companyStore:  companyStore,
		sessionStore:  sessionStore,
		decisionStore: ceo.NewDecisionStore(),
		instanceStore: instanceStore,
		executor:      executor,
		configsDir:    configsDir,
		dataDir:       dataDir,
		wsHandler:     wsHandler,
	}
}

// NotifyProgress implements task.ProgressNotifier interface.
func (h *Handlers) NotifyProgress(sessionID string, eventType string, stepID string, role string, action string, status string, progress string, requestID string, errMsg string) {
	// Debug: log the notification call
	slog.Info("NotifyProgress called", "session_id", sessionID, "type", eventType, "step_id", stepID, "role", role, "action", action, "status", status, "request_id", requestID)
	if h.wsHandler != nil {
		h.wsHandler.BroadcastProgress(sessionID, ProgressEvent{
			Type:      eventType,
			StepID:    stepID,
			Role:      role,
			Action:    action,
			Status:    status,
			Progress:  progress,
			RequestID: requestID,
			Error:     errMsg,
		})
	}
}

// CreateCompany creates a new company.
func (h *Handlers) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Industry    string `json:"industry"`
		OwnerID     string `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	c := company.New(req.Name, req.Industry, req.OwnerID)
	c.Description = req.Description
	h.companyStore.Save(c)

	json.NewEncoder(w).Encode(c)
}

// ListCompanies returns companies for a CEO with session statistics.
func (h *Handlers) ListCompanies(w http.ResponseWriter, r *http.Request) {
	ownerID := r.URL.Query().Get("owner_id")
	companies, err := h.companyStore.ListByOwner(ownerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Add session statistics for each company
	type CompanyWithStats struct {
		*company.Company
		CompletedCount int `json:"completed_count"`
		PendingCount   int `json:"pending_count"`
	}

	var result []CompanyWithStats
	for _, c := range companies {
		sessions, err := h.sessionStore.List(c.ID)
		if err != nil {
			sessions = []*session.Session{}
		}

		completed := 0
		pending := 0
		for _, s := range sessions {
			if s.Status == session.StatusCompleted {
				completed++
			} else {
				pending++
			}
		}

		result = append(result, CompanyWithStats{
			Company:        c,
			CompletedCount: completed,
			PendingCount:   pending,
		})
	}

	json.NewEncoder(w).Encode(result)
}

// GetCompany retrieves a company by ID.
func (h *Handlers) GetCompany(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	c, err := h.companyStore.Get(vars["id"])
	if err != nil {
		http.Error(w, "company not found", 404)
		return
	}
	json.NewEncoder(w).Encode(c)
}

// DeleteCompany removes a company.
func (h *Handlers) DeleteCompany(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.companyStore.Delete(vars["id"]); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// CreateSession creates a session in company context.
// Fast response: creates session immediately, generates workflow asynchronously via LLM.
func (h *Handlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]

	var req struct {
		Goal string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get company to determine industry
	c, err := h.companyStore.Get(companyID)
	if err != nil {
		http.Error(w, "company not found", 404)
		return
	}

	// Load company-specific role pool
	pool, err := role.LoadCompanyRoles(h.configsDir, companyID, c.Industry)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Load workflow template (for fallback)
	template, err := workflow.LoadTemplate(filepath.Join(h.configsDir, "workflow_template.yaml"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Create session with planning status (workflow generating)
	sess := session.New(companyID, req.Goal)
	sess.SetStatus(session.StatusPlanning) // LLM generating workflow

	// Save session immediately for fast response
	h.sessionStore.Save(sess)

	// Initialize session workspace
	sessionDir := filepath.Join(h.dataDir, companyID, "sessions", sess.ID)
	outputsDir := filepath.Join(sessionDir, "outputs")
	workspaceDir := filepath.Join(sessionDir, "workspace")
	if err := os.MkdirAll(outputsDir, 0755); err != nil {
		slog.Warn("failed to create session outputs directory", "error", err)
	}
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		slog.Warn("failed to create session workspace directory", "error", err)
	}

	// Write goal.md
	goalMd := "# Task Goal\n\n" + req.Goal + "\n"
	goalPath := filepath.Join(sessionDir, "goal.md")
	if err := os.WriteFile(goalPath, []byte(goalMd), 0644); err != nil {
		slog.Warn("failed to write goal.md", "error", err)
	}

	slog.Info("Session created (planning)", "session_id", sess.ID, "company_id", companyID)

	// Fast response: return immediately
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sess.ID,
		"company_id": companyID,
		"status":     sess.Status,
		"message":    "workflow generating, please wait...",
	})

	// Async: generate workflow via LLM
	go func() {
		ctx := context.Background()
		engine := task.NewEngine(pool, template, h.executor, h.instanceStore, h.sessionStore, companyID, h.dataDir)

		// Call LLM to decompose task
		wf, err := engine.DecomposeTask(ctx, sess)
		if err != nil {
			slog.Error("DecomposeTask failed", "error", err)
			// Fallback: keep planning status, frontend will handle
			return
		}

		// Update session status to draft (ready for approval)
		sess.SetStatus(session.StatusDraft)
		h.sessionStore.Save(sess)

		slog.Info("Workflow generated", "session_id", sess.ID, "steps", len(wf.Steps))

		// Notify frontend via WebSocket
		if h.wsHandler != nil {
			h.wsHandler.BroadcastProgress(sess.ID, ProgressEvent{
				Type:   "workflow_generated",
				Status: "draft",
			})
		}
	}()
}

// ListSessions returns sessions for a company.
func (h *Handlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessions, err := h.sessionStore.List(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(sessions)
}

// GetSession retrieves session status.
func (h *Handlers) GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sess, err := h.sessionStore.Get(vars["id"], vars["sid"])
	if err != nil {
		http.Error(w, "session not found", 404)
		return
	}
	json.NewEncoder(w).Encode(sess)
}

// GetWorkflow returns workflow details.
func (h *Handlers) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sess, err := h.sessionStore.Get(vars["id"], vars["sid"])
	if err != nil {
		http.Error(w, "session not found", 404)
		return
	}
	json.NewEncoder(w).Encode(sess.Workflow)
}

// ExecuteStep executes a workflow step using the role's agent.
func (h *Handlers) ExecuteStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]
	stepID := vars["stepId"]

	sess, err := h.sessionStore.Get(companyID, sessionID)
	if err != nil {
		http.Error(w, "session not found", 404)
		return
	}

	// Get company for role pool
	c, err := h.companyStore.Get(companyID)
	if err != nil {
		http.Error(w, "company not found", 404)
		return
	}

	// Load role pool
	pool, err := role.LoadCompanyRoles(h.configsDir, companyID, c.Industry)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Create engine with executor
	engine := task.NewEngine(pool, nil, h.executor, h.instanceStore, h.sessionStore, companyID, h.dataDir)

	// Generate RequestID for tracing
	requestID := uuid.New().String()

	// Execute step
	ctx := context.Background()
	if err := engine.ExecuteStep(ctx, sess, stepID, requestID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Save updated session
	h.sessionStore.Save(sess)

	json.NewEncoder(w).Encode(sess.Workflow)
}

// SubmitDecision handles CEO decision.
// Supports: approve (批准), continue (追加意见继续), restart (重新执行)
func (h *Handlers) SubmitDecision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]

	var req struct {
		StepID  string `json:"step_id"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	sess, err := h.sessionStore.Get(companyID, sessionID)
	if err != nil {
		http.Error(w, "session not found", 404)
		return
	}

	// 记录决策
	d := ceo.NewDecision(sessionID, req.StepID, ceo.DecisionType(req.Type), req.Content)
	h.decisionStore.Add(d)

	// 获取公司和角色池
	c, err := h.companyStore.Get(companyID)
	if err != nil {
		http.Error(w, "company not found", 404)
		return
	}

	pool, err := role.LoadCompanyRoles(h.configsDir, companyID, c.Industry)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// 创建 Engine 处理决策
	template, _ := workflow.LoadTemplate(filepath.Join(h.configsDir, "workflow_template.yaml"))
	engine := task.NewEngine(pool, template, h.executor, h.instanceStore, h.sessionStore, companyID, h.dataDir)

	// 处理决策
	ctx := context.Background()
	if err := engine.HandleDecision(ctx, sess, req.StepID, req.Type, req.Content); err != nil {
		slog.Error("Decision handling failed", "error", err)
		http.Error(w, err.Error(), 500)
		return
	}

	// 保存 session
	h.sessionStore.Save(sess)

	// 如果是批准，尝试继续执行 workflow
	if req.Type == "approve" {
		sess.SetStatus(session.StatusRunning)
		engine.SetNotifier(h)
		go func() {
			if err := engine.RunWorkflow(ctx, sess); err != nil {
				slog.Error("Workflow continuation failed", "error", err)
			}
			h.sessionStore.Save(sess)
		}()
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"step_id":    req.StepID,
		"decision":   req.Type,
		"session_id": sessionID,
	})
}

// RestartStep restarts a specific step and clears all downstream dependent steps.
// POST /api/companies/{id}/sessions/{sid}/steps/{stepId}/restart
func (h *Handlers) RestartStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]
	stepID := vars["stepId"]

	var req struct {
		CEOOpinion string `json:"ceo_opinion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req.CEOOpinion = ""
	}

	sess, err := h.sessionStore.Get(companyID, sessionID)
	if err != nil {
		http.Error(w, "session not found", 404)
		return
	}

	if sess.Workflow == nil {
		http.Error(w, "workflow not found", 400)
		return
	}

	// Find the step
	step := sess.Workflow.GetStep(stepID)
	if step == nil {
		http.Error(w, "step not found", 404)
		return
	}

	// Clear downstream steps recursively
	clearedSteps := sess.Workflow.ClearDownstreamSteps(stepID)

	// Reset the target step
	step.Status = workflow.StepPending
	step.Output = ""

	// Delete old instances for restarted step and cleared steps (force new creation)
	if h.instanceStore != nil {
		h.instanceStore.DeleteByStep(companyID, sessionID, stepID)
		for _, clearedStepID := range clearedSteps {
			h.instanceStore.DeleteByStep(companyID, sessionID, clearedStepID)
		}
		slog.Info("Deleted old instances for restart", "step", stepID, "cleared", clearedSteps)
	}

	// Save CEO opinion if provided
	if req.CEOOpinion != "" {
		d := ceo.NewDecision(sessionID, stepID, ceo.DecisionRestart, req.CEOOpinion)
		h.decisionStore.Add(d)
	}

	// Save session
	h.sessionStore.Save(sess)

	// Notify frontend via WebSocket
	if h.wsHandler != nil {
			h.wsHandler.Broadcast(sessionID, map[string]interface{}{
				"type":      "steps_cleared",
				"step_ids":  clearedSteps,
				"reason":    "restart_step",
				"restarted": stepID,
			})
	}

	slog.Info("Step restarted", "step_id", stepID, "cleared", clearedSteps)

	// Get company and role pool to re-execute
	c, err := h.companyStore.Get(companyID)
	if err != nil {
		http.Error(w, "company not found", 404)
		return
	}

	pool, err := role.LoadCompanyRoles(h.configsDir, companyID, c.Industry)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	template, _ := workflow.LoadTemplate(filepath.Join(h.configsDir, "workflow_template.yaml"))
	engine := task.NewEngine(pool, template, h.executor, h.instanceStore, h.sessionStore, companyID, h.dataDir)
	engine.SetNotifier(h)

	// Set session to running
	sess.SetStatus(session.StatusRunning)
	h.sessionStore.Save(sess)
	requestID := uuid.New().String()
	h.NotifyProgress(sessionID, "workflow_status", "", "", "", "running", "", requestID, "")

	// Execute from the restarted step
	go func() {
		ctx := context.Background()
		slog.Info("Restarting workflow from step", "step", stepID, "session", sessionID)

		// Re-execute the target step with CEO opinion
		if req.CEOOpinion != "" {
			// Store opinion as instance input for the step
			// The engine will pick it up during ExecuteStep
		}

		if err := engine.RunWorkflow(ctx, sess); err != nil {
			slog.Error("Workflow restart failed", "error", err)
		}
		h.sessionStore.Save(sess)
		slog.Info("Workflow restart completed", "session", sessionID, "status", sess.Status)
	}()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"step_id":       stepID,
		"cleared_steps": clearedSteps,
		"session_id":    sessionID,
	})
}

// ListCompanyRoles returns roles available in a company.
func (h *Handlers) ListCompanyRoles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	c, err := h.companyStore.Get(vars["id"])
	if err != nil {
		http.Error(w, "company not found", 404)
		return
	}

	pool, err := role.LoadCompanyRoles(h.configsDir, vars["id"], c.Industry)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(pool.List())
}

// GetRole returns a specific role configuration with system prompt.
func (h *Handlers) GetRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	roleID := vars["roleId"]

	c, err := h.companyStore.Get(companyID)
	if err != nil {
		http.Error(w, "company not found", 404)
		return
	}

	pool, err := role.LoadCompanyRoles(h.configsDir, companyID, c.Industry)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	rl, err := pool.Get(roleID)
	if err != nil {
		http.Error(w, "role not found", 404)
		return
	}

	json.NewEncoder(w).Encode(rl)
}

// GetReview returns workflow review information for CEO.
func (h *Handlers) GetReview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]

	sess, err := h.sessionStore.Get(companyID, sessionID)
	if err != nil {
		http.Error(w, "session not found", 404)
		return
	}

	c, err := h.companyStore.Get(companyID)
	if err != nil {
		http.Error(w, "company not found", 404)
		return
	}

	pool, err := role.LoadCompanyRoles(h.configsDir, companyID, c.Industry)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Build review data with role prompts
	type StepReview struct {
		*workflow.Step
		RoleName        string `json:"role_name"`
		RolePrompt      string `json:"role_prompt"`
		IsDecisionPoint bool   `json:"is_decision_point"`
	}

	var steps []StepReview
	for _, step := range sess.Workflow.Steps {
		rl, err := pool.Get(step.Role)
		if err != nil {
			rl = &role.Role{Name: step.Role, SystemPrompt: "Role not found"}
		}
		steps = append(steps, StepReview{
			Step:            step,
			RoleName:        rl.Name,
			RolePrompt:      rl.SystemPrompt,
			IsDecisionPoint: step.IsDecisionPoint,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sess.ID,
		"status":     sess.Status,
		"goal":       sess.Goal,
		"steps":      steps,
	})
}

// ApproveWorkflow approves workflow for execution.
func (h *Handlers) ApproveWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]

	sess, err := h.sessionStore.Get(companyID, sessionID)
	if err != nil {
		http.Error(w, "session not found", 404)
		return
	}

	c, err := h.companyStore.Get(companyID)
	if err != nil {
		http.Error(w, "company not found", 404)
		return
	}

	pool, err := role.LoadCompanyRoles(h.configsDir, companyID, c.Industry)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	engine := task.NewEngine(pool, nil, h.executor, h.instanceStore, h.sessionStore, companyID, h.dataDir)
	if err := engine.ApproveWorkflow(sess); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	h.sessionStore.Save(sess)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sess.ID,
		"status":     sess.Status,
	})
}

// StartWorkflow starts automatic workflow execution.
// 异步执行：立即返回，后台继续执行工作流
func (h *Handlers) StartWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]

	slog.Info("StartWorkflow called", "company", companyID, "session", sessionID)

	sess, err := h.sessionStore.Get(companyID, sessionID)
	if err != nil {
		slog.Error("StartWorkflow ERROR: session not found")
		http.Error(w, "session not found", 404)
		return
	}

	if sess.Status != session.StatusApproved {
		slog.Error("StartWorkflow ERROR: status not approved", "status", sess.Status)
		http.Error(w, "workflow must be approved to start", 400)
		return
	}

	// Get company for role pool
	c, err := h.companyStore.Get(companyID)
	if err != nil {
		slog.Error("StartWorkflow ERROR: company not found")
		http.Error(w, "company not found", 404)
		return
	}

	pool, err := role.LoadCompanyRoles(h.configsDir, companyID, c.Industry)
	if err != nil {
		slog.Error("StartWorkflow ERROR: failed to load roles", "error", err)
		http.Error(w, err.Error(), 500)
		return
	}

	template, _ := workflow.LoadTemplate(filepath.Join(h.configsDir, "workflow_template.yaml"))
	engine := task.NewEngine(pool, template, h.executor, h.instanceStore, h.sessionStore, companyID, h.dataDir)
	engine.SetNotifier(h) // Enable real-time progress notifications

	// 立即设置状态为 running，保存后返回
	sess.SetStatus(session.StatusRunning)
	h.sessionStore.Save(sess)
	requestID := uuid.New().String()
	h.NotifyProgress(sessionID, "workflow_status", "", "", "", "running", "", requestID, "")

	// 异步执行工作流
	go func() {
		ctx := context.Background()
		slog.Info("RunWorkflow started in background", "session", sessionID)
		if err := engine.RunWorkflow(ctx, sess); err != nil {
			slog.Error("RunWorkflow FAILED", "session", sessionID, "error", err)
		}
		h.sessionStore.Save(sess)
		slog.Info("RunWorkflow completed", "session", sessionID, "status", sess.Status)
	}()

	// 立即返回
	slog.Info("StartWorkflow response sent", "session", sessionID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sess.ID,
		"status":     sess.Status,
		"message":    "Workflow started in background",
	})
}

// ResumeWorkflow resumes workflow execution after failure/pause.
// 异步执行：立即返回，后台继续执行工作流
func (h *Handlers) ResumeWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]

	slog.Info("ResumeWorkflow called", "company", companyID, "session", sessionID)

	sess, err := h.sessionStore.Get(companyID, sessionID)
	if err != nil {
		slog.Error("ResumeWorkflow ERROR: session not found")
		http.Error(w, "session not found", 404)
		return
	}

	if sess.Status != session.StatusFailed && sess.Status != session.StatusPaused {
		slog.Error("ResumeWorkflow ERROR: status not failed/paused", "status", sess.Status)
		http.Error(w, "workflow must be failed or paused to resume", 400)
		return
	}

	// Get company for role pool
	c, err := h.companyStore.Get(companyID)
	if err != nil {
		slog.Error("ResumeWorkflow ERROR: company not found")
		http.Error(w, "company not found", 404)
		return
	}

	pool, err := role.LoadCompanyRoles(h.configsDir, companyID, c.Industry)
	if err != nil {
		slog.Error("ResumeWorkflow ERROR: failed to load roles", "error", err)
		http.Error(w, err.Error(), 500)
		return
	}

	template, _ := workflow.LoadTemplate(filepath.Join(h.configsDir, "workflow_template.yaml"))
	engine := task.NewEngine(pool, template, h.executor, h.instanceStore, h.sessionStore, companyID, h.dataDir)
	engine.SetNotifier(h) // Enable real-time progress notifications

	// 立即设置状态为 running，保存后返回
	sess.SetStatus(session.StatusRunning)
	h.sessionStore.Save(sess)
	requestID := uuid.New().String()
	h.NotifyProgress(sessionID, "workflow_status", "", "", "", "running", "", requestID, "")

	// 异步执行工作流
	go func() {
		ctx := context.Background()
		slog.Info("RunWorkflow resumed in background", "session", sessionID)
		if err := engine.RunWorkflow(ctx, sess); err != nil {
			slog.Error("RunWorkflow FAILED", "session", sessionID, "error", err)
		}
		h.sessionStore.Save(sess)
		slog.Info("RunWorkflow completed", "session", sessionID, "status", sess.Status)
	}()

	// 立即返回
	slog.Info("ResumeWorkflow response sent", "session", sessionID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sess.ID,
		"status":     sess.Status,
		"message":    "Workflow resumed in background",
	})
}

// ListSessionOutputs returns list of agent output files.
func (h *Handlers) ListSessionOutputs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]

	// 步骤文档存放在 workspace 目录
	workspaceDir := filepath.Join(h.dataDir, companyID, "sessions", sessionID, "workspace")
	files, err := os.ReadDir(workspaceDir)
	if err != nil {
		// No workspace directory - return empty list
		json.NewEncoder(w).Encode([]string{})
		return
	}

	outputs := []string{}  // 初始化为空数组，避免 JSON null
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			outputs = append(outputs, f.Name())
		}
	}

	json.NewEncoder(w).Encode(outputs)
}

// GetSessionOutput returns content of a specific output file.
func (h *Handlers) GetSessionOutput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]
	filename := vars["filename"]

	// Validate filename - prevent path traversal
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		http.Error(w, "invalid filename", 400)
		return
	}
	if !strings.HasSuffix(filename, ".md") {
		http.Error(w, "invalid file type", 400)
		return
	}

	path := filepath.Join(h.dataDir, companyID, "sessions", sessionID, "workspace", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "file not found", 404)
		return
	}

	w.Header().Set("Content-Type", "text/markdown")
	w.Write(data)
}

// ListSessionFinalOutputs returns list of final output files (outputs directory).
func (h *Handlers) ListSessionFinalOutputs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]

	// 最终产物存放在 outputs 目录
	outputsDir := filepath.Join(h.dataDir, companyID, "sessions", sessionID, "outputs")
	files, err := os.ReadDir(outputsDir)
	if err != nil {
		// No outputs directory - return empty list
		json.NewEncoder(w).Encode([]string{})
		return
	}

	outputs := []string{}  // 初始化为空数组，避免 JSON null
	for _, f := range files {
		if !f.IsDir() {
			outputs = append(outputs, f.Name())
		}
	}

	json.NewEncoder(w).Encode(outputs)
}

// GetSessionFinalOutput returns content of a specific final output file.
func (h *Handlers) GetSessionFinalOutput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	companyID := vars["id"]
	sessionID := vars["sid"]
	filename := vars["filename"]

	// Validate filename - prevent path traversal
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		http.Error(w, "invalid filename", 400)
		return
	}

	path := filepath.Join(h.dataDir, companyID, "sessions", sessionID, "outputs", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "file not found", 404)
		return
	}

	// Return raw content
	w.Write(data)
}

// GetStepHistory returns conversation history for a step.
func (h *Handlers) GetStepHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sid"]
	stepID := vars["stepId"]

	// 构建 sessionID: {sessionID}-{stepID}
	execSessionID := fmt.Sprintf("%s-%s", sessionID, stepID)

	// 获取历史消息
	ctx := context.Background()
	limit := 100 // 最近 100 条消息

	if h.executor == nil {
		http.Error(w, "executor not configured", 500)
		return
	}

	history, err := h.executor.GetSessionHistory(ctx, execSessionID, limit)
	if err != nil {
		slog.Warn("Failed to get step history", "error", err)
		// 返回空历史而不是错误
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	// 转换为前端友好的格式
	var result []map[string]interface{}
	for _, msg := range history {
		// 从 ContentBlocks 提取文本内容
		var contentText string
		for _, block := range msg.ContentBlocks {
			if tb, ok := block.(memory.TextBlock); ok {
				contentText = tb.Text
				break
			}
		}

		result = append(result, map[string]interface{}{
			"role":       msg.Role,
			"type":       string(msg.Type),
			"content":    contentText,
			"timestamp":  msg.Timestamp,
		})
	}

	json.NewEncoder(w).Encode(result)
}