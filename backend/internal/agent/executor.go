package agent

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/storage/pkg/jsonl"
	"github.com/oneliang/aura/storage/pkg/message"
	"github.com/oneliang/company/internal/logging"
	"github.com/oneliang/company/internal/role"
)

// Executor wraps aura SDK for role-based agent execution.
type Executor struct {
	config       *Config
	sessionStore *jsonl.MessageStore // 消息持久化存储
}

// NewExecutor creates an agent executor from config file.
func NewExecutor(configPath string, dataDir string) (*Executor, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// 创建消息持久化存储
	store, err := jsonl.NewMessageStore(filepath.Join(dataDir, "messages"))
	if err != nil {
		return nil, err
	}

	return &Executor{
		config:       cfg,
		sessionStore: store,
	}, nil
}

// ExecuteStep runs an agent step with role's persona using aura SDK.
func (e *Executor) ExecuteStep(ctx context.Context, r *role.Role, input string) (string, error) {
	// Create runtime config with role's system prompt
	cfg := sdk.DefaultRuntimeConfig()

	// Set role persona
	cfg.SystemPrompt = r.SystemPrompt

	// Set LLM configuration from config.yaml
	cfg.LLM.Provider = e.config.LLM.Provider
	cfg.LLM.Model = e.config.LLM.Model
	cfg.LLM.APIKey = e.config.LLM.APIKey
	cfg.LLM.BaseURL = e.config.LLM.BaseURL

	// Set agent temperature
	cfg.Agent.Temperature = e.config.Agent.Temperature

	// Set permissions to allow all tools (auto-approve without confirmation)
	cfg.Permissions.DefaultLevel = "allow"

	// GLM-5 requires max_completion_tokens > thinking_budget (default 32768)
	// Set BudgetTokens large enough to satisfy this constraint
	cfg.LLM.Thinking.Enabled = true
	cfg.LLM.Thinking.BudgetTokens = 40960 // Used as max_completion_tokens, must > 32768

	// Create runtime
	runtime, err := sdk.NewRuntime(cfg)
	if err != nil {
		return "", err
	}

	// Initialize with timeout
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := runtime.Initialize(initCtx); err != nil {
		return "", err
	}
	defer runtime.Shutdown()

	// Start event stream
	if err := runtime.Start(ctx); err != nil {
		return "", err
	}
	defer runtime.Stop(ctx)

	// Get output channel
	events := runtime.Events()

	// Generate RequestID for tracing
	requestID := uuid.New().String()

	// Send input event
	logging.Debug("Aura input",
		"method", "ExecuteStep",
		"request_id", requestID,
		"role", r.ID,
		"system_prompt", truncate(cfg.SystemPrompt, 500),
		"input", truncate(input, 500),
		"model", cfg.LLM.Model,
	)
	if err := runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, input, requestID)); err != nil {
		return "", err
	}

	// Process event stream
	response, err := processEventStream(events)
	if err != nil {
		return "", err
	}

	logging.Debug("Aura response",
		"method", "ExecuteStep",
		"request_id", requestID,
		"response", truncate(response, 2000),
	)
	return response, nil
}

// ExecuteInstance runs an agent with role instance context.
// Deprecated: Use ExecuteInstanceWithHistory instead.
func (e *Executor) ExecuteInstance(ctx context.Context, r *role.Role, instance *role.Instance) (string, error) {
	return e.ExecuteInstanceWithHistory(ctx, r, instance, "", "")
}

