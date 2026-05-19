package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oneliang/company/internal/agent"
	"github.com/oneliang/company/internal/logging"
	"github.com/oneliang/company/internal/role"
	"github.com/oneliang/company/internal/session"
	"github.com/oneliang/company/internal/workflow"
)

// ProgressNotifier notifies progress updates.
type ProgressNotifier interface {
	NotifyProgress(sessionID string, eventType string, stepID string, role string, action string, status string, progress string, errMsg string)
}

// Engine handles task decomposition and workflow execution.
type Engine struct {
	pool          *role.Pool
	template      *workflow.Template
	executor      *agent.Executor
	instanceStore *role.InstanceStore
	sessionStore  *session.Store
	companyID     string
	dataDir       string
	notifier      ProgressNotifier // Progress notifier for real-time updates
}

// NewEngine creates a task engine.
func NewEngine(pool *role.Pool, template *workflow.Template, executor *agent.Executor, instanceStore *role.InstanceStore, sessionStore *session.Store, companyID string, dataDir string) *Engine {
	return &Engine{
		pool:          pool,
		template:      template,
		executor:      executor,
		instanceStore: instanceStore,
		sessionStore:  sessionStore,
		companyID:     companyID,
		dataDir:       dataDir,
	}
}

// SetNotifier sets the progress notifier.
func (e *Engine) SetNotifier(notifier ProgressNotifier) {
	e.notifier = notifier
}

// CreateWorkflow generates a workflow from template for a session.
func (e *Engine) CreateWorkflow(sess *session.Session) (*workflow.Workflow, error) {
	wf := e.template.ToWorkflow()
	sess.Workflow = wf
	sess.SetStatus(session.StatusDraft) // Draft status - waiting for CEO approval
	return wf, nil
}

// DecomposeTask generates workflow dynamically by calling LLM.
// Falls back to template if executor is not available or LLM fails.
func (e *Engine) DecomposeTask(ctx context.Context, sess *session.Session) (*workflow.Workflow, error) {
	// Fallback to template if no executor
	if e.executor == nil {
		logging.Debug("DecomposeTask: executor nil, fallback to template")
		return e.CreateWorkflow(sess)
	}

	// Build decomposition prompt
	prompt := e.buildDecomposePrompt(sess.Goal)

	// Call LLM to decompose task
	systemPrompt := "你是任务分解专家，输出JSON格式的步骤列表，不要包含其他解释文字。"
	response, err := e.executor.ExecutePlain(ctx, systemPrompt, prompt)
	if err != nil {
		logging.Warn("DecomposeTask: LLM error, fallback to template", "error", err)
		return e.CreateWorkflow(sess)
	}

	logging.Debug("DecomposeTask: LLM response", "response", response)

	// Parse JSON response
	steps, err := e.parseStepsFromJSON(response)
	if err != nil {
		logging.Warn("DecomposeTask: parse error, fallback to template", "error", err)
		return e.CreateWorkflow(sess)
	}

	logging.Debug("DecomposeTask: parsed steps", "count", len(steps))

	// Create workflow from parsed steps
	wf := &workflow.Workflow{Steps: steps, Status: "pending"}
	sess.Workflow = wf
	sess.SetStatus(session.StatusDraft)
	return wf, nil
}

// buildDecomposePrompt constructs the task decomposition prompt.
func (e *Engine) buildDecomposePrompt(goal string) string {
	prompt := `根据以下任务描述，生成工作流步骤。

任务：` + goal + `

可用角色：
- pm: 产品经理，分析需求、分解任务
- techlead: 技术负责人，设计架构、技术方案
- dev: 开发工程师，实现代码
- qa: QA工程师，测试验证
- ceo: CEO，决策审批

输出格式（仅输出JSON数组，不要其他文字）：
[
  {
    "id": "step1",
    "role": "pm",
    "action": "analyze_requirements",
    "description": "分析用户需求",
    "depends_on": [],
    "is_decision_point": false
  }
]

要求：
1. 根据任务复杂度决定步骤数量（简单任务3-5步，复杂任务可更多）
2. 合理设置依赖关系（depends_on）
3. CEO决策点(is_decision_point=true)放在关键审批环节
4. 每个步骤要有明确的action和description
5. 步骤id使用英文，如：analyze, design, implement, review等`
	return prompt
}

