import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { getCompany, listSessions, createSession, getWebSocketUrl } from '../api/companyApi'
import SessionCard from './SessionCard'

export default function CompanyDetailPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const navigate = useNavigate()
  const [company, setCompany] = useState<any>(null)
  const [sessions, setSessions] = useState<any[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [newGoal, setNewGoal] = useState('')
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)

  // Check if any session is in planning status
  const hasPlanningSession = sessions.some(s => s.status === 'planning')

  // WebSocket for planning sessions
  useEffect(() => {
    if (companyId && hasPlanningSession) {
      // Connect to WebSocket to receive workflow_generated events
      const ws = new WebSocket(getWebSocketUrl(`/ws?company_id=${companyId}`))

      ws.onmessage = async (event) => {
        const data = JSON.parse(event.data)
        console.log('WebSocket event:', data)

        if (data.type === 'workflow_generated') {
          // Refresh session list to get updated status
          const updatedSessions = await listSessions(companyId)
          setSessions(updatedSessions || [])
        }
      }

      ws.onerror = (err) => console.error('WebSocket error:', err)
      wsRef.current = ws

      return () => ws.close()
    }
  }, [companyId, hasPlanningSession])

  // Backup polling for planning sessions
  useEffect(() => {
    if (companyId && hasPlanningSession && !wsRef.current) {
      const interval = setInterval(async () => {
        try {
          const updatedSessions = await listSessions(companyId)
          setSessions(updatedSessions || [])
        } catch (err) {
          console.error('Polling error:', err)
        }
      }, 3000) // Poll every 3 seconds

      return () => clearInterval(interval)
    }
  }, [companyId, hasPlanningSession])

  useEffect(() => {
    if (companyId) {
      setLoading(true)
      Promise.all([
        getCompany(companyId),
        listSessions(companyId)
      ])
        .then(([c, s]) => {
          setCompany(c)
          setSessions(s || [])
        })
        .catch(console.error)
        .finally(() => setLoading(false))
    }
  }, [companyId])

  const handleCreateSession = async () => {
    if (!companyId || !newGoal.trim()) return
    setCreating(true)
    try {
      await createSession(companyId, newGoal)
      setShowCreate(false)
      setNewGoal('')
      // Refresh list to show new session with "生成中" status
      const updatedSessions = await listSessions(companyId)
      setSessions(updatedSessions || [])
    } catch (err) {
      console.error('Create session failed:', err)
    }
    setCreating(false)
  }

  if (loading) {
    return <div className="text-center py-12 text-gray-500">加载中...</div>
  }

  if (!company) {
    return <div className="text-center py-12 text-gray-500">公司不存在</div>
  }

  return (
    <div className="space-y-6">
      {/* Company Header */}
      <div className="bg-white rounded-lg shadow-md p-6 border border-gray-100">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">{company.name}</h2>
            <div className="flex items-center gap-3 mt-2">
              <span className="text-xs bg-blue-100 text-primary px-2 py-1 rounded">{company.industry}</span>
              {company.description && <span className="text-sm text-gray-500">{company.description}</span>}
            </div>
          </div>
          <button
            onClick={() => navigate('/')}
            className="text-gray-500 hover:text-gray-700"
          >
            返回公司列表
          </button>
        </div>
      </div>

      {/* Sessions Section */}
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold text-gray-800">
          任务列表
          <span className="text-sm text-gray-500 ml-2">({sessions.length} 个会话)</span>
        </h3>
        <button
          onClick={() => setShowCreate(true)}
          className="bg-primary text-white px-4 py-2 rounded-lg hover:bg-blue-600 transition-colors"
        >
          创建新任务
        </button>
      </div>

      {/* Session List */}
      <div className="grid grid-cols-1 gap-4">
        {sessions.map(s => (
          <SessionCard key={s.id} session={s} companyId={companyId!} />
        ))}
      </div>

      {sessions.length === 0 && !showCreate && (
        <div className="text-center py-12 text-gray-500 bg-white rounded-lg border border-gray-100">
          <p>还没有创建任何任务</p>
          <p className="text-sm mt-2">点击上方按钮开始创建</p>
        </div>
      )}

      {/* Create Session Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">创建新任务</h3>
            <textarea
              value={newGoal}
              onChange={(e) => setNewGoal(e.target.value)}
              placeholder="任务目标描述..."
              rows={3}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary resize-none"
            />
            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
              >
                取消
              </button>
              <button
                onClick={handleCreateSession}
                disabled={creating}
                className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-blue-600 disabled:opacity-50"
              >
                {creating ? '创建中...' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}