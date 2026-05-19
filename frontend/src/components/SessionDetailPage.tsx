import { useState, useEffect, useRef, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { getSession, getWorkflow, getReview, approveWorkflow, startWorkflow, resumeWorkflow, getRole, submitDecision, getStepHistory, restartStep } from '../api/companyApi'
import WorkflowTopology from './WorkflowTopology'
import SessionOutputs from './SessionOutputs'
import PreviousStepCard from './PreviousStepCard'

// Progress event types from backend
interface ProgressEvent {
  type: string
  step_id: string
  role: string
  action: string
  status: string
  progress: string
  error: string
  session_id: string
  step_ids?: string[] // For steps_cleared event
  reason?: string
  restarted?: string
}

// Conversation history message
interface HistoryMessage {
  role: string
  type: string
  content: string
  timestamp: number
}

export default function SessionDetailPage() {
  const { companyId, sessionId } = useParams<{ companyId: string; sessionId: string }>()
  const navigate = useNavigate()
  const [session, setSession] = useState<any>(null)
  const [workflow, setWorkflow] = useState<any>(null)
  const [review, setReview] = useState<any>(null)
  const [selectedRole, setSelectedRole] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [approving, setApproving] = useState(false)
  const [decisionContent, setDecisionContent] = useState<string>('') // Decision content for CEO
  const [stepHistory, setStepHistory] = useState<Record<string, HistoryMessage[]>>({}) // Conversation history per step
  const [expandedHistory, setExpandedHistory] = useState<Record<string, boolean>>({}) // Expanded state per step
  const [outputRefreshKey, setOutputRefreshKey] = useState<number>(0) // Key to trigger SessionOutputs refresh
  const wsRef = useRef<WebSocket | null>(null)
  const fetchedStepsRef = useRef<Set<string>>(new Set())

  // WebSocket message handler - shared between auto-connect and manual-connect
  const handleWebSocketMessage = async (event: MessageEvent) => {
    const data: ProgressEvent = JSON.parse(event.data)
    console.log('Progress event:', data)

    // workflow_generated: LLM finished generating workflow
    if (data.type === 'workflow_generated') {
      // Refresh session data
      try {
        const updated = await getSession(companyId!, sessionId!)
        setSession(updated)
        setWorkflow(updated.workflow)
      } catch (err) {
        console.error('Failed to refresh session:', err)
      }
    }

    // step_start: 收到消息后刷新后端真实状态
    if (data.type === 'step_start') {
      console.log('step_start received, step_id:', data.step_id)
      // 刷新后端真实状态（session.current_step 由后台设置）
      try {
        const updated = await getSession(companyId!, sessionId!)
        console.log('getSession (step_start) returned, current_step:', updated?.current_step, 'steps:', updated?.workflow?.steps?.map((s: any) => ({ id: s.id, status: s.status })))
        setSession(updated)
        setWorkflow(updated.workflow)
      } catch (err) {
        console.error('Failed to refresh session (step_start):', err)
      }
    } else if (data.type === 'step_complete') {
      // 触发输出文档列表刷新
      setOutputRefreshKey((prev: number) => prev + 1)
      // 刷新后端真实状态（不做本地猜测）
      try {
        const updated = await getSession(companyId!, sessionId!)
        console.log('getSession (step_complete) returned, steps:', updated?.workflow?.steps?.map((s: any) => ({ id: s.id, status: s.status })))
        setSession(updated)
        setWorkflow(updated.workflow)
      } catch (err) {
        console.error('Failed to refresh session (step_complete):', err)
      }
    } else if (data.type === 'workflow_update') {
      // 触发输出文档列表刷新
      setOutputRefreshKey((prev: number) => prev + 1)
      // 刷新 session 数据
      try {
        const updated = await getSession(companyId!, sessionId!)
        setSession(updated)
        setWorkflow(updated.workflow)
      } catch (err) {
        console.error('Failed to refresh session:', err)
      }
    } else if (data.type === 'step_failed') {
      // 刷新后端真实状态（不做本地猜测）
      try {
        const updated = await getSession(companyId!, sessionId!)
        console.log('getSession (step_failed) returned, steps:', updated?.workflow?.steps?.map((s: any) => ({ id: s.id, status: s.status })))
        setSession(updated)
        setWorkflow(updated.workflow)
      } catch (err) {
        console.error('Failed to refresh session (step_failed):', err)
      }
    } else if (data.type === 'workflow_status') {
      // 工作流状态变化 - 刷新后端真实状态（不做本地猜测）
      const newStatus = data.status
      console.log('workflow_status received, newStatus:', newStatus)
      try {
        const updated = await getSession(companyId!, sessionId!)
        console.log('getSession (workflow_status) returned, status:', updated?.status, 'steps:', updated?.workflow?.steps?.map((s: any) => ({ id: s.id, status: s.status })))
        setSession(updated)
        setWorkflow(updated.workflow)
      } catch (err) {
        console.error('Failed to refresh session (workflow_status):', err)
      }
      // 非 running/planning/paused 状态时关闭 WebSocket
      // paused = 决策点等待审批，工作流还未结束
      if (newStatus !== 'running' && newStatus !== 'planning' && newStatus !== 'paused') {
        wsRef.current?.close()
      }
    } else if (data.type === 'steps_cleared') {
      // Steps cleared due to restart - 刷新后端真实状态
      console.log('Steps cleared:', data)
      try {
        const updated = await getSession(companyId!, sessionId!)
        console.log('getSession (steps_cleared) returned')
        setSession(updated)
        setWorkflow(updated.workflow)
      } catch (err) {
        console.error('Failed to refresh session (steps_cleared):', err)
      }
    }
  }

  // Create WebSocket connection and return Promise that resolves when connected
  const connectWebSocket = (): Promise<WebSocket> => {
    return new Promise((resolve, reject) => {
      if (!sessionId) {
        reject(new Error('No sessionId'))
        return
      }

      // Close existing connection if any
      if (wsRef.current) {
        wsRef.current.close()
      }

      const ws = new WebSocket(`ws://localhost:8181/ws?session_id=${sessionId}`)

      ws.onopen = () => {
        console.log('WebSocket connected for session:', sessionId)
        wsRef.current = ws
        resolve(ws)
      }

      ws.onerror = (err) => {
        console.error('WebSocket error:', err)
        reject(err)
      }

      ws.onclose = () => {
        console.log('WebSocket closed')
        wsRef.current = null
      }

      ws.onmessage = handleWebSocketMessage
    })
  }

  // Find pending decision points (memoized to prevent unnecessary re-renders)
  const pendingDecisionPoints = useMemo(() => {
    return workflow?.steps?.filter((s: any) =>
      s.is_decision_point && s.status === 'pending'
    ) || []
  }, [workflow?.steps])

  // 当前执行步骤显示（从 session.current_step 和 workflow.steps 计算）
  const currentStepDisplay = useMemo(() => {
    if (!session?.current_step || !workflow?.steps) return ''
    const step = workflow.steps.find((s: any) => s.id === session.current_step)
    if (!step) return ''
    return `${step.id} (${step.role}: ${step.action})`
  }, [session?.current_step, workflow?.steps])

  // Load conversation history for decision points
  useEffect(() => {
    if (!companyId || !sessionId || pendingDecisionPoints.length === 0) return

    pendingDecisionPoints.forEach(async (step: any) => {
      if (!fetchedStepsRef.current.has(step.id)) {
        fetchedStepsRef.current.add(step.id)
        try {
          const history = await getStepHistory(companyId, sessionId, step.id)
          setStepHistory(prev => ({ ...prev, [step.id]: history }))
        } catch (err) {
          console.error('Failed to get history for step', step.id, err)
        }
      }
    })
  }, [companyId, sessionId, pendingDecisionPoints])

  // Handle decision - approve, continue, or restart
  const handleDecision = async (stepId: string, decisionType: string) => {
    if (!companyId || !sessionId) return
    setApproving(true)
    try {
      await submitDecision(companyId, sessionId, stepId, decisionType, decisionContent)
      // Refresh workflow status
      const updated = await getSession(companyId, sessionId)
      setSession(updated)
      setWorkflow(updated.workflow)
      setDecisionContent('')
      // Clear history cache for this step
      setStepHistory(prev => {
        const newHistory = { ...prev }
        delete newHistory[stepId]
        return newHistory
      })
      // Clear ref to allow re-fetch if step becomes pending again
      fetchedStepsRef.current.delete(stepId)
    } catch (err) {
      console.error('Decision failed:', err)
    }
    setApproving(false)
  }

  // Toggle history expansion
  const toggleHistory = (stepId: string) => {
    setExpandedHistory(prev => ({ ...prev, [stepId]: !prev[stepId] }))
  }

  // Calculate downstream steps (steps that depend on a given step)
  const getDownstreamSteps = (stepId: string): string[] => {
    if (!workflow?.steps) return []
    const result: string[] = []
    const visited = new Set<string>()

    const findDownstream = (id: string) => {
      for (const step of workflow.steps) {
        if (step.depends_on?.includes(id) && !visited.has(step.id)) {
          visited.add(step.id)
          result.push(step.id)
          findDownstream(step.id) // Recursive
        }
      }
    }

    findDownstream(stepId)
    return result
  }

  // Handle restart on previous step
  const handleRestartPreviousStep = async (stepId: string, ceoOpinion: string) => {
    if (!companyId || !sessionId) return
    setApproving(true)
    try {
      await restartStep(companyId, sessionId, stepId, ceoOpinion)
      // Refresh workflow status
      const updated = await getSession(companyId, sessionId)
      setSession(updated)
      setWorkflow(updated.workflow)
    } catch (err) {
      console.error('Restart step failed:', err)
    }
    setApproving(false)
  }

  // Handle continue on previous step
  const handleContinuePreviousStep = async (stepId: string, ceoOpinion: string) => {
    if (!companyId || !sessionId) return
    setApproving(true)
    try {
      await submitDecision(companyId, sessionId, stepId, 'continue', ceoOpinion)
      // Refresh workflow status
      const updated = await getSession(companyId, sessionId)
      setSession(updated)
      setWorkflow(updated.workflow)
    } catch (err) {
      console.error('Continue step failed:', err)
    }
    setApproving(false)
  }

  // WebSocket connection for planning status only (auto-connect)
  // For running status, we manually connect before calling API to ensure timing
  useEffect(() => {
    // Only auto-connect for planning status
    if (sessionId && session?.status === 'planning' && companyId && !wsRef.current) {
      connectWebSocket().catch(err => {
        console.error('Failed to connect WebSocket for planning:', err)
      })
    }
  }, [sessionId, session?.status, companyId])

  // Initial data load
  useEffect(() => {
    if (companyId && sessionId) {
      setLoading(true)
      Promise.all([
        getSession(companyId, sessionId),
        getWorkflow(companyId, sessionId),
        getReview(companyId, sessionId)
      ])
        .then(([s, w, r]) => {
          setSession(s)
          setWorkflow(w)
          setReview(r)
        })
        .catch(console.error)
        .finally(() => setLoading(false))
    }
  }, [companyId, sessionId])

  // Backup polling (for when WebSocket disconnects)
  useEffect(() => {
    if (session?.status === 'running' && companyId && sessionId && !wsRef.current) {
      const interval = setInterval(async () => {
        try {
          const updated = await getSession(companyId, sessionId)
          setSession(updated)
          setWorkflow(updated.workflow)

          if (updated.status !== 'running') {
            clearInterval(interval)
          }
        } catch (err) {
          console.error(err)
          clearInterval(interval)
        }
      }, 5000) // Poll every 5 seconds as backup

      return () => clearInterval(interval)
    }
  }, [session?.status, companyId, sessionId])

  const handleApprove = async () => {
    if (!companyId || !sessionId) return
    setApproving(true)
    try {
      const result = await approveWorkflow(companyId, sessionId)
      setSession({ ...session, status: result.status })
    } catch (err) {
      console.error(err)
    } finally {
      setApproving(false)
    }
  }

  const handleStartWorkflow = async () => {
    if (!companyId || !sessionId) return
    setApproving(true)
    try {
      // 先连接 WebSocket，确保能收到所有事件
      await connectWebSocket()
      console.log('WebSocket connected before API call')

      // 调用 API 启动 workflow
      await startWorkflow(companyId, sessionId)
    } catch (err) {
      console.error('Start workflow failed:', err)
    }
    try {
      const updated = await getSession(companyId, sessionId)
      setSession(updated)
      setWorkflow(updated.workflow)
    } catch (err) {
      console.error('Failed to refresh session:', err)
    }
    setApproving(false)
  }

  const handleResumeWorkflow = async () => {
    if (!companyId || !sessionId) return
    setApproving(true)
    try {
      // 先连接 WebSocket，确保能收到所有事件
      await connectWebSocket()
      console.log('WebSocket connected before API call')

      // 调用 API 恢复 workflow
      await resumeWorkflow(companyId, sessionId)
    } catch (err) {
      console.error('Resume workflow failed:', err)
    }
    try {
      const updated = await getSession(companyId, sessionId)
      setSession(updated)
      setWorkflow(updated.workflow)
    } catch (err) {
      console.error('Failed to refresh session:', err)
    }
    setApproving(false)
  }

  const handleViewRole = async (roleId: string) => {
    if (!companyId) return
    try {
      const role = await getRole(companyId, roleId)
      setSelectedRole(role)
    } catch (err) {
      console.error(err)
    }
  }

  const getStatusLabel = (status: string) => {
    const labels: Record<string, string> = {
      draft: '待审批',
      approved: '已审批',
      running: '执行中',
      completed: '已完成',
      failed: '失败',
      pending: '待处理',
      paused: '已暂停',
      planning: '生成中'
    }
    return labels[status] || status
  }

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      draft: 'bg-yellow-100 text-yellow-800',
      approved: 'bg-blue-100 text-blue-800',
      running: 'bg-green-100 text-green-800',
      completed: 'bg-green-100 text-green-800',
      failed: 'bg-red-100 text-red-800',
      paused: 'bg-orange-100 text-orange-800',
      pending: 'bg-gray-100 text-gray-800',
      planning: 'bg-purple-100 text-purple-800'
    }
    return colors[status] || 'bg-gray-100 text-gray-800'
  }

  if (loading) {
    return <div className="text-center py-12 text-gray-500">加载中...</div>
  }

  if (!session) {
    return <div className="text-center py-12 text-gray-500">会话不存在</div>
  }

  const completedCount = workflow?.steps?.filter((s: any) => s.status === 'completed').length || 0
  const totalSteps = workflow?.steps?.length || 0

  return (
    <div className="space-y-6">
      {/* Session Header */}
      <div className="bg-white rounded-lg shadow-md p-6 border border-gray-100">
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-3">
              <button
                onClick={() => navigate(`/companies/${companyId}`)}
                className="text-gray-500 hover:text-gray-700"
              >
                ← 返回任务列表
              </button>
            </div>
            <h2 className="text-xl font-bold text-gray-900 mt-3">{session.goal}</h2>
            <div className="flex items-center gap-4 mt-2 text-sm text-gray-500">
              <span className={`px-2 py-1 rounded-full ${getStatusColor(session.status)}`}>
                {getStatusLabel(session.status)}
              </span>
              <span>创建: {new Date(session.created_at).toLocaleString('zh-CN')}</span>
            </div>
            {/* Workspace directory path */}
            <div className="mt-3 p-2 bg-gray-50 rounded text-sm text-gray-600">
              <span className="font-medium">📁 工作空间目录: </span>
              <code className="text-gray-800">backend/data/companys/{companyId}/sessions/{sessionId}/</code>
            </div>
          </div>
          {session.status === 'draft' && (
            <button
              onClick={handleApprove}
              disabled={approving}
              className="bg-primary hover:bg-primary-dark text-white px-4 py-2 rounded-md disabled:opacity-50"
            >
              {approving ? '审批中...' : '审批工作流'}
            </button>
          )}
        </div>
      </div>

      {/* Planning Status - Workflow Generating */}
      {session.status === 'planning' && (
        <div className="bg-purple-50 border border-purple-200 rounded-lg p-4">
          <div className="flex items-center gap-2">
            <div className="animate-pulse w-2 h-2 bg-purple-500 rounded-full"></div>
            <p className="text-purple-800 font-medium">工作流生成中...</p>
          </div>
          <p className="text-purple-700 text-sm mt-1">
            LLM 正在根据任务描述动态生成工作流步骤，请稍候
          </p>
        </div>
      )}

      {/* Draft Status Warning */}
      {session.status === 'draft' && review && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <p className="text-yellow-800 font-medium">工作流等待审批</p>
          <p className="text-yellow-700 text-sm mt-1">请查看下方工作流详情和角色配置，确认后点击"审批工作流"开始执行</p>
        </div>
      )}

      {/* Approved Status - Start Execution Button */}
      {session.status === 'approved' && (
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <p className="text-blue-800 font-medium mb-2">工作流已审批，可以开始执行</p>
          <button
            onClick={handleStartWorkflow}
            disabled={approving}
            className="bg-primary hover:bg-primary-dark text-white px-4 py-2 rounded-md disabled:opacity-50"
          >
            {approving ? '启动中...' : '启动执行'}
          </button>
        </div>
      )}

      {/* Running Status - Real-time Progress Display */}
      {session.status === 'running' && workflow && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4">
          <div className="flex items-center gap-2">
            <div className="animate-pulse w-2 h-2 bg-green-500 rounded-full"></div>
            <p className="text-green-800 font-medium">执行中</p>
          </div>
          <p className="text-green-700 text-sm mt-1">
            进度: {completedCount}/{totalSteps} 步骤
          </p>
          {currentStepDisplay && (
            <div className="mt-2 p-2 bg-green-100 rounded text-green-800 text-sm">
              <span className="font-medium">正在执行: </span>
              {currentStepDisplay}
            </div>
          )}
        </div>
      )}

      {/* Failed/Paused Status - Resume Button */}
      {(session.status === 'failed' || session.status === 'paused') && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-800 font-medium mb-2">
            {session.status === 'failed' ? '执行失败' : '执行暂停'}
          </p>
          <p className="text-red-700 text-sm mb-3">
            {session.status === 'failed' ? '修复问题后可恢复执行' : '等待决策点审批或依赖满足后可恢复'}
          </p>
          <button
            onClick={handleResumeWorkflow}
            disabled={approving}
            className="bg-orange-500 hover:bg-orange-600 text-white px-4 py-2 rounded-md disabled:opacity-50"
          >
            {approving ? '恢复中...' : '恢复执行'}
          </button>
        </div>
      )}

      {/* Decision Point Approval */}
      {session.status === 'paused' && pendingDecisionPoints.length > 0 && (
        <div className="bg-orange-50 border border-orange-300 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-3">
            <span className="text-orange-600 text-lg">⚠️</span>
            <p className="text-orange-800 font-semibold">需要您审批决策点</p>
          </div>
          <p className="text-orange-700 text-sm mb-4">
            请查看执行过程记录后做出决策，可以追加意见继续改进或重新执行：
          </p>

          {/* Decision Points */}
          <div className="space-y-4">
            {pendingDecisionPoints.map((step: any) => (
              <div key={step.id} className="bg-white border border-orange-200 rounded-lg p-4">
                <div className="flex items-start justify-between mb-3">
                  <div>
                    <p className="font-medium text-gray-800">{step.action}</p>
                    <p className="text-sm text-gray-600 mt-1">{step.description}</p>
                    <p className="text-xs text-gray-500 mt-2">步骤: {step.id} | 角色: {step.role}</p>
                  </div>
                </div>

                {/* Conversation History */}
                {stepHistory[step.id] && stepHistory[step.id].length > 0 && (
                  <div className="mb-3 border border-gray-200 rounded-lg">
                    <button
                      onClick={() => toggleHistory(step.id)}
                      className="w-full flex items-center justify-between p-2 text-sm text-gray-700 hover:bg-gray-50"
                    >
                      <span className="font-medium">📋 执行过程记录 ({stepHistory[step.id].length} 条消息)</span>
                      <span>{expandedHistory[step.id] ? '收起' : '展开'}</span>
                    </button>
                    {expandedHistory[step.id] && (
                      <div className="p-3 max-h-60 overflow-auto bg-gray-50 space-y-2">
                        {stepHistory[step.id].map((msg, idx) => (
                          <div key={idx} className="text-sm">
                            <span className="font-medium text-gray-600">
                              [{msg.role === 'user' ? '用户' : msg.role === 'assistant' ? 'Agent' : msg.role}]
                            </span>
                            <span className="ml-2 text-gray-500 text-xs">{msg.type}</span>
                            <div className="mt-1 text-gray-700 whitespace-pre-wrap">
                              {msg.content || '(无内容)'}
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}

                {/* Previous Steps - Collapsible Cards */}
                {workflow?.steps?.filter((s: any) => s.status === 'completed').length > 0 && (
                  <div className="mb-3">
                    <p className="font-medium text-gray-700 text-sm mb-2">前置步骤检查：</p>
                    <div className="space-y-2">
                      {workflow.steps
                        .filter((s: any) => s.status === 'completed')
                        .map((s: any) => (
                          <PreviousStepCard
                            key={s.id}
                            step={s}
                            downstreamSteps={getDownstreamSteps(s.id)}
                            onRestart={handleRestartPreviousStep}
                            onContinue={handleContinuePreviousStep}
                            disabled={approving}
                          />
                        ))}
                    </div>
                  </div>
                )}

                {/* CEO Opinion Input */}
                <div className="mb-3">
                  <textarea
                    value={decisionContent}
                    onChange={(e) => setDecisionContent(e.target.value)}
                    placeholder="CEO意见（可选，用于继续执行或重新执行时补充要求）"
                    className="w-full p-2 border border-gray-200 rounded text-sm resize-none"
                    rows={3}
                  />
                </div>

                {/* Decision Buttons */}
                <div className="flex gap-2">
                  <button
                    onClick={() => handleDecision(step.id, 'approve')}
                    disabled={approving}
                    className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-md disabled:opacity-50 text-sm"
                  >
                    {approving ? '处理中...' : '批准'}
                  </button>
                  <button
                    onClick={() => handleDecision(step.id, 'continue')}
                    disabled={approving}
                    className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md disabled:opacity-50 text-sm"
                  >
                    {approving ? '处理中...' : '追加意见继续执行'}
                  </button>
                  <button
                    onClick={() => handleDecision(step.id, 'restart')}
                    disabled={approving}
                    className="bg-orange-600 hover:bg-orange-700 text-white px-4 py-2 rounded-md disabled:opacity-50 text-sm"
                  >
                    {approving ? '处理中...' : '重新执行'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Completed Status */}
      {session.status === 'completed' && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4">
          <p className="text-green-800 font-medium">执行完成</p>
          <p className="text-green-700 text-sm mt-1">所有 {totalSteps} 个步骤已完成</p>
        </div>
      )}

      {/* Workflow Review (for draft status) */}
      {review && session.status === 'draft' && (
        <div className="bg-white rounded-lg shadow-md p-6 border border-gray-100">
          <h3 className="text-lg font-semibold text-gray-800 mb-4">工作流审批详情</h3>
          <div className="space-y-4">
            {review.steps?.map((step: any) => (
              <div key={step.id} className="border border-gray-200 rounded-lg p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <span className="font-medium text-gray-800">{step.role_name}</span>
                    <span className="text-sm text-gray-500 ml-2">({step.id})</span>
                    {step.is_decision_point && (
                      <span className="ml-2 px-2 py-0.5 bg-orange-100 text-orange-800 text-xs rounded">决策点</span>
                    )}
                  </div>
                  <button
                    onClick={() => handleViewRole(step.role)}
                    className="text-sm text-primary hover:text-primary-dark"
                  >
                    查看Prompt
                  </button>
                </div>
                <p className="text-sm text-gray-600 mt-2">{step.description}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Role Prompt Modal */}
      {selectedRole && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg max-w-2xl w-full mx-4 p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">{selectedRole.name} 配置</h3>
              <button onClick={() => setSelectedRole(null)} className="text-gray-500 hover:text-gray-700">✕</button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium text-gray-700">ID</label>
                <p className="text-gray-600">{selectedRole.id}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700">描述</label>
                <p className="text-gray-600">{selectedRole.description}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700">System Prompt</label>
                <pre className="bg-gray-50 p-3 rounded text-sm text-gray-800 whitespace-pre-wrap overflow-auto max-h-60">
                  {selectedRole.system_prompt}
                </pre>
              </div>
              {selectedRole.tools_allowed?.length > 0 && (
                <div>
                  <label className="text-sm font-medium text-gray-700">允许工具</label>
                  <p className="text-gray-600">{selectedRole.tools_allowed.join(', ')}</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Workflow Visualization */}
      <div className="bg-white rounded-lg shadow-md p-6 border border-gray-100">
        <h3 className="text-lg font-semibold text-gray-800 mb-4">工作流拓扑图</h3>
        {workflow ? (
          <WorkflowTopology workflow={workflow} />
        ) : (
          <div className="text-center py-12 text-gray-500">暂无工作流数据</div>
        )}
      </div>

      {/* Agent Output Documents */}
      <SessionOutputs refreshKey={outputRefreshKey} workflow={workflow} />
    </div>
  )
}