// parseStepsFromJSON parses LLM response into workflow steps.
func (e *Engine) parseStepsFromJSON(jsonStr string) ([]*workflow.Step, error) {
	// Extract JSON array from response (handle markdown code blocks)
	jsonStr = extractJSON(jsonStr)

	var stepDefs []struct {
		ID              string   `json:"id"`
		Role            string   `json:"role"`
		Action          string   `json:"action"`
		Description     string   `json:"description"`
		DependsOn       []string `json:"depends_on"`
		IsDecisionPoint bool     `json:"is_decision_point"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &stepDefs); err != nil {
		return nil, err
	}

	steps := make([]*workflow.Step, len(stepDefs))
	for i, def := range stepDefs {
		steps[i] = &workflow.Step{
			ID:              def.ID,
			Role:            def.Role,
			Action:          def.Action,
			Description:     def.Description,
			Status:          workflow.StepPending,
			DependsOn:       def.DependsOn,
			IsDecisionPoint: def.IsDecisionPoint,
		}
	}

	return steps, nil
}

// extractJSON extracts the first JSON array from response.
func extractJSON(s string) string {
	// Handle markdown code blocks
	if strings.Contains(s, "```json") {
		start := strings.Index(s, "```json") + 7
		end := strings.Index(s[start:], "```")
		if end > 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	if strings.Contains(s, "```") {
		start := strings.Index(s, "```") + 3
		// Skip language identifier if present
		if start < len(s) {
			// Find newline after language identifier
			newline := strings.Index(s[start:], "\n")
			if newline > 0 && newline < 20 {
				start += newline + 1
			}
		}
		end := strings.Index(s[start:], "```")
		if end > 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	// Find first JSON array only
	start := strings.Index(s, "[")
	if start < 0 {
		return s
	}
	// Find matching closing bracket for first array
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '[' {
			depth++
		} else if s[i] == ']' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return s[start:]
}

// ApproveWorkflow marks workflow as approved by CEO.
func (e *Engine) ApproveWorkflow(sess *session.Session) error {
	if sess.Status != session.StatusDraft {
		return errors.New("workflow must be in draft status to approve")
	}
	sess.SetStatus(session.StatusApproved)
	return nil
}

// ExecuteStep executes a workflow step using the role's agent.
func (e *Engine) ExecuteStep(ctx context.Context, sess *session.Session, stepID string) error {
	logging.Info("ExecuteStep START", "session", sess.ID, "step", stepID)

	// Check if workflow is approved
	if sess.Status != session.StatusApproved && sess.Status != session.StatusRunning {
		logging.Warn("ExecuteStep BLOCKED", "status", sess.Status)
		return errors.New("workflow must be approved before execution")
	}

	if e.executor == nil {
		logging.Error("ExecuteStep ERROR: executor not configured")
		return errors.New("agent executor not configured")
	}

	// Set status to running on first execution
	if sess.Status == session.StatusApproved {
		sess.SetStatus(session.StatusRunning)
		logging.Info("Workflow status changed", "from", "approved", "to", "running")
		// Notify progress
		if e.notifier != nil {
			e.notifier.NotifyProgress(sess.ID, "workflow_status", "", "", "", "running", "", "")
		}
	}

	step := sess.Workflow.GetStep(stepID)
	if step == nil {
		logging.Error("ExecuteStep ERROR: step not found", "step", stepID)
		return errors.New("step not found: " + stepID)
	}

	logging.Debug("Step info", "role", step.Role, "action", step.Action, "status", step.Status)

	// Skip CEO decision points - they require human input
	if step.IsDecisionPoint {
		logging.Info("ExecuteStep SKIPPED: decision point", "step", stepID)
		return errors.New("step is a decision point, requires CEO approval")
	}

	// Check dependencies
	logging.Debug("Checking dependencies", "step", stepID)
	for _, depID := range step.DependsOn {
		dep := sess.Workflow.GetStep(depID)
		if dep == nil {
			logging.Debug("Dependency NOT FOUND", "dep", depID)
		} else {
			logging.Debug("Dependency status", "dep", depID, "status", dep.Status)
		}
	}

	// Get role configuration
	r, err := e.pool.Get(step.Role)
	if err != nil {
		logging.Error("ExecuteStep ERROR: role not found", "role", step.Role, "error", err)
		return err
	}

	// Create role instance
	instance := role.NewInstance(r.ID, sess.ID, stepID)

	// Build context from dependency outputs
	instance.SetContext(e.buildContext(sess.Workflow, step))

	// Build input from action
	instance.SetInput(e.buildInput(sess.Workflow, step, sess))

	// Mark instance as running
	instance.SetRunning()

	// Save instance before execution
	if e.instanceStore != nil {
		e.instanceStore.Save(e.companyID, instance)
	}

	// Mark step as running and notify progress
	step.Status = workflow.StepRunning
	sess.CurrentStep = stepID // 设置当前执行步骤，前端通过 getSession 获取显示
	logging.Info("Step marked as running", "step", stepID, "role", step.Role, "action", step.Action)

	// 保存 session（确保状态持久化，getSession 能获取最新状态）
	if e.sessionStore != nil {
		e.sessionStore.Save(sess)
	}

	if e.notifier != nil {
		e.notifier.NotifyProgress(sess.ID, "step_start", stepID, step.Role, step.Action, "running", "", "")
	}

	// Execute via agent executor
	output, err := e.executor.ExecuteInstance(ctx, r, instance)
	if err != nil {
		step.Status = workflow.StepFailed
		sess.CurrentStep = "" // 步骤失败，清空当前执行步骤
		instance.SetFailed()
		if e.instanceStore != nil {
			e.instanceStore.Save(e.companyID, instance)
		}
		logging.Error("ExecuteStep FAILED", "step", stepID, "error", err)
		if e.notifier != nil {
			e.notifier.NotifyProgress(sess.ID, "step_failed", stepID, step.Role, step.Action, "failed", "", err.Error())
		}
		return err
	}

	// Set output and mark as completed
	instance.SetOutput(output)
	step.Status = workflow.StepCompleted
	sess.CurrentStep = "" // 步骤完成，清空当前执行步骤
	step.Output = output

	// Save instance after execution
	if e.instanceStore != nil {
		e.instanceStore.Save(e.companyID, instance)
	}

	// 立即保存 session（确保状态持久化）
	if e.sessionStore != nil {
		e.sessionStore.Save(sess)
	}

	// Count completed steps and notify progress
	completedCount := 0
	for _, s := range sess.Workflow.Steps {
		if s.Status == workflow.StepCompleted {
			completedCount++
		}
	}
	progress := fmt.Sprintf("%d/%d", completedCount, len(sess.Workflow.Steps))
	logging.Info("ExecuteStep SUCCESS", "step", stepID, "progress", progress)

	// 推送完整 workflow 状态（让前端同步所有步骤）
	if e.notifier != nil {
		e.notifier.NotifyProgress(sess.ID, "step_complete", stepID, step.Role, step.Action, "completed", progress, "")
		// 推送完整 workflow 更新
		e.notifier.NotifyProgress(sess.ID, "workflow_update", "", "", "", "", progress, "")
	}

	// Write agent output to markdown file
	if err := e.writeStepOutputMd(sess.ID, step, output); err != nil {
		logging.Warn("Failed to write output md file", "error", err)
	}

	return nil
}

// writeStepOutputMd writes agent output as markdown file to workspace directory.
func (e *Engine) writeStepOutputMd(sessionID string, step *workflow.Step, output string) error {
	// 步骤文档写入 workspace 目录（工作空间中间产物）
	workspaceDir := filepath.Join(e.dataDir, e.companyID, "sessions", sessionID, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return err
	}

	filename := "step-" + step.ID + ".md"
	path := filepath.Join(workspaceDir, filename)

	var md strings.Builder
	md.WriteString("# " + step.Action + "\n\n")
	md.WriteString("**Role:** " + step.Role + "\n")
	md.WriteString("**Step:** " + step.ID + "\n")
	md.WriteString("**Description:** " + step.Description + "\n")
	md.WriteString("**Status:** Completed\n\n")
	md.WriteString("---\n\n")
	md.WriteString("## Output\n\n")
	md.WriteString(output)

	return os.WriteFile(path, []byte(md.String()), 0644)
}

// RunWorkflow executes all ready steps in workflow until completion or error.
func (e *Engine) RunWorkflow(ctx context.Context, sess *session.Session) error {
	logging.Info("RunWorkflow START", "session", sess.ID, "goal", sess.Goal)

	sess.SetStatus(session.StatusRunning)
	logging.Debug("Workflow status set to running")

	// Reset failed steps to pending for retry
	for _, step := range sess.Workflow.Steps {
		if step.Status == workflow.StepFailed {
			logging.Info("Resetting failed step for retry", "step", step.ID)
			step.Status = workflow.StepPending
			step.Output = ""
		}
	}

	// Save session immediately after status change to running
	// This ensures frontend sees correct status even if HTTP request times out
	if e.sessionStore != nil {
		e.sessionStore.Save(sess)
		logging.Debug("Session saved with status=running")
	}
	if e.notifier != nil {
		e.notifier.NotifyProgress(sess.ID, "workflow_status", "", "", "", "running", "", "")
	}

	for {
		// Get ready steps (dependencies satisfied)
		ready := sess.Workflow.GetReadySteps()
		logging.Debug("GetReadySteps", "count", len(ready))

		if len(ready) == 0 {
			// Check if all completed
			allDone := true
			for _, s := range sess.Workflow.Steps {
				if s.Status != workflow.StepCompleted {
					allDone = false
					break
				}
			}

			if allDone {
				sess.SetStatus(session.StatusCompleted)
				logging.Info("Workflow COMPLETED", "session", sess.ID)
				if e.notifier != nil {
					e.notifier.NotifyProgress(sess.ID, "workflow_status", "", "", "", "completed", "", "")
				}
				return nil
			}

			// Still has pending steps but none ready - blocked (e.g., CEO decision pending)
			sess.SetStatus(session.StatusPaused)
			logging.Info("Workflow PAUSED", "session", sess.ID, "reason", "waiting for blocked steps")
			if e.notifier != nil {
				e.notifier.NotifyProgress(sess.ID, "workflow_status", "", "", "", "paused", "", "")
			}
			return nil
		}

		// Check if any ready step is a decision point - pause workflow for human input
		hasDecisionPoint := false
		for _, step := range ready {
			if step.IsDecisionPoint {
				hasDecisionPoint = true
				break
			}
		}
		if hasDecisionPoint {
			sess.SetStatus(session.StatusPaused)
			logging.Info("Workflow PAUSED", "reason", "decision point requires human approval")
			if e.sessionStore != nil {
				e.sessionStore.Save(sess)
			}
			if e.notifier != nil {
				e.notifier.NotifyProgress(sess.ID, "workflow_status", "", "", "", "paused", "", "")
			}
			return nil
		}

		// Execute all ready steps (none are decision points at this point)
		for _, step := range ready {
			logging.Info("Executing step", "step", step.ID, "role", step.Role, "action", step.Action)

			if err := e.ExecuteStep(ctx, sess, step.ID); err != nil {
				sess.SetStatus(session.StatusFailed)
				logging.Error("Workflow FAILED", "step", step.ID, "error", err)
				// Save failed state
				if e.sessionStore != nil {
					e.sessionStore.Save(sess)
				}
				if e.notifier != nil {
					e.notifier.NotifyProgress(sess.ID, "workflow_status", "", "", "", "failed", "", err.Error())
				}
				return err
			}

			// Save session after each step to preserve progress
			if e.sessionStore != nil {
				e.sessionStore.Save(sess)
				logging.Debug("Session saved after step", "step", step.ID)
			}
		}
	}
}

func (e *Engine) buildContext(wf *workflow.Workflow, step *workflow.Step) string {
	if len(step.DependsOn) == 0 {
		return ""
	}
	var ctx strings.Builder
	ctx.WriteString("Previous step results:\n")
	for _, depID := range step.DependsOn {
		dep := wf.GetStep(depID)
		if dep != nil && dep.Output != "" {
			ctx.WriteString("- " + dep.Role + " (" + depID + "): " + dep.Output + "\n")
		}
	}
	return ctx.String()
}

// buildInput constructs the input prompt for a step.
// sess: session for workspace path context
func (e *Engine) buildInput(wf *workflow.Workflow, step *workflow.Step, sess *session.Session) string {
	// 输出路径约束（Aura Agent 需要外部告知输出路径）
	// outputs：最终交付产物（代码、设计稿、文档等）
	outputPath := filepath.Join(e.dataDir, e.companyID, "sessions", sess.ID, "outputs")

	input := "Task: " + step.Action + "\n\n"
	input += "Description: " + step.Description + "\n\n"

	// 输出路径约束
	input += "输出文件路径：" + outputPath + "\n"
	input += "所有文件写入操作（代码、设计稿等最终产物）必须使用输出文件路径。\n\n"

	// Include goal context
	input += "Goal: Complete this step in the workflow.\n\n"

	// Include outputs from dependencies
	if len(step.DependsOn) > 0 {
		input += "Previous results:\n"
		for _, depID := range step.DependsOn {
			dep := wf.GetStep(depID)
			if dep != nil && dep.Output != "" {
				input += "- " + dep.Role + " output: " + dep.Output + "\n"
			}
		}
		input += "\n"
	}

	input += "Please complete this task and provide your output."

	return input
}

// HandleDecision processes CEO decision and continues workflow.
// decisionType: approve (批准), continue (追加意见继续), restart (重新执行)
func (e *Engine) HandleDecision(ctx context.Context, sess *session.Session, stepID string, decisionType string, content string) error {
	step := sess.Workflow.GetStep(stepID)
	if step == nil {
		return errors.New("step not found: " + stepID)
	}

	switch decisionType {
	case "approve":
		// 批准，标记步骤完成
		sess.Workflow.CompleteStep(stepID, content)
		logging.Info("Decision approved", "step", stepID)

	case "continue":
		// 追加 CEO 意见继续执行（保留历史）
		if e.executor == nil {
			return errors.New("executor not configured")
		}

		// 获取角色配置
		r, err := e.pool.Get(step.Role)
		if err != nil {
			return err
		}

		// 创建实例
		instance := role.NewInstance(r.ID, sess.ID, stepID)
		instance.SetContext(e.buildContext(sess.Workflow, step))
		instance.SetInput(e.buildInput(sess.Workflow, step, sess))

		// SessionID 用于保留历史
		sessionID := fmt.Sprintf("%s-%s", sess.ID, stepID)

		// 执行（追加 CEO 意见）
		output, err := e.executor.ExecuteInstanceWithHistory(ctx, r, instance, sessionID, content)
		if err != nil {
			logging.Error("Continue execution failed", "step", stepID, "error", err)
			return err
		}

		// 更新步骤输出
		sess.Workflow.CompleteStep(stepID, output)
		logging.Info("Decision continued", "step", stepID)

	case "restart":
		// 清空历史，重新执行
		if e.executor == nil {
			return errors.New("executor not configured")
		}

		// 获取角色配置
		r, err := e.pool.Get(step.Role)
		if err != nil {
			return err
		}

		// 创建实例
		instance := role.NewInstance(r.ID, sess.ID, stepID)
		instance.SetContext(e.buildContext(sess.Workflow, step))
		instance.SetInput(e.buildInput(sess.Workflow, step, sess))

		// 新 SessionID（清空历史）
		sessionID := fmt.Sprintf("%s-%s-restart-%d", sess.ID, stepID, time.Now().Unix())

		// 重新执行
		output, err := e.executor.ExecuteInstanceWithHistory(ctx, r, instance, sessionID, content)
		if err != nil {
			logging.Error("Restart execution failed", "step", stepID, "error", err)
			return err
		}

		// 更新步骤输出
		sess.Workflow.CompleteStep(stepID, output)
		logging.Info("Decision restarted", "step", stepID)

	default:
		return errors.New("unknown decision type: " + decisionType)
	}

	return nil
}