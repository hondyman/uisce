import { useState, useEffect } from 'react';
import { useLocation, useSearchParams } from 'react-router-dom';
import useBlockableNavigate from '../components/RouteBlocker/useBlockableNavigate';
import { useAuth } from '../contexts/AuthContext';
import { devLog } from '../utils/devLogger';
import './AuthPage.css';
// Icons removed for a cleaner, brand-aligned auth experience

type AuthMode = 'login' | 'register' | 'forgot' | 'reset';

const AuthPage: React.FC = () => {
  const [mode, setMode] = useState<AuthMode>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [name, setName] = useState('');
  const [organization, setOrganization] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const { login, register, forgotPassword, resetPassword } = useAuth();
  const navigate = useBlockableNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const [showToast, setShowToast] = useState(false);

  // Get the intended destination from location state, or default to home
  const from = location.state?.from?.pathname || '/';

  // Check for reset token in URL
  useEffect(() => {
    const resetToken = searchParams.get('token');
    if (resetToken && searchParams.get('mode') === 'reset') {
      setMode('reset');
    }
    // If the user opened /signup or /register, start in register mode
    if (location.pathname === '/signup' || location.pathname === '/register') {
      setMode('register');
    }
      // Show demo toast briefly on login pages
      if (location.pathname === '/login') {
        setShowToast(true);
        const t = setTimeout(() => setShowToast(false), 4000);
        return () => clearTimeout(t);
      }
  }, [searchParams, location.pathname]);

  const validateForm = (): boolean => {
    if (!email) {
      setError('Email is required');
      return false;
    }

    if (!email.includes('@')) {
      setError('Please enter a valid email address');
      return false;
    }

    if (mode === 'forgot') {
      return true;
    }

    if (!password) {
      setError('Password is required');
      return false;
    }

    if (mode === 'register' || mode === 'reset') {
      if (password.length < 8) {
        setError('Password must be at least 8 characters long');
        return false;
      }

      if (password !== confirmPassword) {
        setError('Passwords do not match');
        return false;
      }

      if (mode === 'register' && !name.trim()) {
        setError('Full name is required');
        return false;
      }
    }

    return true;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    if (!validateForm()) {
      return;
    }

    setIsLoading(true);

    try {
      switch (mode) {
        case 'login':
          await login(email, password);
          devLog('Login successful, redirecting to:', from);
          // Give React time to update auth state before navigating
          await new Promise(resolve => setTimeout(resolve, 100));
          void navigate(from, { replace: true });
          break;

        case 'register':
          await register(email, password, name, organization);
          devLog('Registration successful, redirecting to:', from);
          // Give React time to update auth state before navigating
          await new Promise(resolve => setTimeout(resolve, 100));
          void navigate(from, { replace: true });
          break;

        case 'forgot':
          await forgotPassword(email);
          setSuccess('Password reset instructions have been sent to your email');
          setTimeout(() => setMode('login'), 3000);
          break;

        case 'reset':
          const resetToken = searchParams.get('token');
          if (!resetToken) {
            setError('Invalid reset token');
            return;
          }
          await resetPassword(resetToken, password);
          setSuccess('Password has been reset successfully');
            setTimeout(() => {
            setMode('login');
            void navigate('/login', { replace: true });
          }, 2000);
          break;
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : `${mode} failed`;
      setError(errorMessage);
      devLog(`${mode} failed:`, err);
    } finally {
      setIsLoading(false);
    }
  };

  const resetForm = () => {
    setEmail('');
    setPassword('');
    setConfirmPassword('');
    setName('');
    setOrganization('');
    setError('');
    setSuccess('');
  };

  const switchMode = (newMode: AuthMode) => {
    setMode(newMode);
    resetForm();
  };

  const getTitle = () => {
    switch (mode) {
      case 'login': return 'Welcome Back';
      case 'register': return 'Create Account';
      case 'forgot': return 'Reset Password';
      case 'reset': return 'Set New Password';
    }
  };

  const getSubtitle = () => {
    switch (mode) {
      case 'login': return 'Sign in to your SemLayer account';
      case 'register': return 'Join the semantic data platform';
      case 'forgot': return 'Enter your email to receive reset instructions';
      case 'reset': return 'Enter your new password';
    }
  };

  const getButtonText = () => {
    if (isLoading) {
      switch (mode) {
        case 'login': return 'Signing in...';
        case 'register': return 'Creating account...';
        case 'forgot': return 'Sending email...';
        case 'reset': return 'Resetting password...';
      }
    }
    
    switch (mode) {
      case 'login': return 'Sign In';
      case 'register': return 'Create Account';
      case 'forgot': return 'Send Reset Email';
      case 'reset': return 'Reset Password';
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
            <div className="mx-auto h-20 w-20 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-3xl flex items-center justify-center mb-6 relative">
              <div className="absolute inset-0 bg-indigo-600 rounded-3xl pulse-ring"></div>
              {/* Brand mark intentionally left minimal for a clean look */}
              <span className="auth-header-initial text-white font-bold">S</span>
            </div>
            <h1 className="text-4xl font-bold auth-gradient-text mb-3">
              {getTitle()}
            </h1>
            <p className="text-gray-600 text-lg">{getSubtitle()}</p>
          </div>

        {/* Back Button for non-login modes */}
        {mode !== 'login' && mode !== 'reset' && (
          <button
            onClick={() => switchMode('login')}
            className="text-gray-600 hover:text-gray-900 mb-6 transition-colors font-medium"
          >
            Back to sign in
          </button>
        )}

        {/* Success Message */}
        {success && (
          <div className="auth-message auth-message-success">
            <div className="font-medium">{success}</div>
          </div>
        )}

        {/* Error Message */}
        {error && (
          <div className="auth-message auth-message-error">
            <div className="font-medium">{error}</div>
          </div>
        )}

        <form onSubmit={handleSubmit} className="auth-form flex flex-col">
          {/* Email Field */}
          <div className="auth-form-group">
            <label htmlFor="email" className="auth-label">
              Email Address
            </label>
            <div className="auth-input-wrapper">
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                className="auth-input w-full border border-gray-300 rounded-xl text-gray-900 placeholder-gray-500 focus:outline-none focus:border-indigo-500 transition-all duration-200"
                placeholder="Enter your email address"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={isLoading}
              />
            </div>
          </div>

          {/* Name Field (Register only) */}
          {mode === 'register' && (
            <div className="auth-form-group">
              <label htmlFor="name" className="auth-label">
                Full Name
              </label>
              <div className="auth-input-wrapper">
                <input
                  id="name"
                  name="name"
                  type="text"
                  autoComplete="name"
                  required
                  className="auth-input w-full border border-gray-300 rounded-xl text-gray-900 placeholder-gray-500 focus:outline-none focus:border-indigo-500 transition-all duration-200"
                  placeholder="Enter your full name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  disabled={isLoading}
                />
              </div>
            </div>
          )}

          {/* Organization Field (Register only) */}
          {mode === 'register' && (
            <div className="auth-form-group">
              <label htmlFor="organization" className="auth-label">
                Organization <span className="text-gray-400 font-normal">(Optional)</span>
              </label>
              <div className="auth-input-wrapper">
                <input
                  id="organization"
                  name="organization"
                  type="text"
                  autoComplete="organization"
                  className="auth-input w-full border border-gray-300 rounded-xl text-gray-900 placeholder-gray-500 focus:outline-none focus:border-indigo-500 transition-all duration-200"
                  placeholder="Enter your organization"
                  value={organization}
                  onChange={(e) => setOrganization(e.target.value)}
                  disabled={isLoading}
                />
              </div>
            </div>
          )}

          {/* Password Field */}
          {mode !== 'forgot' && (
            <div className="auth-form-group">
              <label htmlFor="password" className="auth-label">
                {mode === 'reset' ? 'New Password' : 'Password'}
              </label>
              <div className="auth-input-wrapper">
                <input
                  id="password"
                  name="password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                  required
                  className="auth-input w-full border border-gray-300 rounded-xl text-gray-900 placeholder-gray-500 focus:outline-none focus:border-indigo-500 transition-all duration-200"
                  placeholder={mode === 'reset' ? 'Enter new password' : 'Enter your password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  disabled={isLoading}
                />
                <button
                  type="button"
                  className="auth-toggle-icon"
                  onClick={() => setShowPassword(!showPassword)}
                >
                  {showPassword ? 'Hide' : 'Show'}
                </button>
              </div>
              {(mode === 'register' || mode === 'reset') && (
                <p className="text-xs text-gray-500 mt-2">
                  Must be at least 8 characters long
                </p>
              )}
            </div>
          )}

          {/* Confirm Password Field */}
          {(mode === 'register' || mode === 'reset') && (
            <div className="auth-form-group">
              <label htmlFor="confirmPassword" className="auth-label">
                Confirm Password
              </label>
              <div className="auth-input-wrapper">
                <input
                  id="confirmPassword"
                  name="confirmPassword"
                  type={showConfirmPassword ? 'text' : 'password'}
                  autoComplete="new-password"
                  required
                  className="auth-input w-full border border-gray-300 rounded-xl text-gray-900 placeholder-gray-500 focus:outline-none focus:border-indigo-500 transition-all duration-200"
                  placeholder="Confirm your password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  disabled={isLoading}
                />
                <button
                  type="button"
                  className="auth-toggle-icon"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                >
                  {showConfirmPassword ? 'Hide' : 'Show'}
                </button>
              </div>
            </div>
          )}

          {/* Submit Button */}
          <button
            type="submit"
            disabled={isLoading}
            className="auth-button w-full flex justify-center items-center rounded-xl shadow-lg text-white font-semibold disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 mt-6"
          >
            {isLoading && (
              <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            )}
            {getButtonText()}
          </button>

          {/* Mode Switching Links */}
          <div className="text-center space-y-3 mt-6">
            {mode === 'login' && (
              <>
                <button
                  type="button"
                  onClick={() => switchMode('forgot')}
                  className="auth-link text-sm hover:underline"
                >
                  Forgot your password?
                </button>
                <div className="text-sm text-gray-600">
                  Don't have an account?{' '}
                  <button
                    type="button"
                    onClick={() => switchMode('register')}
                    className="auth-link font-semibold hover:underline"
                  >
                    Sign up
                  </button>
                </div>
              </>
            )}

            {mode === 'register' && (
              <div className="text-sm text-gray-600">
                Already have an account?{' '}
                <button
                  type="button"
                  onClick={() => switchMode('login')}
                  className="auth-link font-semibold hover:underline"
                >
                  Sign in
                </button>
              </div>
            )}
          </div>
        </form>

        {/* Demo toast (subtle) */}
        {showToast && (
          <div className="auth-toast">
            <div className="toast-title">Demo Environment</div>
            <div className="toast-body">Use any email/password combination to access the demo</div>
          </div>
        )}

  {/* Footer intentionally hidden on auth routes for a focused experience */}
      </div>
    </div>
  );
};

export default AuthPage;
