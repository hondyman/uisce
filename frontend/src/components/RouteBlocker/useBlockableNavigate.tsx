import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useRouteBlocker } from './RouteBlocker';

type NavigateOptions = { replace?: boolean; state?: any };

export const useBlockableNavigate = () => {
  const navigate = useNavigate();
  const { run } = useRouteBlocker();

  const blockableNavigate = useCallback(
    async (to: string | number, options?: NavigateOptions) => {
      // Build a tx representing the intended navigation
      const tx = {
        location: typeof to === 'string' ? new URL(to, window.location.origin) : undefined,
        method: options?.replace ? 'replace' : 'push',
        args: [to, options],
        // retry will perform the actual navigation
        retry: () => {
          if (typeof to === 'number') {
            // a delta navigation
            window.history.go(to);
          } else {
            navigate(to, options as any);
          }
        },
      } as any;

      const allowed = await run(tx);
      if (allowed) {
        if (typeof to === 'number') {
          window.history.go(to);
        } else {
          navigate(to, options as any);
        }
      }
      return allowed;
    },
    [navigate, run]
  );

  return blockableNavigate;
};

export default useBlockableNavigate;
