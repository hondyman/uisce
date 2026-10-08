import { useEffect, useState } from 'react';
import { RouteAliasApi, type RouteAlias } from '../api/navigationMenu';

/**
 * Route aliases come from the database (route_aliases), not code: each maps an
 * app URL to the Page Designer page that serves it. `loaded` is true once the
 * request settles, success or failure, so routing is never blocked on a failed fetch.
 */
export function useRouteAliases(): { aliases: RouteAlias[]; loaded: boolean } {
  const [aliases, setAliases] = useState<RouteAlias[]>([]);
  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    let live = true;
    RouteAliasApi.list()
      .then((a) => live && setAliases(Array.isArray(a) ? a : []))
      .catch(() => undefined)
      .finally(() => live && setLoaded(true));
    return () => {
      live = false;
    };
  }, []);
  return { aliases, loaded };
}
