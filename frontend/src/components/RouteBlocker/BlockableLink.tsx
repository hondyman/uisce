import React, { forwardRef, ReactNode, AnchorHTMLAttributes } from 'react';
import { Link as RouterLink, useNavigate as useRouterNavigate } from 'react-router-dom';
import { useRouteBlocker } from './RouteBlocker';

interface BlockableLinkProps extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'href'> {
  to?: string | { pathname: string; search?: string; hash?: string };
  href?: string | { pathname: string; search?: string; hash?: string };
  onBeforeNavigate?: () => Promise<boolean> | boolean;
  children?: ReactNode;
}

const BlockableLink = forwardRef<HTMLAnchorElement, BlockableLinkProps>(
  ({ to, href, onBeforeNavigate, ...rest }, ref) => {
    const rb = useRouteBlocker();
    const routerNavigate = useRouterNavigate();

    // Use 'to' if provided, otherwise fall back to 'href'
    const destination = to || href;

    const handleClick: React.MouseEventHandler<HTMLAnchorElement> = async (e) => {
      // Prevent default navigation synchronously so we can decide after async blockers
      e.preventDefault();

      // Debug: log click target and destination for easier tracing in browser
      // Intentionally avoid noisy navigation logs in production; keep behavior unchanged

      if (onBeforeNavigate) {
        const allowed = await Promise.resolve(onBeforeNavigate());
        if (!allowed) {
          return;
        }
      }

      // Build a tx (best-effort) and consult the route blocker
      type NavTx = {
        location: Location | { pathname: string };
        method: 'push' | 'replace';
        args: unknown[];
        retry: () => void;
      };

      const tx: NavTx = {
        location: typeof destination === 'string' ? new URL(destination, window.location.origin) : (destination && 'pathname' in destination ? { pathname: destination.pathname } : new URL(String(destination), window.location.origin)),
        method: 'push',
        args: [destination],
        retry: () => {},
      };

      const allowed = await rb.run(tx);
      if (!allowed) return;

      // perform navigation programmatically
      try {
        // routerNavigate supports both string and Location object
        // routerNavigate may accept a string or a Location-like object; pass through defensively
        try {
          // routerNavigate accepts string or location-like objects; guard before calling
          if (typeof destination === 'string' || (destination && 'pathname' in destination)) {
            // safe to pass through
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore-next-line - react-router navigate types are broad here
            routerNavigate(destination);
          } else if (typeof destination === 'string') {
            window.location.href = destination;
          }
        } catch (err) {
          // fallback: set window.location for absolute URLs
          if (typeof destination === 'string') window.location.href = destination;
        }
      } catch (err) {
        // fallback: set window.location for absolute URLs
        if (typeof destination === 'string') window.location.href = destination;
      }

      // Call any user-provided onClick (e.g., MenuItem's onClick to close menus)
      try {
        const userOnClick = (rest as unknown as { onClick?: (ev: React.MouseEvent<HTMLAnchorElement>) => void }).onClick;
        if (typeof userOnClick === 'function') userOnClick(e);
      } catch (e) { /* ignore */ }
    };

    // Render a RouterLink but we intercept clicks. Pass through rest props.
    return <RouterLink ref={ref} to={destination ?? ''} {...rest} onClick={handleClick} />;
  }
);

BlockableLink.displayName = 'BlockableLink';

export default BlockableLink;
