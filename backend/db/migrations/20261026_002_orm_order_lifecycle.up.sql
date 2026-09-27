-- 20261026_002_orm_order_lifecycle.up.sql
-- Section 1: Order lifecycle audit. Adds order_event, order_amendment,
-- order_reject, order_history.
-- message_id left as plain UUID (no orm.fix_message exists).

CREATE TABLE IF NOT EXISTS orm.order_event (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    event_type varchar(30) NOT NULL,
    from_status varchar(20),
    to_status varchar(20) NOT NULL,
    event_time timestamptz NOT NULL,
    event_source varchar(30),
    message_id uuid,
    reason_cd varchar(50),
    reason_text varchar(500),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_event_pkey PRIMARY KEY (id),
    CONSTRAINT chk_oe_event_type CHECK (event_type IN (
        'NEW','ACKNOWLEDGED','PARTIAL_FILL','FILLED','CANCELLED',
        'REJECTED','EXPIRED','REPLACED','PENDING_NEW','PENDING_CANCEL',
        'SUSPENDED','HELD','RELEASED')),
    CONSTRAINT chk_oe_source CHECK (event_source IS NULL OR event_source IN (
        'OMS','BROKER','EXCHANGE','MANUAL','SYSTEM')),
    CONSTRAINT fk_oe_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_oe_order ON orm.order_event (order_id, event_time);
CREATE INDEX IF NOT EXISTS idx_oe_tenant ON orm.order_event (tenant_id);

CREATE TABLE IF NOT EXISTS orm.order_amendment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    amendment_num int4 NOT NULL,
    amendment_time timestamptz NOT NULL,
    field_changed varchar(50) NOT NULL,
    prior_value text,
    new_value text,
    reason varchar(500),
    amended_by uuid,
    is_client_directed bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_amendment_pkey PRIMARY KEY (id),
    CONSTRAINT uq_oa UNIQUE (tenant_id, order_id, amendment_num),
    CONSTRAINT fk_oa_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_oa_order ON orm.order_amendment (order_id, amendment_time);

CREATE TABLE IF NOT EXISTS orm.order_reject (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    placement_id uuid,
    reject_time timestamptz NOT NULL,
    reject_source varchar(30) NOT NULL,
    reject_code varchar(50),
    reject_reason varchar(500),
    reject_text text,
    is_retriable bool DEFAULT false NOT NULL,
    retried_as_order_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_reject_pkey PRIMARY KEY (id),
    CONSTRAINT chk_orj_source CHECK (reject_source IN (
        'OMS','BROKER','EXCHANGE','COMPLIANCE','RISK','MANUAL')),
    CONSTRAINT fk_orj_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_orj_order ON orm.order_reject (order_id, reject_time);

CREATE TABLE IF NOT EXISTS orm.order_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    version_num int4 NOT NULL,
    valid_from timestamptz NOT NULL,
    valid_to timestamptz,
    is_current bool DEFAULT true NOT NULL,
    record_snapshot jsonb NOT NULL,
    changed_columns text[],
    change_source varchar(30),
    changed_by uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_history_pkey PRIMARY KEY (id),
    CONSTRAINT uq_oh_version UNIQUE (tenant_id, order_id, version_num),
    CONSTRAINT fk_oh_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_oh_order ON orm.order_history (order_id, is_current);
