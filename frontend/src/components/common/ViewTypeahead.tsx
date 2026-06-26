import React, { useState, useEffect, useRef, useMemo } from 'react';
import { devDebug } from '../../utils/devLogger';
import { Autocomplete, TextField, Box, CircularProgress, Typography } from '@mui/material';
import type { AutocompleteRenderInputParams } from '@mui/material/Autocomplete';
import { fetchViews } from '../../services/viewsService';
import { deriveViewFlags } from '../../utils/viewFlags';
import renderCoreCustomChips from './semanticChips';

// small debounce helper
const debounce = (fn: (...args: any[]) => void, wait = 300) => {
  let t: any;
  return (...args: any[]) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...args), wait);
  };
};

export type ViewOption = {
  id?: string;
  name?: string;
  title?: string;
  description?: string;
  isCore?: boolean;
  isCustom?: boolean;
  fetchKey?: string;
  extends?: string;
  tags?: any;
  metadata?: any;
  attributes?: any;
  flags?: any;
};

interface ViewTypeaheadProps {
  options: ViewOption[];
  loading?: boolean;
  inputValue?: string;
  onInputChange?: (value: string) => void;
  value?: ViewOption | string | null;
  onChange?: (value: ViewOption | string | null) => void;
  placeholder?: string;
  disabled?: boolean;
  // If true, the component will fetch options itself from /api/views
  fetchOptions?: boolean;
  // tenant/datasource scope for server fetches
  tenantId?: string;
  datasourceId?: string;
  // server-side params
  pageSize?: number;
  status?: string; // e.g., 'published'
  minQueryLength?: number;
  // If true, when the component resolves a string-valued `value` by fetching
  // the single-view endpoint, it will call `onChange(resolved)` to notify the
  // parent and allow switching from an id string to the richer object.
  resolveOnFetch?: boolean;
}

