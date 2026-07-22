import { useState, useEffect, useCallback, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface CalendarEvent {
  eventId: string;
  eventType: string;
  calendarDate: string;
  isBusinessDay: boolean;
  holidayName?: string;
  region: string;
  exchange?: string;
  confidenceScore: number;
  sourceSystem: string;
  ruleApplied?: string;
  timestamp: number;
}

export interface IngestionEvent {
  ingestionId: string;
  eventType: string;
  status: string;
  recordsIngested: number;
  recordsCreated: number;
  recordsUpdated: number;
  conflictsDetected: number;
  sourcesQueried: number;
  sourcesSucceeded: number;
  sourcesFailed: number;
  durationMs: number;
  timestamp: number;
}

export interface ConflictEvent {
  conflictId: string;
  region: string;
  calendarDate: string;
  fieldName: string;
  conflictingValues: string[];
  sourceSystems: string[];
  severity: number;
  reason: string;
  resolved: boolean;
  timestamp: number;
}

export interface UseCalendarSubscriptionResult {
  calendarEvents: CalendarEvent[];
  ingestionEvents: IngestionEvent[];
  conflictEvents: ConflictEvent[];
  isConnected: boolean;
  error?: Error;
  lastCalendarUpdate?: CalendarEvent;
  lastIngestionUpdate?: IngestionEvent;
  stats: {
    totalCalendarEvents: number;
    totalIngestionEvents: number;
    totalConflicts: number;
    updateRate: number;
  };
  clearEvents: () => void;
}

export function useCalendarSubscription(): UseCalendarSubscriptionResult {
  const [calendarEvents, setCalendarEvents] = useState<CalendarEvent[]>([]);
  const [ingestionEvents, setIngestionEvents] = useState<IngestionEvent[]>([]);
  const [conflictEvents, setConflictEvents] = useState<ConflictEvent[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error>();
  const [lastCalendarUpdate, setLastCalendarUpdate] = useState<CalendarEvent>();
  const [lastIngestionUpdate, setLastIngestionUpdate] = useState<IngestionEvent>();
  const [updateRate, setUpdateRate] = useState(0);

  const eventCountRef = useRef(0);
  const lastRateCalculationRef = useRef(Date.now());

  const { data: calendarData, error: calendarError } = useQuery({
    queryKey: ['calendar-updates'],
    queryFn: () => apiFetch('/api/calendar/updates').then(r => r.json()),
    refetchInterval: 5000,
  });

  const { data: ingestionData, error: ingestionError } = useQuery({
    queryKey: ['ingestion-events'],
    queryFn: () => apiFetch('/api/ingestion/events').then(r => r.json()),
    refetchInterval: 5000,
  });

  const { data: conflictData, error: conflictError } = useQuery({
    queryKey: ['calendar-conflicts'],
    queryFn: () => apiFetch('/api/calendar/conflicts').then(r => r.json()),
    refetchInterval: 5000,
  });

  useEffect(() => {
    if (calendarError) {
      setError(calendarError as Error);
      setIsConnected(false);
    } else if (calendarData) {
      setIsConnected(true);
      setError(undefined);
    }
  }, [calendarData, calendarError]);

  useEffect(() => {
    if (calendarData?.eventId) {
      const event: CalendarEvent = calendarData;
      setCalendarEvents(prev => [event, ...prev].slice(0, 100));
      setLastCalendarUpdate(event);
      eventCountRef.current++;
    }
  }, [calendarData]);

  useEffect(() => {
    if (ingestionData?.ingestionId) {
      const event: IngestionEvent = ingestionData;
      setIngestionEvents(prev => [event, ...prev].slice(0, 50));
      setLastIngestionUpdate(event);
      eventCountRef.current++;
    }
  }, [ingestionData]);

  useEffect(() => {
    if (conflictData?.conflictId) {
      const event: ConflictEvent = conflictData;
      setConflictEvents(prev => [event, ...prev].slice(0, 50));
      eventCountRef.current++;
    }
  }, [conflictData]);

  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now();
      const elapsed = (now - lastRateCalculationRef.current) / 1000;
      const rate = elapsed > 0 ? eventCountRef.current / elapsed : 0;
      setUpdateRate(Math.round(rate * 100) / 100);
      eventCountRef.current = 0;
      lastRateCalculationRef.current = now;
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  const clearEvents = useCallback(() => {
    setCalendarEvents([]);
    setIngestionEvents([]);
    setConflictEvents([]);
  }, []);

  return {
    calendarEvents,
    ingestionEvents,
    conflictEvents,
    isConnected,
    error,
    lastCalendarUpdate,
    lastIngestionUpdate,
    stats: {
      totalCalendarEvents: calendarEvents.length,
      totalIngestionEvents: ingestionEvents.length,
      totalConflicts: conflictEvents.length,
      updateRate,
    },
    clearEvents,
  };
}
