import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  Node,
  Edge,
  Position,
  MarkerType,
  Handle
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

interface WorkflowStep {
  id: string
  name?: string
  status?: string
  dependencies?: string[]
  depends_on?: string[]
  agent_role?: string
  role?: string
  action?: string
  is_decision_point?: boolean
}

interface Workflow {
  steps: WorkflowStep[]
}

interface Props {
  workflow: Workflow
}

// 状态颜色：红色=出错，绿色=完成，黄色=进行中，灰色=待处理
const getStatusColor = (status: string): string => {
  switch (status) {
    case 'completed':
      return '#22c55e' // 绿色 - 完成
    case 'running':
      return '#eab308' // 黄色 - 进行中
    case 'failed':
    case 'error':
    case 'blocked':
      return '#ef4444' // 红色 - 出错
    case 'pending':
    default:
      return '#6b7280' // 灰色 - 待处理
  }
}

const getStatusLabel = (status: string): string => {
  switch (status) {
    case 'completed':
      return '已完成'
    case 'running':
      return '进行中'
    case 'failed':
    case 'error':
    case 'blocked':
      return '出错'
    case 'pending':
    default:
      return '待处理'
  }
}

// Custom Node Component
function WorkflowNode({ data }: { data: { label: string; status: string; role?: string } }) {
  const bgColor = getStatusColor(data.status)
  const statusLabel = getStatusLabel(data.status)

  return (
    <>
      <Handle type="target" position={Position.Left} style={{ background: '#555' }} />
      <div
        style={{
          padding: '10px 14px',
          borderRadius: '10px',
          background: bgColor,
          color: '#fff',
          fontSize: '13px',
          fontWeight: 500,
          minWidth: '140px',
          textAlign: 'center',
          boxShadow: '0 3px 10px rgba(0,0,0,0.2)',
          border: '2px solid rgba(255,255,255,0.3)',
        }}
      >
        <div style={{ marginBottom: '4px' }}>{data.label}</div>
        {data.role && (
          <div style={{ fontSize: '11px', opacity: 0.85, marginBottom: '2px' }}>
            👤 {data.role}
          </div>
        )}
        <div style={{
          fontSize: '10px',
          padding: '2px 6px',
          background: 'rgba(0,0,0,0.2)',
          borderRadius: '4px',
          marginTop: '4px'
        }}>
          {statusLabel}
        </div>
      </div>
      <Handle type="source" position={Position.Right} style={{ background: '#555' }} />
    </>
  )
}

const nodeTypes = {
  workflow: WorkflowNode,
}

export default function WorkflowTopology({ workflow }: Props) {
  if (!workflow || !workflow.steps || workflow.steps.length === 0) {
    return <div className="text-center py-12 text-gray-500">暂无工作流数据</div>
  }

  const steps = workflow.steps

  // 计算节点位置 - 使用层级布局
  const nodePositions: Record<string, { x: number; y: number; level: number }> = {}

  // 先找出没有依赖的节点作为第一层
  const rootSteps = steps.filter(s => {
    const deps = s.dependencies || s.depends_on || []
    return Array.isArray(deps) && deps.length === 0
  })

  // BFS 分层
  let currentLevel = 0
  let currentNodes = rootSteps.map(s => s.id)
  const processed = new Set<string>()

  while (currentNodes.length > 0) {
    currentNodes.forEach(id => {
      if (!processed.has(id)) {
        nodePositions[id] = {
          level: currentLevel,
          x: 100 + currentLevel * 200,
          y: 50 + (processed.size % 3) * 80
        }
        processed.add(id)
      }
    })

    // 找下一层：依赖当前层的节点
    const nextNodes: string[] = []
    steps.forEach(s => {
      const deps = s.dependencies || s.depends_on || []
      if (Array.isArray(deps) && deps.some(d => currentNodes.includes(d)) && !processed.has(s.id)) {
        nextNodes.push(s.id)
      }
    })

    currentLevel++
    currentNodes = nextNodes
  }

  // 为未处理的节点分配位置
  steps.forEach((s, i) => {
    if (!nodePositions[s.id]) {
      nodePositions[s.id] = {
        level: 0,
        x: 100 + (i % 4) * 180,
        y: 50 + Math.floor(i / 4) * 100
      }
    }
  })

  const nodes: Node[] = steps.map((step) => ({
    id: step.id,
    type: 'workflow',
    position: { x: nodePositions[step.id].x, y: nodePositions[step.id].y },
    data: {
      label: step.name || `${step.role}: ${step.action}`,
      status: step.status || 'pending',
      role: step.agent_role || step.role
    },
    width: 140,
    height: 60,
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
  }))

  const edges: Edge[] = steps
    .filter(step => {
      const deps = step.dependencies || step.depends_on || []
      return Array.isArray(deps) && deps.length > 0
    })
    .flatMap(step => {
      const deps = step.dependencies || step.depends_on || []
      if (!Array.isArray(deps)) return []
      return deps.map(dep => ({
        id: `${dep}-${step.id}`,
        source: dep,
        target: step.id,
        animated: step.status === 'running',
        style: {
          stroke: step.status === 'failed' ? '#ef4444' : '#6b7280',
          strokeWidth: 2,
        },
        markerEnd: { type: MarkerType.ArrowClosed, color: '#6b7280' },
        label: step.status === 'running' ? '→' : undefined,
        labelStyle: { fill: '#6b7280', fontWeight: 700 },
      }))
    })

  return (
    <div style={{ width: '100%', height: '400px', background: '#fafafa' }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        attributionPosition="bottom-left"
        defaultEdgeOptions={{
          type: 'smoothstep',
        }}
        panOnDrag={true}
        zoomOnScroll={true}
      >
        <Background color="#e5e7eb" gap={20} />
        <Controls style={{ marginBottom: 10 }} />
        <MiniMap
          nodeColor={(node: any) => {
            // 从原始 steps 数据中查找状态
            const step = steps.find(s => s.id === node.id)
            const status = step?.status || 'pending'
            return getStatusColor(status)
          }}
          nodeStrokeWidth={3}
          nodeStrokeColor="#fff"
          style={{
            background: '#f3f4f6',
            borderRadius: '8px',
            width: 150,
            height: 100
          }}
          position="top-right"
          maskColor="rgba(0,0,0,0.1)"
        />
      </ReactFlow>
    </div>
  )
}