import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "../lib/apiClient";

export function useMicroBundles() {
  return useQuery({
    queryKey: ["micro-bundles"],
    queryFn: async () => {
  const res = await apiFetch("/api/micro-bundles", { credentials: 'include' });
      return res.json();
    },
  });
}

export function useCreateMicroBundle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (bundle: any) => {
      const res = await apiFetch("/api/micro-bundles", {
        method: "POST",
        credentials: 'include',
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(bundle),
      });
      return res.json();
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["micro-bundles"] }),
  });
}

export function useJITGrants(userId: string) {
  return useQuery({
    queryKey: ["jit-grants", userId],
    queryFn: async () => {
  const res = await apiFetch(`/api/jit-grants?user_id=${userId}`, { credentials: 'include' });
      return res.json();
    },
    enabled: !!userId,
  });
}

export function useCreateJITGrant() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (grant: any) => {
      const res = await apiFetch("/api/jit-grants", {
        method: "POST",
        credentials: 'include',
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(grant),
      });
      return res.json();
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["jit-grants"] }),
  });
}
