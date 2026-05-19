---
name: virtual-company-design
description: 虚拟公司系统设计方案 - 多 AI Agent 协作平台，CEO 驱动任务执行
---

# 虚拟公司系统设计方案

## Context

用户希望构建一个"虚拟公司"系统：一个多 AI Agent 协作平台，用户作为 CEO 与 Agent 团队交互完成任务。系统核心特点：
- 任务驱动组织，按需组建 Agent 团队
- 预设角色池 + 可扩展自定义
- 可视化工作流拓扑 + CEO 监控干预
- 行业模板预设 + 可视化设计器
- 复用 Aura 项目的核心组件（ReAct引擎、LLM客户端、Memory、Tool系统）
- MVP：软件公司开发小需求场景验证链路

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        WebUI (Frontend)                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐            │
│  │拓扑可视化│ │CEO交互   │ │状态监控  │ │设计器    │            │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘            │
└─────────────────────────────────────────────────────────────────┘
                              │ WebSocket / REST API
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Company Backend (Go)                          │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Company Core Layer                        ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       ││
│  │  │RolePool │ │TaskEngine│ │Workflow │ │CEOBridge│       ││
│  │  │角色池   │ │任务引擎  │ │工作流   │ │CEO接口  │       ││
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘       ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Aura Core (Reuse)                         ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       ││
│  │  │ReAct    │ │LLMClient│ │Memory   │ │Tool     │       ││
│  │  │Engine   │ │(OpenAI) │ │System   │ │System   │       ││
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘       ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Storage Layer                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                        │
│  │JSONL     │ │Config    │ │Template  │                        │
│  │Session   │ │YAML      │ │Store     │                        │
│  └──────────┘ └──────────┘ └──────────┘                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## Core Concepts

### 1. Role（角色）

角色是 Agent 的"身份模板"，定义了：
- **System Prompt**：角色的行为指令、职责描述
- **Tool Set**：角色可使用的工具集合
- **Personality**：性格、沟通风格
- **Skill Set**：角色具备的专业技能

```yaml
# 示例：软件工程师角色定义
name: software_engineer
description: 软件工程师，负责代码实现
system_prompt: |
  你是一名专业的软件工程师，职责是：
  - 理解需求并设计技术方案
  - 编写高质量代码
  - 编写单元测试
  风格：简洁、专业、注重代码质量
tools:
  - file_read
  - file_write
  - file_search
  - shell_execute
  - web_search
skills:
  - coding
  - testing
  - debugging
```

### 2. Task（任务）

任务是 CEO 发起的待完成事项：
- **Input**：任务描述、目标、约束条件
- **Workflow**：动态生成的执行计划（拓扑图）
- **Roles Required**：任务需要的角色列表
- **Acceptance Criteria**：验收标准（CEO 可介入定义）
- **Status**：pending → planning → executing → reviewing → completed

### 3. Workflow（工作流）

工作流是任务执行的拓扑结构：
- **Nodes**：各环节的任务步骤，绑定到具体 Role
- **Edges**：依赖关系（串行/并行）
- **State**：每个 Node 的执行状态
- **CEO Intervention Points**：决策点、验收点

```
┌─────────┐     ┌─────────┐     ┌─────────┐
│需求分析 │────▶│方案设计 │────▶│CEO审核  │
│(PM)    │     │(TechLead)│     │(Decision)│
└─────────┘     └─────────┘     └─────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    │                                   │
                    ▼                                   ▼
              ┌─────────┐                         ┌─────────┐
              │前端开发 │                         │后端开发 │
              │(FE Dev)│                         │(BE Dev) │
              └─────────┘                         └─────────┘
                    │                                   │
                    └─────────────────┬─────────────────┘
                                      │
                                      ▼
                              ┌─────────┐
                              │测试验收 │
                              │(QA)    │
                              └─────────┘
                                      │
                                      ▼
                              ┌─────────┐
                              │CEO确认  │
                              │(Final) │
                              └─────────┘
```

### 4. Universal Agent

核心设计：**单一 Agent 实现，角色由数据驱动**

