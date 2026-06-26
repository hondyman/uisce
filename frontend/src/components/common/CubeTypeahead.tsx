import { useEffect, useMemo, useState } from 'react';
import Autocomplete from '@mui/material/Autocomplete';
import TextField from '@mui/material/TextField';
import CircularProgress from '@mui/material/CircularProgress';
// Chip rendering centralized in semanticChips
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { fetchCubes } from '../../services/cubesService';
import renderCoreCustomChips from './semanticChips';

export type CubeOption = {
  id: string;
  name: string;
  display_name?: string;
  description?: string;
  isCore?: boolean;
  isCustom?: boolean;
};

type Props = {
  options?: CubeOption[];
  fetchOptions?: boolean; // if true, component fetches options itself
  tenantId?: string;
  datasourceId?: string;
  pageSize?: number;
  placeholder?: string;
  value?: string | CubeOption | null;
  onChange?: (ev: any, value: CubeOption | null) => void;
  // If true, when a string id is provided as the value and we successfully
  // fetch the full cube object, call onChange with the resolved object so the
  // parent can switch to the richer representation (and chips/friendly name
  // will display). Default false to avoid surprising parent shape.
  resolveOnFetch?: boolean;
  inputValue?: string;
  onInputChange?: (ev: any, val: string) => void;
  disabled?: boolean;
  minQueryLength?: number;
};

