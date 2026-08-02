import { useState, useCallback } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { calendarApi } from '../services/api';
import type { ExternalCalendar, SyncStatus, SyncedEvent } from '../types/calendar';
import type { CalendarProvider } from '../services/api';

export const useCalendarSync = (tenantId: string, userId: string) => {
    const [activeSyncId, setActiveSyncId] = useState<string | null>(null);
    const [activeProvider, setActiveProvider] = useState<CalendarProvider>('google');

    // Fetch Auth URL for Google
    const { data: authData, isLoading: isLoadingAuth } = useQuery({
        queryKey: ['calendarAuthUrl', tenantId, userId, 'google'],
        queryFn: () => calendarApi.getAuthUrl(tenantId, userId, 'google'),
        enabled: !!tenantId && !!userId,
    });

    // Fetch Auth URL for Microsoft
    const { data: msAuthData, isLoading: isLoadingMsAuth } = useQuery({
        queryKey: ['calendarAuthUrl', tenantId, userId, 'microsoft'],
        queryFn: () => calendarApi.getAuthUrl(tenantId, userId, 'microsoft'),
        enabled: !!tenantId && !!userId,
    });

    // Fetch Calendars for Google
    const { data: calendars, isLoading: isLoadingCalendars, refetch: refetchCalendars } = useQuery<ExternalCalendar[]>({
        queryKey: ['calendars', tenantId, userId, 'google'],
        queryFn: () => calendarApi.getCalendars(tenantId, userId, 'google'),
        enabled: !!tenantId && !!userId,
        retry: false,
    });

    // Fetch Calendars for Microsoft
    const { data: msCalendars, isLoading: isLoadingMsCalendars, refetch: refetchMsCalendars } = useQuery<ExternalCalendar[]>({
        queryKey: ['calendars', tenantId, userId, 'microsoft'],
        queryFn: () => calendarApi.getCalendars(tenantId, userId, 'microsoft'),
        enabled: !!tenantId && !!userId,
        retry: false,
    });

    // Fetch Calendars for Apple
    const { data: appleCalendars, isLoading: isLoadingAppleCalendars, refetch: refetchAppleCalendars } = useQuery<ExternalCalendar[]>({
        queryKey: ['calendars', tenantId, userId, 'apple'],
        queryFn: () => calendarApi.getCalendars(tenantId, userId, 'apple'),
        enabled: !!tenantId && !!userId,
        retry: false,
    });

    // Trigger Sync for a specific provider
    const syncMutation = useMutation({
        mutationFn: ({ calendarId, provider }: { calendarId: string; provider: CalendarProvider }) =>
            calendarApi.triggerSync(tenantId, userId, provider, calendarId),
        onSuccess: (data) => {
            setActiveSyncId(data.id || data.sync_id);
            setActiveProvider('google');
        },
    });

    // Trigger Microsoft Sync
    const msSyncMutation = useMutation({
        mutationFn: (calendarId: string) =>
            calendarApi.triggerSync(tenantId, userId, 'microsoft', calendarId),
        onSuccess: (data) => {
            setActiveSyncId(data.id || data.sync_id);
            setActiveProvider('microsoft');
        },
    });

    // Poll Sync Status
    const { data: syncStatus } = useQuery<SyncStatus>({
        queryKey: ['syncStatus', activeSyncId],
        queryFn: () => calendarApi.getSyncStatus(activeSyncId!),
        enabled: !!activeSyncId,
        refetchInterval: (query) => {
            const status = query?.state?.data?.status;
            return status === 'completed' || status === 'failed' || status === 'cancelled' ? false : 2000;
        },
    });

    // Fetch Synced Events (unified, all providers)
    const { data: syncedEvents, isLoading: isLoadingEvents, refetch: refetchEvents } = useQuery<SyncedEvent[]>({
        queryKey: ['syncedEvents', tenantId, userId],
        queryFn: () => calendarApi.getSyncedEvents(tenantId, userId),
        enabled: !!tenantId,
    });

    const handleConnect = useCallback(() => {
        if (authData?.auth_url) {
            window.location.href = authData.auth_url;
        }
    }, [authData]);

    const handleMicrosoftConnect = useCallback(() => {
        if (msAuthData?.auth_url) {
            window.location.href = msAuthData.auth_url;
        }
    }, [msAuthData]);

    const handleSync = useCallback((calendarId: string) => {
        syncMutation.mutate({ calendarId, provider: 'google' });
    }, [syncMutation]);

    const handleMicrosoftSync = useCallback((calendarId: string) => {
        msSyncMutation.mutate(calendarId);
    }, [msSyncMutation]);

    const isGoogleSyncing = syncMutation.isPending || syncStatus?.status === 'running';
    const isMicrosoftSyncing = msSyncMutation.isPending || (syncStatus?.status === 'running' && activeProvider === 'microsoft');

    return {
        // Google integration properties
        authUrl: authData?.auth_url,
        isLoadingAuth,
        isConnected: !!calendars && calendars.length > 0,
        calendars,
        isLoadingCalendars,
        refetchCalendars,
        handleConnect,
        handleSync,
        isSyncing: isGoogleSyncing,
        error: syncMutation.error,

        // Microsoft integration properties
        msAuthUrl: msAuthData?.auth_url,
        isLoadingMsAuth,
        isMicrosoftConnected: !!msCalendars && msCalendars.length > 0,
        msCalendars,
        isLoadingMsCalendars,
        refetchMsCalendars,
        handleMicrosoftConnect,
        handleMicrosoftSync,
        isMsSyncing: isMicrosoftSyncing,
        msError: msSyncMutation.error,

        // Apple integration properties
        appleCalendars,
        isLoadingAppleCalendars,
        refetchAppleCalendars,

        // Unified properties
        syncStatus,
        syncedEvents,
        isLoadingEvents,
        refetchEvents,
    };
};
