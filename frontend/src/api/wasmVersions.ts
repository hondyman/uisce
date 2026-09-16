import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface WASMVersion {
  wasm_version_id: string;
  module_name: string;
  version: string;
  build_hash: string;
  build_time: string;
  artifact_uri: string;
  checksum_sha256: string;
  is_active: boolean;
}

export function useWASMVersions(moduleName: string) {
  return useQuery({
    queryKey: ['wasm-versions', moduleName],
    queryFn: async () => {
      const res = await apiFetch(`/api/wasm-versions?module_name=${moduleName}`);
      const json = await res.json();
      return (json.versions ?? []) as WASMVersion[];
    },
    enabled: !!moduleName,
  });
}

export function useActivateWASMVersion() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await apiFetch(`/api/wasm-versions/${id}/activate`, {
        method: 'POST',
      });
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['wasm-versions'] });
    },
  });
}