```go
// Agent 初始化
agent := NewReActAgent(
    llmClient,      // Aura 的 LLM 客户端
    memory,         // Aura 的 Memory 系统
    tools,          // 根据角色动态加载的工具集
    systemPrompt,   // 角色的 System Prompt
)
```

不同角色 = 不同 `systemPrompt` + 不同 `tools` 组合

---

## Component Design

### Company Backend (Go)

**项目结构：**

```
company/
├── cmd/
│   └── server/
│       └── main.go          # HTTP/WebSocket 服务入口
├── internal/
│   ├── role/
│   │   ├── pool.go          # 角色池管理
│   │   ├── loader.go        # YAML 角色定义加载
│   │   └── definition.go    # Role 结构定义
│   ├── task/
│   │   ├── engine.go        # 任务引擎（解析、分解、生成工作流）
│   │   ├── workflow.go      # 工作流 DAG 定义和执行
│   │   ├── node.go          # 工作流节点
│   │   └── executor.go      # 节点执行器
│   ├── ceo/
│   │   ├── bridge.go        # CEO 交互接口
│   │   ├── decision.go      # 决策点处理
│   │   └── monitor.go       # 实时监控推送
│   ├── template/
│   │   ├── industry.go      # 行业模板加载
│   │   ├── store.go         # 模板存储
│   └── api/
│       ├── handlers.go      # REST API handlers
│       ├── websocket.go     # WebSocket 实时推送
│       └── routes.go        # 路由定义
├── configs/
│   ├── roles/               # 角色定义 YAML 文件
│   ├── templates/           # 行业模板
│   └── config.yaml          # 系统配置
├── pkg/                     # 公共包（可对外暴露）
└── go.mod                   # 依赖 Aura core modules
```

**核心模块职责：**

