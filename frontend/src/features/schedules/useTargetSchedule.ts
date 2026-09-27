import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { schedulesApi, type Schedule } from './api';

/** Load the (at most one) schedule for a product target and drive the embed dialog. */
export function useTargetSchedule(kind: string | undefined, ref: string | undefined, enabled = true) {
  const [open, setOpen] = useState(false);
  const query = useQuery({
    queryKey: ['sched-list', kind, ref],
    enabled: !!kind && !!ref && enabled,
    queryFn: () => schedulesApi.list({ kind: kind!, ref: ref! }),
  });
  const schedule: Schedule | undefined = query.data?.schedules?.[0];
  return {
    open,
    setOpen,
    openEditor: () => setOpen(true),
    closeEditor: () => setOpen(false),
    schedule,
    isLoading: query.isLoading,
    error: query.error,
  };
}
