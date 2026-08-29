export interface Permission {
    id: string;
    name: string;
    description: string;
    resource: string;
    action: 'create' | 'read' | 'update' | 'delete' | 'manage';
}

export interface Role {
    id: string;
    tenant_id: string;
    role_key: string;
    role_name: string;
    description?: string;
    role_type: 'system' | 'custom';
    role_level: 'viewer' | 'editor' | 'admin' | 'super_admin';
    is_active: boolean;
    // is_template marks a gold-copy role authored in the gold-copy tenant and
    // inheritable by every other tenant (see backend bp_roles.is_template).
    is_template: boolean;
    parent_role_id?: string | null;
    security_profile_id?: string | null;
    // Derived by the backend, not stored: "gold_copy" (a template role),
    // "extended" (a local clone of a template role), or "tenant" (fully custom).
    origin: 'gold_copy' | 'extended' | 'tenant';
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
