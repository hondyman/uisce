import { useState, useEffect } from 'react';
import { useLocation, useSearchParams } from 'react-router-dom';
import useBlockableNavigate from '../components/RouteBlocker/useBlockableNavigate';
import { useAuth } from '../contexts/AuthContext';
import { devLog } from '../utils/devLogger';
import './AuthPage.css';
import { 
  
  Lock as _Lock, 
  User as _User, 
  Building2 as _Building2, 
  ArrowLeft as _ArrowLeft,
  CheckCircle as _CheckCircle,
  AlertCircle as _AlertCircle,
  Shield,
  Sparkles
} from 'lucide-react';

type AuthMode = 'login' | 'register' | 'forgot' | 'reset';

const AuthPage: React.FC = () => {
  const [mode, setMode] = useState<AuthMode>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [name, setName] = useState('');
  const [organization, setOrganization] = useState('');
  const [_showPassword, _setShowPassword] = useState(false);
  const [_showConfirmPassword, _setShowConfirmPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const { login, register, forgotPassword, resetPassword } = useAuth();
  const navigate = useBlockableNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();

  // Get the intended destination from location state, or default to home
  const from = location.state?.from?.pathname || '/';

  // Check for reset token in URL
  useEffect(() => {
    const resetToken = searchParams.get('token');
    if (resetToken && searchParams.get('mode') === 'reset') {
      setMode('reset');
    }
  }, [searchParams]);

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
          void navigate(from, { replace: true });
          break;

        case 'register':
          await register(email, password, name, organization);
          devLog('Registration successful, redirecting to:', from);
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
    <div className="auth-container min-h-screen bg-gradient-to-br from-blue-50 via-white to-indigo-50 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8 relative">
      {/* Floating background shapes */}
      <div className="floating-shapes">
        <div className="floating-shape w-64 h-64 bg-indigo-200 rounded-full"></div>
        <div className="floating-shape w-48 h-48 bg-blue-200 rounded-full"></div>
        <div className="floating-shape w-32 h-32 bg-purple-200 rounded-full"></div>
      </div>

      <div className="max-w-md w-full relative z-10">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="mx-auto h-16 w-16 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-2xl flex items-center justify-center mb-4 relative">
            <div className="absolute inset-0 bg-indigo-600 rounded-2xl pulse-ring"></div>
            <Shield className="h-8 w-8 text-white relative z-10" />
            <Sparkles className="h-4 w-4 text-indigo-200 absolute top-1 right-1" />
          </div>
          <h2 className="text-4xl font-bold bg-gradient-to-r from-gray-900 to-gray-600 bg-clip-text text-transparent mb-2">
            {getTitle()}
          </h2>
          <p className="text-gray-600">{getSubtitle()}</p>
        </div>

        {/* Main Card */}
        <div className="auth-card bg-white shadow-2xl rounded-3xl border border-gray-100 p-8 relative overflow-hidden">
          {/* Subtle gradient overlay */}
          <div className="absolute inset-0 bg-gradient-to-br from-indigo-50/50 to-purple-50/50 rounded-3xl pointer-events-none"></div>
          
          <div className="relative z-10">
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && <div className="text-sm text-red-600">{error}</div>}
              {success && <div className="text-sm text-green-600">{success}</div>}

              <div>
                <label className="block text-sm font-medium text-gray-700">Email</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="mt-1 block w-full rounded-md border-gray-200 shadow-sm"
                  placeholder="you@company.com"
                  aria-label="Email address"
                  title="Email address"
                />
              </div>

              {mode !== 'forgot' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700">Password</label>
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="mt-1 block w-full rounded-md border-gray-200 shadow-sm"
                    placeholder="Enter password"
                    aria-label="Password"
                    title="Password"
                  />
                </div>
              )}

              {(mode === 'register' || mode === 'reset') && (
                <div>
                  <label className="block text-sm font-medium text-gray-700">Confirm Password</label>
                  <input
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    className="mt-1 block w-full rounded-md border-gray-200 shadow-sm"
                    placeholder="Confirm password"
                    aria-label="Confirm password"
                    title="Confirm password"
                  />
                </div>
              )}

              {mode === 'register' && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700">Full name</label>
                    <input
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      className="mt-1 block w-full rounded-md border-gray-200 shadow-sm"
                      placeholder="Full name"
                      aria-label="Full name"
                      title="Full name"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700">Organization</label>
                    <input
                      value={organization}
                      onChange={(e) => setOrganization(e.target.value)}
                      className="mt-1 block w-full rounded-md border-gray-200 shadow-sm"
                      placeholder="Organization"
                      aria-label="Organization"
                      title="Organization"
                    />
                  </div>
                </>
              )}

              <div className="flex items-center justify-between">
                <button
                  type="submit"
                  disabled={isLoading}
                  className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700"
                >
                  {getButtonText()}
                </button>

                <div className="text-sm">
                  {mode !== 'login' ? (
                    <button type="button" onClick={() => switchMode('login')} className="text-indigo-600 hover:text-indigo-500">
                      Back to sign in
                    </button>
                  ) : (
                    <button type="button" onClick={() => switchMode('forgot')} className="text-indigo-600 hover:text-indigo-500">
                      Forgot password?
                    </button>
                  )}
                </div>
              </div>
            </form>
          </div>
        </div>

        {/* Footer */}
        <div className="mt-8 text-center">
          <p className="text-xs text-gray-500">
            By signing in, you agree to our{' '}
            <a href="#" className="text-indigo-600 hover:text-indigo-500 transition-colors">
              Terms of Service
            </a>{' '}
            and{' '}
            <a href="#" className="text-indigo-600 hover:text-indigo-500 transition-colors">
              Privacy Policy
            </a>
          </p>
        </div>
      </div>
    </div>
  );
};

export default AuthPage;
