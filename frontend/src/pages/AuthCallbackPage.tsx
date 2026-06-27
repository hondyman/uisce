// Module-level flag to prevent multiple OAuth code exchanges (works across HMR too)
let signinCallbackInProgress = false;

const AuthCallbackPage: React.FC = () => {
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Prevent multiple concurrent signin callbacks - this is the definitive fix
    if (signinCallbackInProgress) {
      devLog('[AuthCallback] Callback already in progress, skipping');
      return;
    }
    signinCallbackInProgress = true;

    const completeSignin = async () => {
      try {
        const user = await userManager.signinRedirectCallback();

        devLog('[AuthCallback] OIDC callback succeeded', {
          sub: user.profile?.sub,
          expired: user.expired,
        });

        // AuthContext listens to userManager events and will persist the token/user.
        navigate('/', { replace: true });
      } catch (err) {
        devError('[AuthCallback] OIDC callback failed', err);
        setError(err instanceof Error ? err.message : 'Authentication callback failed');
      } finally {
        signinCallbackInProgress = false;
      }
    };

    void completeSignin();
  }, [navigate]);

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="rounded-lg border border-red-200 bg-red-50 p-8 text-center dark:border-red-900 dark:bg-red-950">
          <h1 className="mb-2 text-xl font-semibold text-red-800 dark:text-red-200">
            Authentication failed
          </h1>
          <p className="text-red-700 dark:text-red-300">{error}</p>
          <a
            href="/login"
            className="mt-4 inline-block rounded-md bg-red-600 px-4 py-2 text-white hover:bg-red-700"
          >
            Back to login
          </a>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <div className="mx-auto mb-4 h-10 w-10 animate-spin rounded-full border-4 border-indigo-200 border-t-indigo-600"></div>
        <p className="text-gray-600 dark:text-gray-300">Completing sign in...</p>
      </div>
    </div>
  );
};

export default AuthCallbackPage;
