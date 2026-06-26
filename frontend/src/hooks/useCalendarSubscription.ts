import { useState, useEffect, useCallback, useRef } from 'react';
import { useApolloClient } from '@apollo/client';
import { gql } from '@apollo/client/core';

/**
 * GraphQL subscription for real-time calendar updates
 * This uses Apollo Client subscriptions over WebSocket to Redpanda
 */
export const CALENDAR_UPDATES_SUBSCRIPTION = gql`
  subscription OnCalendarUpdate {
    calendarUpdate {
      eventId
      eventType
      calendarDate
      isBusinessDay
      holidayName
      region
      exchange
      confidenceScore
      sourceSystem
      ruleApplied
      timestamp
    }
  }
`;

/**
 * GraphQL subscription for ingestion lifecycle events
 */
export const INGESTION_LIFECYCLE_SUBSCRIPTION = gql`
  subscription OnIngestionEvent {
    ingestionEvent {
      ingestionId
      eventType
      status
      recordsIngested
      recordsCreated
      recordsUpdated
      conflictsDetected
      sourcesQueried
      sourcesSucceeded
      sourcesFailed
      durationMs
      timestamp
    }
  }
`;

/**
 * GraphQL subscription for conflict detection events
 */
export const CALENDAR_CONFLICTS_SUBSCRIPTION = gql`
  subscription OnConflictDetected {
    conflictDetected {
      conflictId
      region
      calendarDate
      fieldName
      conflictingValues
      sourceSystems
      severity
      reason
      resolved
      timestamp
    }
  }
`;

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
    updateRate: number; // updates per second
  };
  clearEvents: () => void;
}

/**
 * Hook for subscribing to real-time calendar updates
 * Maintains separate event streams for calendar updates, ingestion lifecycle, and conflicts
 * 
 * Usage:
 * ```tsx
 * const { calendarEvents, isConnected, lastUpdate } = useCalendarSubscription();
 * 
 * if (!isConnected) return <p>Connecting...</p>;
 * return <div>{calendarEvents.map(e => <EventCard key={e.eventId} event={e} />)}</div>;
 * ```
 */
export function useCalendarSubscription(): UseCalendarSubscriptionResult {
  const client = useApolloClient();
  const [calendarEvents, setCalendarEvents] = useState<CalendarEvent[]>([]);
  const [ingestionEvents, setIngestionEvents] = useState<IngestionEvent[]>([]);
  const [conflictEvents, setConflictEvents] = useState<ConflictEvent[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error>();
  const [lastCalendarUpdate, setLastCalendarUpdate] = useState<CalendarEvent>();
  const [lastIngestionUpdate, setLastIngestionUpdate] = useState<IngestionEvent>();
  const [updateRate, setUpdateRate] = useState(0);

  const subscriptionsRef = useRef<Set<{ unsubscribe: () => void }>>(new Set());
  const eventCountRef = useRef(0);
  const lastRateCalculationRef = useRef(Date.now());

  /**
   * Calculate events per second
   */
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

  /**
   * Subscribe to all three event streams
   */
  useEffect(() => {
    // Subscribe to calendar updates
    const calendarSub = client
      .subscribe({
        query: CALENDAR_UPDATES_SUBSCRIPTION,
      })
      .subscribe({
        next: (data: any) => {
          const event: CalendarEvent = data.data.calendarUpdate;
          setCalendarEvents((prev) => [event, ...prev].slice(0, 100));
          setLastCalendarUpdate(event);
          eventCountRef.current++;
        },
        error: (err: Error) => {
          console.error('Calendar subscription error:', err);
          setError(err);
          setIsConnected(false);
        },
        complete: () => {
          console.log('Calendar subscription completed');
        },
      });

    subscriptionsRef.current.add(calendarSub);

    // Subscribe to ingestion events
    const ingestionSub = client
      .subscribe({
        query: INGESTION_LIFECYCLE_SUBSCRIPTION,
      })
      .subscribe({
        next: (data: any) => {
          const event: IngestionEvent = data.data.ingestionEvent;
          setIngestionEvents((prev) => [event, ...prev].slice(0, 50));
          setLastIngestionUpdate(event);
          eventCountRef.current++;
        },
        error: (err: Error) => {
          console.error('Ingestion subscription error:', err);
        },
        complete: () => {
          console.log('Ingestion subscription completed');
        },
      });

    subscriptionsRef.current.add(ingestionSub);

    // Subscribe to conflicts
    const conflictSub = client
      .subscribe({
        query: CALENDAR_CONFLICTS_SUBSCRIPTION,
      })
      .subscribe({
        next: (data: any) => {
          const event: ConflictEvent = data.data.conflictDetected;
          setConflictEvents((prev) => [event, ...prev].slice(0, 50));
          eventCountRef.current++;
        },
        error: (err: Error) => {
          console.error('Conflict subscription error:', err);
        },
        complete: () => {
          console.log('Conflict subscription completed');
        },
      });

    subscriptionsRef.current.add(conflictSub);

    setIsConnected(true);
    setError(undefined);

    // Cleanup
    return () => {
      subscriptionsRef.current.forEach((sub) => {
        sub.unsubscribe();
      });
      subscriptionsRef.current.clear();
    };
  }, [client]);

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