export default function CubeTypeahead({
  options = [],
  fetchOptions = false,
  tenantId,
  datasourceId,
  pageSize = 100,
  placeholder = 'Search cubes',
  value = null,
  onChange,
  inputValue,
  onInputChange,
  disabled = false,
  // minQueryLength unused for now
  resolveOnFetch = true,
  minQueryLength = 0,
}: Props) {
  const [internalOptions, setInternalOptions] = useState<CubeOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [enrichedMap, setEnrichedMap] = useState<Record<string, Partial<CubeOption>>>({});
  // small debounce helper
  const debounce = (fn: (...args: any[]) => void, wait = 300) => {
    let t: any;
    return (...args: any[]) => {
      clearTimeout(t);
      t = setTimeout(() => fn(...args), wait);
    };
  };

  useEffect(() => {
    if (!fetchOptions) return;
    const q = (inputValue || '').trim();
    if (q.length < (minQueryLength || 0) && q.length > 0) {
      setInternalOptions([]);
      return;
    }

    let active = true;
    const controller = new AbortController();

    const doFetch = async (query: string) => {
      setLoading(true);
      try {
        const cubes = await fetchCubes({ tenantId, datasourceId, pageSize, q: query }, { signal: controller.signal });
        if (!active) return;
        // normalize
        const items = Array.isArray(cubes)
          ? cubes.map((m: any) => ({
              id: m.id,
              name: m.model_key || m.id,
              display_name: m.display_name || m.model_key,
              description: m.description,
              isCore: Boolean(m.is_core),
              isCustom: Boolean(m.is_custom),
            }))
          : [];
        setInternalOptions(items);
      } catch (e) {
        if (active) setInternalOptions([]);
      } finally {
        if (active) setLoading(false);
      }
    };

    const debounced = debounce(doFetch, 300);
    debounced(q);

    return () => {
      active = false;
      controller.abort();
    };
  }, [fetchOptions, inputValue, tenantId, datasourceId, pageSize, minQueryLength]);

  const mergedOptions = useMemo(() => {
    if (fetchOptions) return internalOptions;
    return options.map((o) => ({ ...(o as any), ...(enrichedMap[String(o.id)] || {}) }));
  }, [fetchOptions, internalOptions, options, enrichedMap]);

  // If the parent passed a string id as `value` and we have fetchOptions=true,
  // try to resolve it by querying the model definition endpoint so we can show
  // friendly labels and chips. If resolveOnFetch is true, notify parent via
  // onChange with the enriched object.
  useEffect(() => {
    if (!fetchOptions) return;
    if (!resolveOnFetch) return;
    if (!tenantId || !datasourceId) return;
  if (!value) return;
  const asStr = typeof value === 'string' ? value.trim() : String((value as any).id || '');
    if (!asStr) return;
    const exists = mergedOptions.some((o) => String(o.id) === String(asStr));
    if (exists) return;

    let active = true;
    const controller = new AbortController();
    (async () => {
      try {
        // If the value looks like a UUID (frontend sometimes supplies the model id),
        // fetch the models list for the datasource and resolve by id. The
        // /api/fabric/models/definition endpoint expects a model_key path (e.g. '/public/foo'),
        // so sending a UUID to it causes a 404.
        const uuidLike = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
        if (uuidLike.test(asStr)) {
          // List models for the datasource and find by id
          const listUrl = `/api/fabric/models?tenant_instance_id=${encodeURIComponent(datasourceId || '')}`;
          const listRes = await fetch(listUrl, { signal: controller.signal, cache: 'no-store' } as any);
          if (!listRes.ok) return;
          const listData = await listRes.json().catch(() => null);
          const modelsArray = listData?.models || listData || [];
          const model = Array.isArray(modelsArray) ? modelsArray.find((m: any) => String(m.id) === String(asStr)) : null;
          if (!model) return;
          const resolved: CubeOption = {
            id: model.id || asStr,
            name: model.model_key || model.id || asStr,
            display_name: model.display_name || model.model_key || model.id,
            description: model.description,
            isCore: Boolean(model.is_core),
            isCustom: Boolean(model.is_custom),
          };
          if (!active) return;
          setInternalOptions((prev) => {
            const found = prev.some((p) => String(p.id) === String(resolved.id));
            return found ? prev.map((p) => (String(p.id) === String(resolved.id) ? { ...p, ...resolved } : p)) : [resolved, ...prev];
          });
          if (onChange) {
            try { onChange?.(null, resolved); } catch (e) { /* ignore */ }
          }
          return;
        }

        // Fallback: backend provides a definition endpoint for models (expects model_key path)
        const url = `/api/fabric/models/definition?tenant_instance_id=${encodeURIComponent(datasourceId || '')}&model_key=${encodeURIComponent(asStr)}`;
        const res = await fetch(url, { signal: controller.signal, cache: 'no-store' } as any);
        if (!res.ok) return;
        const data = await res.json().catch(() => null);
        const model = data?.model || data;
        if (!model) return;
        const resolved: CubeOption = {
          id: model.id || asStr,
          name: model.model_key || model.id || asStr,
          display_name: model.display_name || model.model_key || model.id,
          description: model.description,
          isCore: Boolean(model.is_core),
          isCustom: Boolean(model.is_custom),
        };
        if (!active) return;
        setInternalOptions((prev) => {
          const found = prev.some((p) => String(p.id) === String(resolved.id));
          return found ? prev.map((p) => (String(p.id) === String(resolved.id) ? { ...p, ...resolved } : p)) : [resolved, ...prev];
        });
        if (onChange) {
          try { onChange?.(null, resolved); } catch (e) { /* ignore */ }
        }
      } catch (e) {
        // ignore
      }
    })();

    return () => { active = false; controller.abort(); };
  }, [value, fetchOptions, tenantId, datasourceId, mergedOptions, resolveOnFetch]);

  // When options are provided by the parent (fetchOptions=false) the passed
  // primaryCube object(s) may lack isCore/isCustom flags. Try to enrich the
  // specific selected value by fetching its definition so the chip can render.
  useEffect(() => {
    if (fetchOptions) return;
    if (!tenantId || !datasourceId) return;
    if (!value) return;
    // value may be string or object
    const key = typeof value === 'string' ? value : (value as any).id || (value as any).name;
    if (!key) return;
    const existing = options.find((o) => String(o.id) === String(key));
    if (!existing) return;
    const hasFlags = Boolean((existing as any).isCore) || Boolean((existing as any).isCustom);
    if (hasFlags) return;

    let active = true;
    const controller = new AbortController();
    (async () => {
      try {
        const url = `/api/fabric/models/definition?tenant_instance_id=${encodeURIComponent(datasourceId || '')}&model_key=${encodeURIComponent(key)}`;
        const res = await fetch(url, { signal: controller.signal, cache: 'no-store' } as any);
        if (!res.ok) return;
        const data = await res.json().catch(() => null);
        const model = data?.model || data;
        if (!model || !active) return;
        const resolved: Partial<CubeOption> = {
          id: model.id || key,
          name: model.model_key || model.id || key,
          display_name: model.display_name || model.model_key,
          description: model.description,
          isCore: Boolean(model.is_core),
          isCustom: Boolean(model.is_custom),
        };
        setEnrichedMap((prev) => ({ ...prev, [String(resolved.id)]: resolved }));
      } catch (e) {
        // ignore
      }
    })();

    return () => { active = false; controller.abort(); };
  }, [value, fetchOptions, tenantId, datasourceId, options]);

  // Resolve the provided value into a full option object when possible so we can
  // reliably show the Core/Custom chip and friendly name. Accept either:
  // - a string id, or
  // - an object that may be partial (missing isCore/isCustom), in which case
  //   we prefer the matching merged option.
  const displayValue = (() => {
    if (!value) return null;
    // value might be a raw id string
    if (typeof value === 'string') {
      return mergedOptions.find((o) => String(o.id) === String(value)) || null;
    }
    // value is an object — if it already contains flags, use it; otherwise try to
    // find the full option by id and prefer the richer option
    const hasFlags = Boolean((value as any).isCore) || Boolean((value as any).isCustom);
    if (hasFlags) return value as CubeOption;
    const found = mergedOptions.find((o) => String(o.id) === String((value as any).id));
    return (found as CubeOption) || (value as CubeOption);
  })();

  const renderOption = (props: any, option: CubeOption) => (
    <li {...props} key={option.id}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Box sx={{ flex: 1 }}>
          <Typography variant="body2">{option.display_name || option.name}</Typography>
          {option.description ? (
            <Typography variant="caption" color="text.secondary">
              {option.description}
            </Typography>
          ) : null}
        </Box>
        <Box>
          {renderCoreCustomChips(option)}
        </Box>
      </Box>
    </li>
  );

  return (
    <Autocomplete
      options={mergedOptions}
      getOptionLabel={(o) => (o ? o.display_name || o.name : '')}
      isOptionEqualToValue={(o, v) => String(o.id) === String(v?.id)}
      value={displayValue}
      onChange={onChange}
      inputValue={inputValue}
      onInputChange={onInputChange}
      disabled={disabled}
      loading={loading}
      renderOption={renderOption}
      renderInput={(params) => {
        // selected chip(s) displayed on the right of the input (endAdornment)
        const selectedChip = displayValue ? <Box component="span">{renderCoreCustomChips(displayValue)}</Box> : null;

        return (
          <TextField
            {...params}
            placeholder={placeholder}
            InputProps={{
              ...params.InputProps,
              endAdornment: (
                <>
                  {selectedChip}
                  {loading ? <CircularProgress color="inherit" size={16} sx={{ ml: 1 }} /> : null}
                  {params.InputProps.endAdornment}
                </>
              ),
            }}
          />
        );
      }}
      sx={{ width: '100%' }}
    />
  );
}
