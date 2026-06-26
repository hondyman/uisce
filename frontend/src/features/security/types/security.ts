export interface Permission {
    id: string;
    name: string;
    description: string;
    resource: string;
    action: 'create' | 'read' | 'update' | 'delete' | 'manage';
}

export interface Role {
    role_id: string;
    role_name: string;
    description?: string;
    is_global_admin: boolean;
    tenant_id: string;
    created_at: string;
    updated_at: string;
    permissions?: Permission[];
}

export interface User {
    id: string;
    email: string;
    name: string;
    role: string; // "admin", "user", "global_admin" or custom role name
    organization: string;
    is_active: boolean;
    last_login?: string;
    created_at: string;
}

export interface AuditEvent {
    event_id: string;
    event_type: string;
    entity_type: string;
    entity_id: string;
    actor_id: string; // user_id or system
    tenant_id: string;
    payload: Record<string, any>;
    created_at: string;
    processed: boolean;
}

export interface SecurityStats {
    total_users: number;
    active_sessions: number;
    active_roles: number;
    recent_alerts: number;
    sync_status: 'healthy' | 'degraded' | 'down';
    last_sync_time: string;
}

export interface ComplianceReport {
    id: string;
    title: string;
    type: 'SOC2' | 'GDPR' | 'Internal' | 'ISO27001';
    status: 'draft' | 'generated' | 'published';
    created_at: string;
    created_by: string;
    download_url?: string;
}
