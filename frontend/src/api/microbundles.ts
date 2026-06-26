import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

export function useMicroBundles() {
  return useQuery({
    queryKey: ["micro-bundles"],
    queryFn: async () => {
  const res = await fetch("/api/micro-bundles", { credentials: 'include' });
      if (!res.ok) throw new Error("Failed to fetch micro-bundles");
      return res.json();
    },
  });
}

export function useCreateMicroBundle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (bundle: any) => {
      const res = await fetch("/api/micro-bundles", {
        method: "POST",
        credentials: 'include',
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(bundle),
      });
      if (!res.ok) throw new Error("Failed to create micro-bundle");
      return res.json();
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["micro-bundles"] }),
  });
}

export function useJITGrants(userId: string) {
  return useQuery({
    queryKey: ["jit-grants", userId],
    queryFn: async () => {
  const res = await fetch(`/api/jit-grants?user_id=${userId}`, { credentials: 'include' });
      if (!res.ok) throw new Error("Failed to fetch JIT grants");
      return res.json();
    },
    enabled: !!userId,
  });
}

export function useCreateJITGrant() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (grant: any) => {
      const res = await fetch("/api/jit-grants", {
        method: "POST",
        credentials: 'include',
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(grant),
      });
      if (!res.ok) throw new Error("Failed to create JIT grant");
      return res.json();
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["jit-grants"] }),
  });
}
