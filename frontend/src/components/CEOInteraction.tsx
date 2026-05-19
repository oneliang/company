import { useState } from 'react'
import { submitDecision } from '../api/companyApi'

interface Props {
  sessionId: string
  companyId: string
}

export default function CEOInteraction({ sessionId, companyId }: Props) {
  const [stepId, setStepId] = useState('')
  const [type, setType] = useState('approve')
  const [content, setContent] = useState('')

  const handleSubmit = async () => {
    if (!stepId.trim()) {
      alert('请输入步骤ID')
      return
    }
    await submitDecision(companyId, sessionId, stepId, type, content)
    setStepId('')
    setContent('')
    alert('决策已提交')
  }

  return (
    <div style={{ padding: '20px', borderTop: '1px solid #ccc' }}>
      <h2>CEO 决策面板</h2>
      <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
        <input
          placeholder="步骤ID (如: ceo_review_design)"
          value={stepId}
          onChange={(e) => setStepId(e.target.value)}
          style={{ padding: '8px', width: '200px' }}
        />
        <select
          value={type}
          onChange={(e) => setType(e.target.value)}
          style={{ padding: '8px' }}
        >
          <option value="approve">批准</option>
          <option value="reject">驳回</option>
          <option value="redirect">重定向</option>
        </select>
        <input
          placeholder="决策意见"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          style={{ padding: '8px', width: '200px' }}
        />
        <button onClick={handleSubmit} style={{ padding: '8px 15px' }}>提交</button>
      </div>
    </div>
  )
}