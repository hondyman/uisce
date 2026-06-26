// src/hooks/useNotificationSocket.ts
import { useEffect, useRef } from 'react';
import { io, Socket } from 'socket.io-client';

export interface NotificationLog {
    id: string;
    template_key: string;
    recipient_user_id: string;
    subject: string;
    body: string;
    channel: string;
    status: string;
    priority: string;
    sent_at: string;
    opened_at?: string;
    clicked_at?: string;
    action_taken?: string;
    action_taken_at?: string;
    process_id?: string;
    process_instance_id?: string;
    step_id?: string;
    created_at: string;
}

/**
 * Hook that establishes a WebSocket connection to receive real‑time notifications.
 * It returns a function that can be used to add incoming notifications to the state.
 */
export const useNotificationSocket = (
    tenantId: string,
    datasourceId: string,
    userId: string,
    onNewNotification: (notif: NotificationLog) => void
) => {
    const socketRef = useRef<Socket | null>(null);

    useEffect(() => {
        // Build the URL – adjust if your backend uses a different path or host.
        const url = `${window.location.origin}/ws/notifications`;
        const socket = io(url, {
            transports: ['websocket'],
            query: {
                tenant_id: tenantId,
                tenant_instance_id: datasourceId,
                user_id: userId,
            },
            reconnectionAttempts: 5,
        });
        socketRef.current = socket;

        socket.on('connect', () => {
            console.info('🔔 Notification socket connected');
        });

        socket.on('notification', (data: NotificationLog) => {
            // Ensure the payload matches our type – you may want runtime validation here.
            onNewNotification(data);
        });

        socket.on('disconnect', (reason) => {
            console.warn('🔔 Notification socket disconnected:', reason);
        });

        return () => {
            socket.disconnect();
            socketRef.current = null;
        };
    }, [tenantId, datasourceId, userId, onNewNotification]);

    // Expose a manual reconnect function if needed.
    const reconnect = () => {
        socketRef.current?.connect();
    };

    return { reconnect };
};
