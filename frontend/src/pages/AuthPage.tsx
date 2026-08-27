import { useState } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { devLog } from '../utils/devLogger';
import { UisceLogo } from '../components/brand/UisceLogo';
import './AuthPage.css';

const AuthPage: React.FC = () => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showDirectForm, setShowDirectForm] = useState(false);
  const { login, isAuthenticated } = useAuth();
  const location = useLocation();

  const from = location.state?.from?.pathname || '/';

  if (isAuthenticated) {
    return <Navigate to={from} replace />;
  }

  const handleKeycloakSignIn = async () => {
    setError('');
    setIsLoading(true);

    try {
      devLog('Redirecting to Keycloak for authentication');
      await login();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to start sign in';
      setError(message);
      devLog('Keycloak redirect failed:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDirectSignIn = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !password) {
      setError('Please enter both email and password');
      return;
    }
    setError('');
    setIsLoading(true);

    try {
      devLog('Authenticating directly with backend');
      await login(email, password);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Invalid email or password';
      setError(message);
      devLog('Direct login failed:', err);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="auth-container">
      {/* Floating background shapes */}
      <div className="floating-shapes">
        <div className="floating-shape"></div>
        <div className="floating-shape"></div>
        <div className="floating-shape"></div>
        <div className="floating-shape"></div>
      </div>

      <div className="auth-card">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="mx-auto h-20 w-20 flex items-center justify-center mb-6 relative">
            <div className="absolute inset-0 rounded-3xl pulse-ring" style={{ background: 'rgba(212, 160, 23, 0.15)' }}></div>
            <UisceLogo variant="mark" size="lg" animated />
          </div>
          <h1 className="text-4xl font-bold auth-gradient-text mb-3">Welcome Back</h1>
          <p className="text-lg" style={{ color: '#F5F0E8', opacity: 0.7 }}>Sign in to your Uisce account</p>
        </div>

        {/* Error Message */}
        {error && (
          <div className="auth-message auth-message-error">
            <div className="font-medium">{error}</div>
          </div>
        )}

        {showDirectForm ? (
          <form onSubmit={handleDirectSignIn} className="auth-form flex flex-col gap-4">
            <div className="auth-form-group">
              <label className="auth-label">Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="user@uisce.internal"
                className="auth-input rounded-xl border border-slate-700 bg-slate-900/80 text-white w-full"
                required
              />
            </div>
            <div className="auth-form-group">
              <label className="auth-label">Password</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="auth-input rounded-xl border border-slate-700 bg-slate-900/80 text-white w-full"
                required
              />
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="auth-button w-full flex justify-center items-center rounded-xl shadow-lg text-white font-semibold disabled:opacity-50 mt-2"
            >
              {isLoading ? 'Signing in...' : 'Sign In'}
            </button>

            <button
              type="button"
              onClick={() => setShowDirectForm(false)}
              className="text-xs text-sky-400 hover:underline mt-2 text-center"
            >
              ← Back to Keycloak SSO
            </button>
          </form>
        ) : (
          <div className="flex flex-col items-center gap-3">
            <button
              type="button"
              onClick={handleKeycloakSignIn}
              disabled={isLoading}
              className="auth-button w-full flex justify-center items-center rounded-xl shadow-lg text-white font-semibold disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200"
            >
              {isLoading ? 'Connecting...' : 'Sign in with Keycloak'}
            </button>

            <button
              type="button"
              onClick={() => setShowDirectForm(true)}
              className="text-xs text-slate-400 hover:text-sky-300 transition-colors mt-2"
            >
              Use Email & Password Login
            </button>
          </div>
        )}
      </div>
    </div>
  );
};


export default AuthPage;