| 模块 | 职责 | Aura 复用 |
|------|------|-----------|
| role/pool | 角色池管理、动态加载 | 无 |
| task/engine | 任务解析、工作流生成 | 无 |
| task/workflow | DAG 执行、状态管理 | 无 |
| ceo/bridge | CEO 交互、决策处理 | 无 |
| api/* | HTTP/WebSocket 服务 | 无 |
| ReAct Agent | Agent 执行循环 | **复用 Aura engine** |
| LLM Client | OpenAI API 调用 | **复用 Aura llm/openai** |
| Memory | 会话记忆 | **复用 Aura memory** |
| Tools | 工具执行 | **复用 Aura tools** |

### Task Engine 设计

任务引擎是核心，职责：

1. **接收任务**：CEO 输入任务描述
2. **分解任务**：调用 LLM 生成工作流 DAG
3. **分配角色**：从 RolePool 选择合适的角色
4. **生成验收标准**：每个节点的验收条件
5. **执行工作流**：按 DAG 顺序启动 Agent
6. **状态推送**：实时推送执行状态到 WebUI
7. **CEO 干预**：在决策点暂停等待 CEO 确认

```go
// TaskEngine 核心接口
type TaskEngine interface {
    // 提交任务，返回生成的 Workflow
    Submit(ctx context.Context, task TaskInput) (*Workflow, error)

    // 执行工作流
    Execute(ctx context.Context, workflow *Workflow) error

    // 暂停/继续（CEO 干预）
    Pause(workflowID string) error
    Resume(workflowID string, decision Decision) error

    // 获取实时状态
    GetStatus(workflowID string) (*WorkflowStatus, error)
}
```

### Workflow DAG 执行

```go
type Workflow struct {
    ID          string
    TaskID      string
    Nodes       []*Node          // 节点列表
    Edges       []*Edge          // 依赖边
    Status      WorkflowStatus   // 整体状态
    CurrentNode string           // 当前执行节点
}

type Node struct {
    ID           string
    Role         Role             // 执行角色
    Task         string           // 节点任务描述
    Acceptance   string           // 验收标准
    Status       NodeStatus       // pending/running/completed/blocked
    IsDecision   bool             // 是否是 CEO 决策点
    Agent        *ReActAgent      // 执行的 Agent（Aura）
    Output       string           // 执行结果
}

type Edge struct {
    From string
    To   string
    Type EdgeType  // serial/parallel
}
```

执行逻辑：
- 遍历 DAG，找到可执行节点（前置依赖已完成）
- 为节点创建 Agent（绑定 Role）
- 执行 Agent ReAct 循环
- 完成后标记状态，触发下游节点
- 遇到决策点暂停，等待 CEO 输入

---

## WebUI Design

### 技术选型

| 前端技术 | 用途 |
|----------|------|
| React + TypeScript | 主流、组件化、类型安全 |
| React Flow | 拓扑图/工作流可视化（拖拽节点、连线） |
| WebSocket | 实时状态推送 |
| TailwindCSS | 快速样式开发 |

### 页面结构

```
/                    # 首页：任务入口
/tasks               # 任务列表
/tasks/:id           # 任务详情 + 工作流拓扑
/tasks/:id/monitor   # 实时监控 + CEO 交互
/designer            # 可视化设计器
/roles               # 角色池管理
/templates           # 行业模板管理
```

### 核心页面

**任务执行页（拓扑 + 监控 + 交互）**

```
┌────────────────────────────────────────────────────┐
│  任务：开发用户登录功能                              │
│  状态：执行中 (35%)                                 │
├────────────────────────────────────────────────────┤
│                                                    │
│  ┌─────────┐     ┌─────────┐     ┌─────────┐      │
│  │需求分析 │────▶│方案设计 │────▶│CEO审核  │      │
│  │ ✓ 完成 │     │ ✓ 完成 │     │ ⏸ 等待 │      │
│  └─────────┘     └─────────┘     └─────────┘      │
│                                      │             │
│                    ┌─────────────────┴───┐         │
│                    │                     │         │
│                    ▼                     ▼         │
│              ┌─────────┐           ┌─────────┐    │
│              │前端开发 │           │后端开发 │    │
│              │ ○ 待定 │           │ ○ 待定 │    │
│              └─────────┘           └─────────┘    │
│                                                    │
├────────────────────────────────────────────────────┤
│  当前节点：CEO审核                                  │
│  Agent 输出：方案已完成，请确认是否继续开发...      │
│                                                    │
│  [确认继续] [修改方案] [暂停]                       │
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ CEO 输入框                                  │   │
│  │ 你的决策或指令...                           │   │
│  └────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────┘
```

**可视化设计器**

- 拖拽节点创建角色环节
- 连线定义依赖关系
- 配置节点属性（角色、任务、验收标准）
- 保存为行业模板

---

## MVP Scope

**场景：软件公司开发小需求**

**验证链路：**

1. CEO 提交任务："开发一个用户登录功能"
2. TaskEngine 生成工作流 DAG
3. 工作流执行，WebUI 实时显示拓扑图
4. 到达决策点暂停，CEO 确认继续
5. Agent 团队完成开发，输出代码
6. CEO 验收确认

**MVP 范围：**

| 功能 | MVP 包含 | 后续扩展 |
|------|----------|----------|
| 角色池 | 预设 3-5 个软件公司角色（PM、TechLead、Dev、QA） | 用户自定义角色 |
| 任务引擎 | 简单任务分解，固定工作流模板 | 动态生成复杂 DAG |
| 工作流执行 | 串行 + 简单并行 | 复杂拓扑、条件分支 |
| CEO 交互 | 决策点确认、继续/暂停 | 多种交互模式 |
| WebUI | 拓扑图 + 状态监控 + 简单交互 | 完整设计器 |
| 行业模板 | 软件公司模板 | 多行业模板 |
| LLM | OpenAI | 多 Provider |

---

## Verification

**MVP 验证步骤：**

1. 启动后端服务：`go run cmd/server/main.go`
2. 打开 WebUI：`http://localhost:3000`
3. 提交测试任务："开发用户登录 API"
4. 观察拓扑图生成和实时状态
5. 在决策点进行 CEO 确认
6. 查看最终产出（代码文件）
7. 检查 JSONL 存储的完整执行记录

---

## Implementation Status (MVP Phase 1)

**已实现功能：**

| 功能 | 状态 | 说明 |
|------|------|------|
| 公司管理 | ✅ 完成 | CRUD API、公司维度数据存储 |
| 角色池 | ✅ 完成 | Role模板定义(YAML)、Pool管理、RoleInstance |
| 工作流模板 | ✅ 完成 | 固定6节点模板(workflow_template.yaml) |
| Workflow Review | ✅ 完成 | Draft状态审批、角色Prompt预览 |
| Aura SDK集成 | ✅ 完成 | Executor调用SDK执行Agent |
| 前端拓扑图 | ✅ 完成 | React Flow可视化 |
| CEO审批流程 | ✅ 完成 | Draft → Approved → Running |

**当前限制：**

- **工作流节点固定** - 从模板加载，非动态生成DAG
- **任务分解未实现** - CreateWorkflow直接调用模板，未调用LLM分解
- **验收标准未生成** - 节点没有Acceptance Criteria

**后续Phase计划：**

| Phase | 功能 | 说明 |
|-------|------|------|
| Phase 2 | 动态工作流生成 | TaskEngine调用LLM分解任务、生成DAG |
| Phase 3 | 验收标准生成 | 每个节点生成Acceptance Criteria |
| Phase 4 | 行业模板库 | 多行业工作流模板预设 |
| Phase 5 | 可视化设计器 | 拖拽设计工作流 |

---

## Aura SDK Integration (已实现)

**实际集成方式：**

```go
// backend/internal/agent/executor.go
import "github.com/oneliang/aura/core/pkg/sdk"

func (e *Executor) ExecuteInstance(ctx context.Context, r *role.Role, instance *role.Instance) (string, error) {
    // 1. 创建运行时配置
    cfg := sdk.DefaultRuntimeConfig()
    cfg.SystemPrompt = instance.GetFullPrompt(r) // Role模板 + Instance上下文
    
    // 2. 设置LLM配置
    cfg.LLM.Provider = e.config.LLM.Provider
    cfg.LLM.Model = e.config.LLM.Model
    cfg.LLM.APIKey = e.config.LLM.APIKey
    cfg.LLM.BaseURL = e.config.LLM.BaseURL
    
    // 3. 创建Runtime
    runtime, err := sdk.NewRuntime(cfg)
    
    // 4. 初始化
    runtime.Initialize(ctx)
    defer runtime.Shutdown()
    
    // 5. 处理输入 - 返回事件流
    events, err := runtime.Process(ctx, instance.Input)
    
    // 6. 收集响应
    for ev := range events {
        switch ev.Type() {
        case sdk.EventTypeResponse:
            response.WriteString(ev.Content())
        case sdk.EventTypeDone:
            break
        }
    }
}
```

**RoleInstance设计：**

```go
// Role是模板(class)，Instance是运行时实例
type Instance struct {
    ID        string    // 实例ID
    RoleID    string    // 关联Role模板
    SessionID string    // 所属Session
    StepID    string    // 关联Step
    Context   string    // 依赖步骤输出(上下文)
    Input     string    // 具体任务输入
    Output    string    // 执行输出
    Status    InstanceStatus
}

// GetFullPrompt组合Role模板和Instance上下文
func (i *Instance) GetFullPrompt(role *Role) string {
    prompt := role.SystemPrompt
    if i.Context != "" {
        prompt += "\n\n--- Context ---\n" + i.Context
    }
    if i.Input != "" {
        prompt += "\n\n--- Task ---\n" + i.Input
    }
    return prompt
}
```

**参考Aura SDK Demo：**

```
/Users/oneliang/AgentProjects/aura/examples/sdk-demo/
├── README.md          # SDK使用说明
├── main.go            # Demo入口
└── examples/          # 各类示例
    ├── basic          # 最小集成
    ├── tool           # 自定义工具
    ├── confirm        # 确认处理
    ├── stream         # 实时事件
    └── conversation   # 多轮对话
```

---

## Next Steps

1. **Phase 2: 动态工作流生成**
   - TaskEngine.DecomposeTask() - 调用LLM分解任务
   - 根据任务复杂度生成DAG节点数量
   - 动态分配角色、生成验收标准

2. **Phase 3: 验收标准**
   - 每个Node生成Acceptance Criteria
   - CEO可在Review阶段修改

3. **Phase 4+: 行业模板库 + 设计器**