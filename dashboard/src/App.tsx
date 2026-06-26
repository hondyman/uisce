import React, { useEffect, useState } from 'react'
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { DashboardHome } from './pages/DashboardHome'
import { ChainsList } from './pages/ChainsList'
import { ChainDetail } from './pages/ChainDetail'
import { LiveFeed } from './pages/LiveFeed'
import { ReportsPage } from './pages/ReportsPage'
import { PredictionsPage } from './pages/PredictionsPage'
import { LoginPage } from './pages/LoginPage'

// Auth context
export const AuthContext = React.createContext<{
  isAuthenticated: boolean
  tenantId: string | null
  login: (tenantId: string, token: string) => void
  logout: () => void
}>({
  isAuthenticated: false,
  tenantId: null,
  login: () => {},
  logout: () => {}
})

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = React.useContext(AuthContext)
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

export function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(() => {
    return localStorage.getItem('auth_token') !== null
  })
  const [tenantId, setTenantId] = useState(() => {
    return localStorage.getItem('tenant_id')
  })

  const login = (tid: string, token: string) => {
    localStorage.setItem('tenant_id', tid)
    localStorage.setItem('auth_token', token)
    setTenantId(tid)
    setIsAuthenticated(true)
  }

  const logout = () => {
    localStorage.removeItem('tenant_id')
    localStorage.removeItem('auth_token')
    setTenantId(null)
    setIsAuthenticated(false)
  }

  return (
    <AuthContext.Provider value={{ isAuthenticated, tenantId, login, logout }}>
      <Router>
        <Routes>
          <Route 
            path="/login" 
            element={<LoginPage />} 
          />
          <Route 
            path="/" 
            element={
              <ProtectedRoute>
                <DashboardHome />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="/chains" 
            element={
              <ProtectedRoute>
                <ChainsList />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="/chains/:chainId" 
            element={
              <ProtectedRoute>
                <ChainDetail />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="/feed" 
            element={
              <ProtectedRoute>
                <LiveFeed />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="/predictions"
            element={
              <ProtectedRoute>
                <PredictionsPage />
              </ProtectedRoute>
            }
          />
          <Route 
            path="/reports" 
            element={
              <ProtectedRoute>
                <ReportsPage />
              </ProtectedRoute>
            } 
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Router>
    </AuthContext.Provider>
  )
}
