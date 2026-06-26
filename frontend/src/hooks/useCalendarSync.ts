import { useState, useCallback } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { calendarApi } from '../services/api';
import { GoogleCalendar, SyncStatus, SyncedEvent } from '../types/calendar';

export const useCalendarSync = (tenantId: string, userId: string) => {
    const [activeSyncId, setActiveSyncId] = useState<string | null>(null);

    // Fetch Auth URL
    const { data: authData, isLoading: isLoadingAuth } = useQuery({
        queryKey: ['googleAuthUrl', tenantId, userId],
        queryFn: () => calendarApi.getAuthUrl(tenantId, userId),
        enabled: !!tenantId && !!userId,
    });

    const { data: msAuthData, isLoading: isLoadingMsAuth } = useQuery({
        queryKey: ['microsoftAuthUrl', tenantId, userId],
        queryFn: () => calendarApi.getMicrosoftAuthUrl(tenantId, userId),
        enabled: !!tenantId && !!userId,
    });

    // Fetch Calendars
    const { data: calendars, isLoading: isLoadingCalendars, refetch: refetchCalendars } = useQuery<GoogleCalendar[]>({
        queryKey: ['googleCalendars', tenantId, userId],
        queryFn: () => calendarApi.getCalendars(tenantId, userId),
        enabled: !!tenantId && !!userId,
        retry: false, // Don't retry if not authenticated
    });

    const { data: msCalendars, isLoading: isLoadingMsCalendars, refetch: refetchMsCalendars } = useQuery<GoogleCalendar[]>({
        queryKey: ['microsoftCalendars', tenantId, userId],
        queryFn: () => calendarApi.getMicrosoftCalendars(tenantId, userId),
        enabled: !!tenantId && !!userId,
        retry: false,
    });

    // Trigger Sync
    const syncMutation = useMutation({
        mutationFn: (calendarId: string) => calendarApi.triggerSync(tenantId, userId, calendarId),
        onSuccess: (data) => {
            setActiveSyncId(data.sync_id || data.id);
        },
    });

    const msSyncMutation = useMutation({
        mutationFn: (calendarId: string) => calendarApi.triggerMicrosoftSync(tenantId, userId, calendarId),
        onSuccess: (data) => {
            setActiveSyncId(data.sync_id || data.id);
        },
    });

    // Poll Sync Status
    const { data: syncStatus } = useQuery<SyncStatus>({
        queryKey: ['syncStatus', activeSyncId],
        queryFn: () => calendarApi.getSyncStatus(activeSyncId!),
        enabled: !!activeSyncId,
        refetchInterval: (query) => {
            const status = query?.state?.data?.status;
            return status === 'completed' || status === 'failed' ? false : 2000;
        },
    });

    // Fetch Synced Events
    const { data: syncedEvents, isLoading: isLoadingEvents, refetch: refetchEvents } = useQuery<SyncedEvent[]>({
        queryKey: ['syncedEvents', tenantId],
        queryFn: () => calendarApi.getSyncedEvents(tenantId),
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
        syncMutation.mutate(calendarId);
    }, [syncMutation]);

    const handleMicrosoftSync = useCallback((calendarId: string) => {
        msSyncMutation.mutate(calendarId);
    }, [msSyncMutation]);

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
        isSyncing: syncMutation.isPending || syncStatus?.status === 'running',
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
        isMsSyncing: msSyncMutation.isPending || syncStatus?.status === 'running',
        msError: msSyncMutation.error,

        // Unified properties
        syncStatus,
        syncedEvents,
        isLoadingEvents,
        refetchEvents,
    };
};
