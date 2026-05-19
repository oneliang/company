import { useNavigate } from 'react-router-dom'

interface Session {
  id: string
  company_id: string
  goal: string
  status: string
  created_at: string
}

interface Props {
  session: Session
  companyId: string
}

const statusColors: Record<string, string> = {
  pending: 'bg-warning text-yellow-900',
  planning: 'bg-purple-100 text-purple-800',
  draft: 'bg-yellow-100 text-yellow-800',
  running: 'bg-blue-100 text-primary',
  completed: 'bg-green-100 text-success',
  failed: 'bg-red-100 text-error',
}

const statusLabels: Record<string, string> = {
  pending: '待处理',
  planning: '生成中',
  draft: '待审批',
  running: '进行中',
  completed: '已完成',
  failed: '失败',
}

export default function SessionCard({ session, companyId }: Props) {
  const navigate = useNavigate()

  // Planning status: show generating animation
  const isPlanning = session.status === 'planning'

  return (
    <div
      onClick={() => !isPlanning && navigate(`/companies/${companyId}/sessions/${session.id}`)}
      className={`bg-white rounded-lg shadow-md p-5 transition-shadow border border-gray-100 ${
        isPlanning ? 'cursor-default' : 'hover:shadow-lg cursor-pointer'
      }`}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <h3 className="text-base font-medium text-gray-900 mb-2">{session.goal}</h3>
          <p className="text-sm text-gray-500">
            创建时间: {new Date(session.created_at).toLocaleString('zh-CN')}
          </p>
        </div>
        <span className={`px-3 py-1 rounded-full text-xs font-medium ${statusColors[session.status] || 'bg-gray-100 text-gray-600'}`}>
          {isPlanning && <span className="animate-pulse mr-1">●</span>}
          {statusLabels[session.status] || session.status}
        </span>
      </div>

      {/* Planning hint */}
      {isPlanning && (
        <div className="mt-3 text-xs text-purple-600">
          LLM 正在生成工作流步骤，请稍候...
        </div>
      )}

      <div className="mt-3 flex items-center text-xs text-gray-400">
        <span>Session ID: {session.id}</span>
      </div>
    </div>
  )
}