import { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { devLog } from '../utils/devLogger';
import './AuthPage.css';

const AuthPage: React.FC = () => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const { login } = useAuth();

  const handleSignIn = async () => {
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
      // The redirect happens synchronously; this is mostly for error paths.
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
          <div className="mx-auto h-20 w-20 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-3xl flex items-center justify-center mb-6 relative">
            <div className="absolute inset-0 bg-indigo-600 rounded-3xl pulse-ring"></div>
            <span className="auth-header-initial text-white font-bold">S</span>
          </div>
          <h1 className="text-4xl font-bold auth-gradient-text mb-3">Welcome Back</h1>
          <p className="text-gray-600 text-lg">Sign in to your SemLayer account</p>
        </div>

        {/* Error Message */}
        {error && (
          <div className="auth-message auth-message-error">
            <div className="font-medium">{error}</div>
          </div>
        )}

        <div className="flex flex-col items-center">
          <button
            type="button"
            onClick={handleSignIn}
            disabled={isLoading}
            className="auth-button w-full flex justify-center items-center rounded-xl shadow-lg text-white font-semibold disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200"
          >
            {isLoading && (
              <svg
                className="animate-spin -ml-1 mr-3 h-5 w-5 text-white"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                ></circle>
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                ></path>
              </svg>
            )}
            Sign in with Keycloak
          </button>
        </div>
      </div>
    </div>
  );
};

export default AuthPage;