const ViewTypeahead: React.FC<ViewTypeaheadProps> = ({
  options,
  loading = false,
  inputValue = '',
  onInputChange,
  value,
  onChange,
  placeholder,
  disabled = false,
  fetchOptions = false,
  tenantId,
  datasourceId,
  pageSize = 100,
  status = 'published',
  minQueryLength = 0,
  resolveOnFetch = true,
}) => {
  const [fetchedOptions, setFetchedOptions] = useState<ViewOption[]>([]);
  const [internalLoading, setInternalLoading] = useState(false);
  const fetchIdRef = useRef(0);
  const abortRef = useRef<AbortController | null>(null);

  const normalizedOptions = useMemo<ViewOption[]>(() => {
    const source = fetchOptions ? fetchedOptions : options;
    return source.map((opt) => {
      const flags = deriveViewFlags(opt);
      const fetchKey = opt.fetchKey || opt.name || opt.title || opt.id;
      return { ...opt, ...flags, fetchKey } as ViewOption;
    });
  }, [fetchOptions, fetchedOptions, options]);

  const effectiveLoading = fetchOptions ? internalLoading : loading;

  const getOptionLabel = (opt: ViewOption | string) => {
    if (!opt) return '';
    if (typeof opt === 'string') {
      const lowered = opt.toLowerCase();
      const found = normalizedOptions.find((o) =>
        (o.id && String(o.id).toLowerCase() === lowered) ||
        (o.name && o.name.toLowerCase() === lowered) ||
        (o.title && o.title.toLowerCase() === lowered) ||
        (o.fetchKey && String(o.fetchKey).toLowerCase() === lowered)
      );
      return found ? (found.name || found.title || String(found.id ?? opt)) : opt;
    }
    // Defensive: prefer name/title, then common fallback tokens like join_path or fetchKey, then id
    try {
      const asAny = opt as any;
      return (
        asAny.name || asAny.title || asAny.join_path || asAny.fetchKey || String(asAny.id || '') || JSON.stringify(asAny)
      ).toString();
    } catch (e) {
      return String(opt as any || '');
    }
  };

  const isOptionEqualToValue = (option: ViewOption, value: ViewOption | string | null) => {
    if (!value) return false;
    // if value is a string, compare against common id-like fields
    if (typeof value === 'string') {
      const lowered = value.toLowerCase();
      return Boolean(
        (option.id && String(option.id).toLowerCase() === lowered) ||
        (option.fetchKey && String(option.fetchKey).toLowerCase() === lowered) ||
        (option.name && option.name.toLowerCase() === lowered) ||
        (option.title && option.title.toLowerCase() === lowered) ||
        ((option as any).join_path && String((option as any).join_path).toLowerCase() === lowered)
      );
    }
    // both are objects: try several identity fields
    const v: any = value as any;
    const keys = ['id', 'fetchKey', 'name', 'title', 'join_path'];
    for (const k of keys) {
      if ((option as any)[k] && v[k] && String((option as any)[k]) === String(v[k])) return true;
    }
    return false;
  };

  const findOption = (identifier: string | undefined | null) => {
    if (!identifier) return undefined;
    const lowered = identifier.toLowerCase();
    return normalizedOptions.find((o) =>
      (o.id && String(o.id).toLowerCase() === lowered) ||
      (o.name && o.name.toLowerCase() === lowered) ||
      (o.title && o.title.toLowerCase() === lowered)
    );
  };

  const displayValue = useMemo(() => {
    if (!value) return null;
    if (typeof value === 'string') {
      return findOption(value) || value;
    }
    const possibleIdentifiers = [value.id, (value as any).id, value.name, value.title]
      .map((v) => (typeof v === 'string' ? v : typeof v === 'number' ? String(v) : undefined))
      .filter((v): v is string => Boolean(v));
    for (const id of possibleIdentifiers) {
      const resolved = findOption(id);
      if (resolved) return resolved;
    }
    const flags = deriveViewFlags(value);
    return { ...(value as any), ...flags } as ViewOption;
  }, [value, normalizedOptions]);

  const selectedOption = typeof displayValue === 'string' ? findOption(displayValue) : displayValue;

  // When a value is selected (object), prefer showing its friendly label in the
  // input instead of any externally-controlled inputValue (which may be the raw
  // id/UUID). This makes a selected view show its name/title rather than the
  // UUID string.
  const effectiveInputValue = useMemo(() => {
    if (!displayValue) return inputValue;
    if (typeof displayValue === 'string') return inputValue;
    // displayValue is an object-like ViewOption
    try {
      const label = getOptionLabel(displayValue as ViewOption);
      return label ?? inputValue;
    } catch (e) {
      return inputValue;
    }
  }, [displayValue, inputValue, normalizedOptions]);

  const renderChips = (option: Partial<ViewOption> | null | undefined) => {
    return renderCoreCustomChips(option);
  };

  // internal fetch logic (debounced)
  useEffect(() => {
    if (!fetchOptions) return;
    const q = (inputValue || '').trim();
    if (q.length < (minQueryLength || 0) && q.length > 0) {
      setFetchedOptions([]);
      return;
    }

    const doFetch = async (query: string) => {
      if (!tenantId || !datasourceId) {
        setFetchedOptions([]);
        return;
      }

      const fetchId = ++fetchIdRef.current;
      abortRef.current?.abort();
      abortRef.current = new AbortController();
      setInternalLoading(true);

      try {
        const views = await fetchViews(
          { tenantId, datasourceId, pageSize, status, q: query },
          { signal: abortRef.current?.signal }
        );
        if (fetchId === fetchIdRef.current) {
          const normalized = (views || [])
            .map((v: any) => {
              if (!v) return null;
              const flags = deriveViewFlags(v);
              const fetchKey = v.fetchKey || v.name || v.title || v.id;
              return {
                id: v.id ?? v.view_id ?? v.name,
                name: v.name ?? v.title,
                title: v.title ?? v.name,
                description: v.description,
                fetchKey,
                extends: v.extends,
                tags: v.tags,
                metadata: v.metadata,
                attributes: v.attributes,
                flags: v.flags,
                ...flags,
              } as ViewOption;
            })
            .filter((v: ViewOption | null): v is ViewOption => Boolean(v));
          setFetchedOptions(normalized);
        }
      } catch (err) {
        if ((err as any)?.name === 'AbortError') return;
        if (fetchId === fetchIdRef.current) setFetchedOptions([]);
      } finally {
        if (fetchId === fetchIdRef.current) setInternalLoading(false);
      }
    };

    const debounced = debounce(doFetch, 300);
    debounced(q);

    return () => { abortRef.current?.abort(); };
  }, [fetchOptions, inputValue, tenantId, datasourceId, pageSize, status, minQueryLength]);

  // Enrich fetched options by requesting per-view details for the first N items that
  // don't include is_core/is_custom. This allows us to render Core/Custom chips even
  // when the list endpoint doesn't include those flags.
  useEffect(() => {
    if (!fetchOptions) return;
    if (!tenantId || !datasourceId) return;
    if (!fetchedOptions || fetchedOptions.length === 0) return;

    const toEnrich = fetchedOptions
      .filter((v) => typeof v.isCore === 'undefined' && typeof v.isCustom === 'undefined')
      .slice(0, 10);
    if (toEnrich.length === 0) return;

    let active = true;
    const controller = new AbortController();

    const runEnrich = async () => {
      try {
        await Promise.all(
          toEnrich.map(async (item) => {
            const key = item.fetchKey || item.id || item.name || item.title;
            if (!key) return;
            const url = `/api/views/${encodeURIComponent(key)}?tenant_id=${encodeURIComponent(
              tenantId || ''
            )}&tenant_instance_id=${encodeURIComponent(datasourceId || '')}`;
            try {
              const res = await fetch(url, { signal: controller.signal, cache: 'no-store' } as any);
              if (!res.ok) return;
              const data = await res.json().catch(() => null);
              const v = data?.view || data; // some endpoints return { view: { ... } }
              if (!v) return;
              const flags = deriveViewFlags(v);
              const nextFetchKey = v.fetchKey || v.name || v.title || v.id || key;
              if (!active) return;
              setFetchedOptions((prev) =>
                prev.map((p) => {
                  const matches =
                    (p.fetchKey && nextFetchKey && String(p.fetchKey) === String(nextFetchKey)) ||
                    (p.id && v.id && String(p.id) === String(v.id)) ||
                    (p.name && v.name && String(p.name) === String(v.name));
                  return matches
                    ? {
                        ...p,
                        ...flags,
                        fetchKey: nextFetchKey,
                        extends: v.extends ?? p.extends,
                        tags: v.tags ?? p.tags,
                        metadata: v.metadata ?? p.metadata,
                        attributes: v.attributes ?? p.attributes,
                        flags: v.flags ?? p.flags,
                      }
                    : p;
                })
              );
            } catch (e) {
              // ignore per-item errors
            }
          })
        );
      } catch (e) {
        // ignore
      }
    };

    runEnrich();

    return () => {
      active = false;
      controller.abort();
    };
  }, [fetchOptions, fetchedOptions, tenantId, datasourceId]);

  // If the provided value is a string (maybe a UUID) and we don't have it in our
  // normalizedOptions, attempt to fetch the single view record so we can show a
  // friendly name and the Core/Custom chips for the selected value.
  const resolvingValueRef = useRef<string | null>(null);
  useEffect(() => {
    if (!fetchOptions) return;
    if (!tenantId || !datasourceId) return;
    if (!value || typeof value !== 'string') return;
    const asStr = value.trim();
    if (!asStr) return;
    // already have it
    if (findOption(asStr)) {
      // already resolved in current options
      devDebug('[ViewTypeahead] value already present in options:', asStr);
      return;
    }
    if (resolvingValueRef.current === asStr) return; // already resolving

    resolvingValueRef.current = asStr;
    let active = true;
    const controller = new AbortController();

    (async () => {
      try {
        const url = `/api/views/${encodeURIComponent(asStr)}?tenant_id=${encodeURIComponent(
          tenantId || ''
        )}&tenant_instance_id=${encodeURIComponent(datasourceId || '')}`;
        devDebug('[ViewTypeahead] resolving string value by fetching single view:', asStr, url);
        const res = await fetch(url, { signal: controller.signal, cache: 'no-store' } as any);
        if (!res.ok) {
          devDebug('[ViewTypeahead] single-view fetch returned not ok for', asStr, res.status);
          resolvingValueRef.current = null;
          return;
        }
        const data = await res.json().catch(() => null);
        const v = data?.view || data;
        if (!v) {
          devDebug('[ViewTypeahead] single-view fetch returned empty body for', asStr);
          resolvingValueRef.current = null;
          return;
        }
        const flags = deriveViewFlags(v);
        const fetchKey = v.fetchKey || v.name || v.title || v.id || asStr;
        const resolved: ViewOption = {
          id: v.id ?? v.view_id ?? v.name ?? asStr,
          name: v.name ?? v.title ?? (v as any)?.display_name,
          title: v.title ?? v.name,
          description: v.description,
          fetchKey,
          extends: v.extends,
          tags: v.tags,
          metadata: v.metadata,
          attributes: v.attributes,
          flags: v.flags,
          ...flags,
        };
        if (!active) return;
  devDebug('[ViewTypeahead] resolved option:', resolved);
        setFetchedOptions((prev) => {
          // prepend resolved option unless already present
          const exists = prev.some((p) => String(p.id) === String(resolved.id) || String(p.fetchKey) === String(resolved.fetchKey));
          return exists ? prev.map((p) => (String(p.id) === String(resolved.id) ? { ...p, ...resolved } : p)) : [resolved, ...prev];
        });
        // Optionally inform parent that we resolved the UUID/string to a full
        // option so they can update controlled `value` if desired.
        if (resolveOnFetch) {
          try {
            devDebug('[ViewTypeahead] calling onChange with resolved option');
            onChange?.(resolved as any);
          } catch (e) {
            // ignore any errors from parent's handler
          }
        }
      } catch (e) {
        devDebug('[ViewTypeahead] error resolving single view', e);
        // ignore
      } finally {
        resolvingValueRef.current = null;
      }
    })();

    return () => {
      active = false;
      controller.abort();
      resolvingValueRef.current = null;
    };
  }, [value, fetchOptions, tenantId, datasourceId]);

  return (
    <Autocomplete
      options={normalizedOptions}
      loading={effectiveLoading}
  getOptionLabel={getOptionLabel as any}
  isOptionEqualToValue={isOptionEqualToValue as any}
      filterOptions={(x) => x} // server-side filtering when fetchOptions=true
  inputValue={effectiveInputValue}
  onInputChange={(_, newInput) => onInputChange?.(newInput)}
      value={(displayValue ?? null) as any}
      onChange={(_, newValue) => onChange?.(newValue)}
      disabled={disabled}
      freeSolo
      renderOption={(props, option: ViewOption) => (
        <li {...props} key={option.id || option.name || option.title}>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1,
              width: '100%',
              justifyContent: 'space-between',
            }}
          >
            <Box>
              <Typography variant="body2">{option.name || option.title}</Typography>
              {option.description ? (
                <Typography variant="caption" color="text.secondary">
                  {option.description}
                </Typography>
              ) : null}
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>{renderChips(option)}</Box>
          </Box>
        </li>
      )}
      renderInput={(params: AutocompleteRenderInputParams) => (
        <TextField
          {...params}
          size="small"
          placeholder={placeholder}
          InputProps={{
            ...params.InputProps,
            startAdornment: params.InputProps.startAdornment,
            endAdornment: (
              <>
                {renderChips(selectedOption)}
                {effectiveLoading ? <CircularProgress color="inherit" size={16} /> : null}
                {params.InputProps.endAdornment}
              </>
            ),
          }}
        />
      )}
    />
  );
};

export default ViewTypeahead;