// ExecuteInstanceWithHistory runs an agent with role instance context and session history.
// sessionID: used to load/save conversation history
// additionalInput: CEO opinion to append to the conversation (for continue mode)
func (e *Executor) ExecuteInstanceWithHistory(ctx context.Context, r *role.Role, instance *role.Instance, sessionID string, additionalInput string) (string, error) {
	// Build full prompt from role template + instance context
	fullPrompt := instance.GetFullPrompt(r)

	// Create runtime config
	cfg := sdk.DefaultRuntimeConfig()
	cfg.SystemPrompt = fullPrompt

	// Set LLM configuration
	cfg.LLM.Provider = e.config.LLM.Provider
	cfg.LLM.Model = e.config.LLM.Model
	cfg.LLM.APIKey = e.config.LLM.APIKey
	cfg.LLM.BaseURL = e.config.LLM.BaseURL
	cfg.Agent.Temperature = e.config.Agent.Temperature

	// Set permissions to allow all tools (auto-approve without confirmation)
	cfg.Permissions.DefaultLevel = "allow"

	// GLM-5 requires max_completion_tokens > thinking_budget (default 32768)
	cfg.LLM.Thinking.Enabled = true
	cfg.LLM.Thinking.BudgetTokens = 40960 // Used as max_completion_tokens, must > 32768

	// Build input
	input := instance.Input
	if additionalInput != "" {
		input = input + "\n\nCEO补充意见：" + additionalInput
	}

	// Create runtime with optional session history
	var runtimeOpts []sdk.RuntimeOption
	if sessionID != "" && e.sessionStore != nil {
		runtimeOpts = append(runtimeOpts,
			sdk.WithSessionStore(e.sessionStore),
			sdk.WithSessionID(sessionID),
		)
	}

	runtime, err := sdk.NewRuntime(cfg, runtimeOpts...)
	if err != nil {
		return "", err
	}

	// Initialize with timeout
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := runtime.Initialize(initCtx); err != nil {
		return "", err
	}
	defer runtime.Shutdown()

	// Start event stream
	if err := runtime.Start(ctx); err != nil {
		return "", err
	}
	defer runtime.Stop(ctx)

	// Get output channel
	events := runtime.Events()

	// Generate RequestID for tracing
	requestID := uuid.New().String()

	// Send input event
	logging.Debug("Aura input",
		"method", "ExecuteInstanceWithHistory",
		"request_id", requestID,
		"session_id", sessionID,
		"role", r.ID,
		"instance_id", instance.ID,
		"system_prompt", truncate(fullPrompt, 500),
		"input", truncate(input, 500),
		"model", cfg.LLM.Model,
	)
	if err := runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, input, requestID)); err != nil {
		return "", err
	}

	// Process event stream
	response, err := processEventStream(events)
	if err != nil {
		return "", err
	}

	logging.Debug("Aura response",
		"method", "ExecuteInstanceWithHistory",
		"request_id", requestID,
		"session_id", sessionID,
		"response", truncate(response, 2000),
	)
	return response, nil
}

// GetSessionHistory retrieves conversation history for a session.
func (e *Executor) GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]message.Message, error) {
	if e.sessionStore == nil {
		return nil, errors.New("session store not initialized")
	}
	return e.sessionStore.Get(ctx, sessionID, limit, "")
}

// ExecutePlain executes a plain prompt without role binding.
// Used for task decomposition and other utility LLM calls.
func (e *Executor) ExecutePlain(ctx context.Context, systemPrompt string, input string) (string, error) {
	cfg := sdk.DefaultRuntimeConfig()
	cfg.SystemPrompt = systemPrompt
	cfg.LLM.Provider = e.config.LLM.Provider
	cfg.LLM.Model = e.config.LLM.Model
	cfg.LLM.APIKey = e.config.LLM.APIKey
	cfg.LLM.BaseURL = e.config.LLM.BaseURL
	cfg.Agent.Temperature = e.config.Agent.Temperature

	// GLM-5 requires max_completion_tokens > thinking_budget (default 32768)
	cfg.LLM.Thinking.Enabled = true
	cfg.LLM.Thinking.BudgetTokens = 40960 // Used as max_completion_tokens, must > 32768

	runtime, err := sdk.NewRuntime(cfg)
	if err != nil {
		return "", err
	}

	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := runtime.Initialize(initCtx); err != nil {
		return "", err
	}
	defer runtime.Shutdown()

	// Start event stream
	if err := runtime.Start(ctx); err != nil {
		return "", err
	}
	defer runtime.Stop(ctx)

	// Get output channel
	events := runtime.Events()

	// Generate RequestID for tracing
	requestID := uuid.New().String()

	// Send input event
	logging.Debug("Aura input",
		"method", "ExecutePlain",
		"request_id", requestID,
		"system_prompt", truncate(systemPrompt, 500),
		"input", truncate(input, 500),
		"model", cfg.LLM.Model,
	)
	if err := runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, input, requestID)); err != nil {
		return "", err
	}

	// Process event stream
	response, err := processEventStream(events)
	if err != nil {
		return "", err
	}

	logging.Debug("Aura response",
		"method", "ExecutePlain",
		"request_id", requestID,
		"response", truncate(response, 2000),
	)
	return response, nil
}

// truncate limits string length for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}

// processEventStream collects response from event stream until Done event.
func processEventStream(events <-chan sdk.Event) (string, error) {
	var response strings.Builder
	for ev := range events {
		switch ev.Type() {
		case sdk.EventTypeResponse:
			response.WriteString(ev.Content())
		case sdk.EventTypeResponseChunk:
			response.WriteString(ev.Content())
		case sdk.EventTypeResponseEnd:
			// Response complete
		case sdk.EventTypeDone:
			return response.String(), nil
		case sdk.EventTypeError:
			return "", errors.New(ev.Content())
		}
	}
	return response.String(), nil
}