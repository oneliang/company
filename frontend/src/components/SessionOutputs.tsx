import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { listSessionOutputs, getSessionOutput } from '../api/companyApi'

interface Props {
  refreshKey?: number // When this changes, refresh the file list
  workflow?: any // Workflow data to get step info (role, action)
}

// Helper: extract step ID from filename like "step-1.md"
function getStepId(filename: string): string {
  return filename.replace('step-', '').replace('.md', '')
}

// Helper: find step by ID in workflow
function findStep(workflow: any, stepId: string): any | undefined {
  if (!workflow?.steps) return undefined
  return workflow.steps.find((s: any) => s.id === stepId)
}

// Helper: format step label (role + action)
function formatStepLabel(workflow: any, filename: string): string {
  const stepId = getStepId(filename)
  const step = findStep(workflow, stepId)
  if (!step) return stepId
  // Same format as topology node: name || role: action
  return step.name || `${step.role}: ${step.action}`
}

export default function SessionOutputs({ refreshKey = 0, workflow }: Props) {
  const { companyId, sessionId } = useParams<{ companyId: string; sessionId: string }>()
  const [files, setFiles] = useState<string[]>([])
  const [selected, setSelected] = useState<string | null>(null)
  const [content, setContent] = useState<string>('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (companyId && sessionId) {
      listSessionOutputs(companyId, sessionId)
        .then(setFiles)
        .catch(() => setFiles([]))
    }
  }, [companyId, sessionId, refreshKey]) // Add refreshKey to trigger refresh

  const handleViewFile = async (filename: string) => {
    if (!companyId || !sessionId) return
    setSelected(filename)
    setLoading(true)
    try {
      const md = await getSessionOutput(companyId, sessionId, filename)
      setContent(md)
    } catch (err) {
      setContent('Failed to load file content')
    }
    setLoading(false)
  }

  if (!files || files.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6 border border-gray-100">
        <h3 className="text-lg font-semibold text-gray-800 mb-2">Agent 输出文档</h3>
        <p className="text-gray-500 text-sm">暂无输出文件，工作流执行后会生成</p>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6 border border-gray-100">
      <h3 className="text-lg font-semibold text-gray-800 mb-2">Agent 输出文档</h3>
      {/* Output directory path */}
      <div className="mb-4 p-2 bg-gray-50 rounded text-sm text-gray-600">
        <span className="font-medium">📄 输出目录: </span>
        <code className="text-gray-800">backend/data/companys/{companyId}/sessions/{sessionId}/workspace/</code>
      </div>
      <div className="flex gap-4">
        <div className="w-1/3 border-r border-gray-200 pr-3">
          <p className="text-sm text-gray-500 mb-2">点击查看详情：</p>
          {files.map(f => (
            <div key={f} className="mb-1">
              <button
                onClick={() => handleViewFile(f)}
                className={`block w-full text-left py-2 px-2 rounded text-sm ${
                  selected === f
                    ? 'bg-primary text-white'
                    : 'hover:bg-gray-100 text-gray-700'
                }`}
              >
                {formatStepLabel(workflow, f)}
              </button>
              <p className={`text-xs px-2 ${selected === f ? 'text-white/70' : 'text-gray-400'}`}>
                <code>{f}</code>
              </p>
            </div>
          ))}
        </div>
        <div className="w-2/3 min-h-[200px]">
          {loading ? (
            <p className="text-gray-500">加载中...</p>
          ) : selected ? (
            <pre className="whitespace-pre-wrap text-sm text-gray-800 font-mono bg-gray-50 p-3 rounded overflow-auto max-h-[400px]">
              {content}
            </pre>
          ) : (
            <p className="text-gray-400 text-sm">选择左侧文件查看内容</p>
          )}
        </div>
      </div>
    </div>
  )
}