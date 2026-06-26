import React, { useMemo } from 'react';
import { Typography, Tooltip } from '@mui/material';

// Lightweight available source shape to avoid tight coupling
type SimpleSource = { id: string | number; name: string };

interface ItemPathProps {
  id?: string;
  // optional list of sources to resolve tokens to friendly names
  availableSources?: SimpleSource[];
  // fallback source name to use for the first token if available
  sourceName?: string;
  noWrap?: boolean;
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const makeSourceMaps = (sources?: SimpleSource[]) => {
  const byId = new Map<string, string>();
  const byNameToken = new Map<string, string>();
  if (!sources) return { byId, byNameToken };
  for (const s of sources) {
    try {
      const idStr = String(s.id || '').toLowerCase();
      if (idStr) byId.set(idStr, s.name);
      const name = String(s.name || '').toLowerCase();
      if (name) {
        // split name into tokens for fuzzy matching
        name.split(/[^a-z0-9]+/i).filter(Boolean).forEach(tok => byNameToken.set(tok, s.name));
      }
    } catch (e) {
      // ignore
    }
  }
  return { byId, byNameToken };
};

const resolveToken = (token: string, maps: ReturnType<typeof makeSourceMaps>, fallbackName?: string) => {
  if (!token) return token;
  const lower = token.toLowerCase();
  // exact id match
  if (maps.byId.has(lower)) return maps.byId.get(lower) as string;
  // UUID-looking token: try fallback name or short uuid
  if (UUID_RE.test(token)) {
    if (fallbackName) return fallbackName;
    return token.slice(0, 8);
  }
  // fuzzy match by name token
  for (const [tok, name] of maps.byNameToken.entries()) {
    if (lower.includes(tok) || tok.includes(lower)) return name;
  }
  // as a final attempt, if fallbackName provided, use it for first token
  if (fallbackName) return fallbackName;
  return token;
};

const ItemPath: React.FC<ItemPathProps> = ({ id, availableSources, sourceName, noWrap }) => {
  const maps = useMemo(() => makeSourceMaps(availableSources), [availableSources]);

  const { display, rawShown } = useMemo(() => {
    if (!id) return { display: '-', rawShown: false };
    const tokens = id.includes('.') ? id.split('.') : (id.includes('::') ? id.split('::') : [id]);
    const resolved = tokens.map((t, i) => resolveToken(t, maps, i === 0 ? sourceName : undefined));
    const d = resolved.join('.');
    // show raw when any token was transformed
    const rawShownFlag = d !== id;
    return { display: d, rawShown: rawShownFlag };
  }, [id, maps, sourceName]);

  if (!id) return <Typography variant="caption" color="text.secondary">-</Typography>;

  return (
    <Tooltip title={rawShown ? id : ''} disableHoverListener={!rawShown}>
      <Typography variant="caption" color="text.secondary" noWrap={!!noWrap}>
        {display}
      </Typography>
    </Tooltip>
  );
};

export default ItemPath;
