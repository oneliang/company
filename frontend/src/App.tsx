import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import CompanyList from './components/CompanyList'
import CompanyDetailPage from './components/CompanyDetailPage'
import SessionDetailPage from './components/SessionDetailPage'
import './index.css'

const MOCK_CEO_ID = "ceo-001"

export default function App() {
  return (
    <Router>
      <div className="min-h-screen bg-gray-50">
        <header className="bg-primary text-white py-4 px-6 shadow-md">
          <h1 className="text-2xl font-bold">Virtual Company Platform</h1>
          <p className="text-sm opacity-80">CEO Dashboard</p>
        </header>

        <main className="container mx-auto px-4 py-6">
          <Routes>
            <Route path="/" element={<Navigate to={`/ceo/${MOCK_CEO_ID}/companies`} replace />} />
            <Route path="/ceo/:ceoId/companies" element={<CompanyList />} />
            <Route path="/companies/:companyId" element={<CompanyDetailPage />} />
            <Route path="/companies/:companyId/sessions/:sessionId" element={<SessionDetailPage />} />
          </Routes>
        </main>
      </div>
    </Router>
  )
}