import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { listCompanies, createCompany } from '../api/companyApi'

export default function CompanyList() {
  const { ceoId } = useParams<{ ceoId: string }>()
  const navigate = useNavigate()
  const [companies, setCompanies] = useState<any[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [newIndustry, setNewIndustry] = useState('software')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (ceoId) {
      setLoading(true)
      listCompanies(ceoId)
        .then(data => {
          setCompanies(data || [])
          setLoading(false)
        })
        .catch(() => setLoading(false))
    }
  }, [ceoId])

  const handleCreate = async () => {
    if (!ceoId || !newName.trim()) return
    const c = await createCompany(newName, newIndustry, ceoId, '')
    setCompanies([...companies, c])
    setShowCreate(false)
    setNewName('')
    navigate(`/companies/${c.id}`)
  }

  if (loading) return <div className="text-center py-12 text-gray-500">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-semibold text-gray-800">我的公司</h2>
        <button
          onClick={() => setShowCreate(true)}
          className="bg-primary text-white px-4 py-2 rounded-lg hover:bg-blue-600 transition-colors"
        >
          创建新公司
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {companies.map(c => (
          <div
            key={c.id}
            onClick={() => navigate(`/companies/${c.id}`)}
            className="bg-white rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow cursor-pointer border border-gray-100"
          >
            <div className="flex items-center justify-between mb-3">
              <div>
                <h3 className="text-lg font-semibold text-gray-900">{c.name}</h3>
                <p className="text-xs text-gray-500 mt-1">ID: {c.id}</p>
              </div>
              <span className="text-xs bg-blue-100 text-primary px-2 py-1 rounded">{c.industry}</span>
            </div>
            {c.description && <p className="text-gray-600 text-sm">{c.description}</p>}
            {/* Session Statistics */}
            <div className="mt-4 flex items-center gap-3">
              <span className="text-xs bg-green-100 text-success px-2 py-1 rounded flex items-center gap-1">
                ✓ {c.completed_count || 0} 已完成
              </span>
              <span className="text-xs bg-yellow-100 text-warning px-2 py-1 rounded flex items-center gap-1">
                ⏳ {c.pending_count || 0} 待处理
              </span>
            </div>
          </div>
        ))}
      </div>

      {companies.length === 0 && !showCreate && (
        <div className="text-center py-12 text-gray-500">
          <p>还没有创建任何公司</p>
          <p className="text-sm mt-2">点击上方按钮开始创建</p>
        </div>
      )}

      {showCreate && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">创建新公司</h3>
            <div className="space-y-4">
              <input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="公司名称"
                className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary"
              />
              <select
                value={newIndustry}
                onChange={(e) => setNewIndustry(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary"
              >
                <option value="software">软件公司</option>
                <option value="marketing">营销公司</option>
                <option value="consulting">咨询公司</option>
              </select>
            </div>
            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
              >
                取消
              </button>
              <button
                onClick={handleCreate}
                className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-blue-600"
              >
                创建
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}