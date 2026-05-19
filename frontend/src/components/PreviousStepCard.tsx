import { useState } from 'react'
import ReactMarkdown from 'react-markdown'

interface PreviousStepCardProps {
  step: any
  downstreamSteps: string[] // IDs of steps that depend on this step
  onRestart: (stepId: string, opinion: string) => void
  onContinue: (stepId: string, opinion: string) => void
  disabled: boolean
}

export default function PreviousStepCard({ step, downstreamSteps, onRestart, onContinue, disabled }: PreviousStepCardProps) {
  const [expanded, setExpanded] = useState(false)
  const [ceoOpinion, setCeoOpinion] = useState('')
  const [showActions, setShowActions] = useState(false)

  return (
    <div className="bg-white border border-gray-200 rounded-lg">
      {/* Header - always visible */}
      <div
        className="flex items-center justify-between p-3 cursor-pointer hover:bg-gray-50"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-2">
          <span className="text-gray-400">{expanded ? '▼' : '◀'}</span>
          <span className="font-medium text-gray-800">{step.role}: {step.action}</span>
          <span className={`px-2 py-0.5 rounded text-xs ${
            step.status === 'completed' ? 'bg-green-100 text-green-800' :
            step.status === 'running' ? 'bg-yellow-100 text-yellow-800' :
            'bg-gray-100 text-gray-800'
          }`}>
            {step.status === 'completed' ? '✓ 已完成' : step.status}
          </span>
        </div>
        <button
          onClick={(e) => {
            e.stopPropagation()
            setExpanded(!expanded)
          }}
          className="text-sm text-primary hover:text-primary-dark"
        >
          {expanded ? '收起' : '查看'}
        </button>
      </div>

      {/* Expanded content */}
      {expanded && (
        <div className="p-4 border-t border-gray-100">
          {/* Input section */}
          <div className="mb-4">
            <p className="text-sm font-medium text-gray-700 mb-1">输入：</p>
            <div className="bg-gray-50 p-3 rounded text-sm text-gray-600 max-h-40 overflow-auto">
              <pre className="whitespace-pre-wrap">{step.description || '(无描述)'}</pre>
            </div>
          </div>

          {/* Output section */}
          {step.output && (
            <div className="mb-4">
              <p className="text-sm font-medium text-gray-700 mb-1">Agent 输出：</p>
              <div className="bg-gray-50 p-3 rounded text-sm text-gray-700 max-h-60 overflow-auto prose prose-sm">
                <ReactMarkdown>{step.output}</ReactMarkdown>
              </div>
            </div>
          )}

          {/* CEO Opinion input */}
          <div className="mb-3">
            <button
              onClick={() => setShowActions(!showActions)}
              className="text-sm text-primary hover:text-primary-dark"
            >
              {showActions ? '隐藏操作' : '提供CEO意见'}
            </button>
          </div>

          {/* Actions */}
          {showActions && (
            <div className="space-y-3 p-3 bg-orange-50 rounded-lg border border-orange-200">
              <textarea
                value={ceoOpinion}
                onChange={(e) => setCeoOpinion(e.target.value)}
                placeholder="CEO意见（可选）"
                className="w-full p-2 border border-gray-200 rounded text-sm resize-none"
                rows={2}
              />

              <div className="flex gap-2">
                <button
                  onClick={() => {
                    onContinue(step.id, ceoOpinion)
                    setCeoOpinion('')
                    setShowActions(false)
                  }}
                  disabled={disabled}
                  className="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1.5 rounded text-sm disabled:opacity-50"
                >
                  加意见继续
                </button>
                <button
                  onClick={() => {
                    onRestart(step.id, ceoOpinion)
                    setCeoOpinion('')
                    setShowActions(false)
                  }}
                  disabled={disabled}
                  className="bg-orange-600 hover:bg-orange-700 text-white px-3 py-1.5 rounded text-sm disabled:opacity-50"
                >
                  重新执行
                </button>
              </div>

              {/* Downstream warning */}
              {downstreamSteps.length > 0 && (
                <p className="text-xs text-orange-700">
                  ⚠️ 重新执行会同时重做以下步骤: {downstreamSteps.join(', ')}
                </p>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}