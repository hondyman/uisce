import React, { useState, useContext } from 'react'
import { useNavigate } from 'react-router-dom'
import { AuthContext } from '../App'

export function LoginPage() {
  const navigate = useNavigate()
  const { login } = useContext(AuthContext)
  const [formData, setFormData] = useState({
    tenantId: '',
    token: ''
  })
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setFormData(prev => ({ ...prev, [name]: value }))
    setError(null)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!formData.tenantId || !formData.token) {
      setError('Please enter both Tenant ID and API Token')
      return
    }

    setLoading(true)
    try {
      // Simulate token validation
      if (formData.token.startsWith('test_')) {
        login(formData.tenantId, formData.token)
        navigate('/')
      } else {
        setError('Invalid API token. Use a token starting with "test_" for demo purposes.')
      }
    } finally {
      setLoading(false)
    }
  }

  const handleDemoLogin = () => {
    login('demo-tenant', 'test_demo_token_12345')
    navigate('/')
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-brand via-brand-light to-slate-900 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Logo / Header */}
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">SemLayer</h1>
          <p className="text-slate-300">Operational Intelligence Platform</p>
        </div>

        {/* Login Card */}
        <div className="bg-white rounded-2xl shadow-2xl p-8 space-y-6">
          <h2 className="text-2xl font-bold text-slate-900 mb-6">Sign In</h2>

          {error && (
            <div className="p-4 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                Tenant ID
              </label>
              <input
                type="text"
                name="tenantId"
                value={formData.tenantId}
                onChange={handleChange}
                placeholder="e.g., demo-tenant"
                className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={loading}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                API Token
              </label>
              <input
                type="password"
                name="token"
                value={formData.token}
                onChange={handleChange}
                placeholder="e.g., test_..."
                className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={loading}
              />
              <p className="text-xs text-slate-600 mt-2">
                For demo: use any tenant ID and token starting with "test_"
              </p>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:bg-slate-400 transition-colors"
            >
              {loading ? 'Signing In...' : 'Sign In'}
            </button>
          </form>

          {/* Divider */}
          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-slate-300"></div>
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="px-2 bg-white text-slate-600">Or</span>
            </div>
          </div>

          {/* Demo Login */}
          <button
            onClick={handleDemoLogin}
            className="w-full px-4 py-2 border border-slate-300 text-slate-700 rounded-lg font-medium hover:bg-slate-50 transition-colors"
          >
            Try Demo Account
          </button>
        </div>

        {/* Footer */}
        <div className="mt-8 text-center">
          <p className="text-slate-300 text-sm">
            Backend: <code className="text-blue-300">http://localhost:8080</code>
          </p>
          <p className="text-slate-300 text-sm mt-1">
            WebSocket: <code className="text-blue-300">ws://localhost:8081</code>
          </p>
          <p className="text-slate-400 text-xs mt-4">
            Phase 3.16: React Dashboard • Demo Data • Not for Production
          </p>
        </div>
      </div>
    </div>
  )
}
