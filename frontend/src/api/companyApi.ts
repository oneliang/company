import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8181/api'

// Get WebSocket URL based on current environment
export function getWebSocketUrl(path: string): string {
  const apiBase = import.meta.env.VITE_API_BASE || 'http://localhost:8181/api'
  if (apiBase.startsWith('/api')) {
    // Production: use current host (nginx proxy)
    return `ws://${window.location.host}${path}`
  } else {
    // Development: extract host from API_BASE
    const host = apiBase.replace('http://', '').replace('/api', '')
    return `ws://${host}${path}`
  }
}

// Company APIs
export async function createCompany(name: string, industry: string, ownerId: string, description?: string) {
  const response = await axios.post(`${API_BASE}/companies`, {
    name, industry, owner_id: ownerId, description
  })
  return response.data
}

export async function listCompanies(ownerId: string) {
  const response = await axios.get(`${API_BASE}/companies?owner_id=${ownerId}`)
  return response.data
}

export async function getCompany(id: string) {
  const response = await axios.get(`${API_BASE}/companies/${id}`)
  return response.data
}

export async function deleteCompany(id: string) {
  const response = await axios.delete(`${API_BASE}/companies/${id}`)
  return response.data
}

// Session APIs (company-scoped)
export async function createSession(companyId: string, goal: string) {
  const response = await axios.post(`${API_BASE}/companies/${companyId}/sessions`, { goal })
  return response.data
}

export async function listSessions(companyId: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/sessions`)
  return response.data
}

export async function getSession(companyId: string, sessionId: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/sessions/${sessionId}`)
  return response.data
}

export async function getWorkflow(companyId: string, sessionId: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/workflow`)
  return response.data
}

export async function getReview(companyId: string, sessionId: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/review`)
  return response.data
}

export async function approveWorkflow(companyId: string, sessionId: string) {
  const response = await axios.post(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/approve`)
  return response.data
}

export async function startWorkflow(companyId: string, sessionId: string) {
  // Short timeout - backend continues execution even if request times out
  const response = await axios.post(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/start`, {}, {
    timeout: 10000 // 10 seconds - backend runs async
  })
  return response.data
}

export async function resumeWorkflow(companyId: string, sessionId: string) {
  const response = await axios.post(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/resume`, {}, {
    timeout: 10000
  })
  return response.data
}

export async function getRole(companyId: string, roleId: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/roles/${roleId}`)
  return response.data
}

export async function submitDecision(companyId: string, sessionId: string, stepId: string, type: string, content: string) {
  const response = await axios.post(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/decision`, {
    step_id: stepId,
    type,
    content
  })
  return response.data
}

// Get step conversation history for CEO decision
export async function getStepHistory(companyId: string, sessionId: string, stepId: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/steps/${stepId}/history`)
  return response.data
}

// Restart a step and clear all downstream steps
export async function restartStep(companyId: string, sessionId: string, stepId: string, ceoOpinion?: string) {
  const response = await axios.post(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/steps/${stepId}/restart`, {
    ceo_opinion: ceoOpinion || ''
  })
  return response.data
}

// Role APIs
export async function listCompanyRoles(companyId: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/roles`)
  return response.data
}

// Output file APIs
export async function listSessionOutputs(companyId: string, sessionId: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/outputs`)
  return response.data
}

export async function getSessionOutput(companyId: string, sessionId: string, filename: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/outputs/${filename}`)
  return response.data
}

// 最终产物（outputs 目录）
export async function listSessionFinalOutputs(companyId: string, sessionId: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/finaloutputs`)
  return response.data
}

export async function getSessionFinalOutput(companyId: string, sessionId: string, filename: string) {
  const response = await axios.get(`${API_BASE}/companies/${companyId}/sessions/${sessionId}/finaloutputs/${filename}`)
  return response.data
}