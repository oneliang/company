# Virtual Company Platform

多公司虚拟企业管理平台 - CEO Dashboard

## 项目结构

```
company/
├── backend/                 # Go 后端服务
│   ├── cmd/server/main.go   # 服务入口
│   ├── internal/            # 内部模块
│   │   ├── api/             # API handlers
│   │   ├── company/         # 公司模型
│   │   ├── session/         # Session模型
│   │   ├── role/            # 角色池
│   │   ├── workflow/        # 工作流
│   │   └── task/            # 任务引擎
│   ├── configs/             # 配置文件
│   └── data/                # 数据存储
│       └── companys/        # 公司数据
│           └── <company_id>/
│               ├── company.jsonl
│               └── sessions/
│                   └── <session_id>.jsonl
├── frontend/                # React 前端
│   ├── src/
│   │   ├── components/      # UI组件
│   │   ├── api/             # API调用
│   │   └── App.tsx          # 主应用
│   ├── package.json
│   └── vite.config.ts
├── docs/                    # 文档
├── start.sh                 # 启动脚本
└── README.md
```

## 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| 后端 API | 8181 | Go HTTP Server |
| 前端 Dev | 8100 | Vite Dev Server |

## 快速启动

### 方式一：使用启动脚本

```bash
# 检查端口并启动服务
./start.sh

# 强制杀掉占用端口并重启
./start.sh --force
```

### 方式二：手动启动

**启动后端:**
```bash
cd backend
go run cmd/server/main.go
```

**启动前端:**
```bash
cd frontend
npm install  # 第一次需要安装依赖
npm run dev
```

## API 接口

### 公司管理
- `POST /api/companies` - 创建公司
- `GET /api/companies?owner_id={id}` - 获取CEO的公司列表
- `GET /api/companies/{id}` - 获取公司详情
- `DELETE /api/companies/{id}` - 删除公司

### Session管理 (公司维度)
- `POST /api/companies/{id}/sessions` - 创建Session
- `GET /api/companies/{id}/sessions` - 获取公司的Session列表
- `GET /api/companies/{id}/sessions/{sid}` - 获取Session详情
- `GET /api/companies/{id}/sessions/{sid}/workflow` - 获取工作流
- `POST /api/companies/{id}/sessions/{sid}/decision` - 提交CEO决策

### 角色管理
- `GET /api/companies/{id}/roles` - 获取公司角色池

## 功能特性

1. **多公司管理** - CEO可以管理多个公司
2. **任务统计** - 公司卡片显示已完成/待处理任务数量
3. **工作流拓扑图** - 可视化展示任务依赖关系
4. **状态颜色编码**:
   - 🟢 绿色 = 完成
   - 🟡 黄色 = 进行中
   - 🔴 红色 = 出错
   - ⚪ 灰色 = 待处理

## 技术栈

- **后端**: Go 1.26+, Gorilla Mux, JSONL存储
- **前端**: React 18, TypeScript, Vite, TailwindCSS, React Flow, React Router
- **数据存储**: JSONL文件（按公司维度隔离）

## 测试

```bash
# 测试API
curl -s "http://localhost:8181/api/companies?owner_id=ceo-001"

# 创建公司
curl -X POST http://localhost:8181/api/companies \
  -H "Content-Type: application/json" \
  -d '{"name": "MyCompany", "industry": "software", "owner_id": "ceo-001"}'

# 创建Session
curl -X POST http://localhost:8181/api/companies/{company_id}/sessions \
  -H "Content-Type: application/json" \
  -d '{"goal": "开发登录功能"}'
```

## 访问地址

- 前端: http://localhost:8100
- 后端: http://localhost:8181