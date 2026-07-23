-- DROP SCHEMA public;

CREATE SCHEMA public AUTHORIZATION pg_database_owner;

-- DROP TYPE public.fabric_status;

CREATE TYPE public.fabric_status AS ENUM (
	'draft',
	'published',
	'archived');

-- DROP TYPE public.gtrgm;

CREATE TYPE public.gtrgm (
	INPUT = gtrgm_in,
	OUTPUT = gtrgm_out,
	ALIGNMENT = 4,
	STORAGE = plain,
	CATEGORY = U,
	DELIMITER = ',');

-- DROP TYPE public.join_relationship;

CREATE TYPE public.join_relationship AS ENUM (
	'one_to_one',
	'one_to_many',
	'many_to_one',
	'many_to_many');

-- DROP SEQUENCE public.fabric_defn_audit_audit_id_seq;

CREATE SEQUENCE public.fabric_defn_audit_audit_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	START 1
	CACHE 1
	NO CYCLE;-- public.alpha_datasource definition

-- Drop table

-- DROP TABLE public.alpha_datasource;

CREATE TABLE public.alpha_datasource (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	datasource_name varchar NOT NULL,
	datasource_code varchar NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	config jsonb NULL,
	created_at timestamptz NULL,
	updated_at timestamptz NULL,
	datasource_type varchar NULL,
	CONSTRAINT alpha_datasources_pk PRIMARY KEY (id),
	CONSTRAINT alpha_datasources_unique UNIQUE (datasource_code)
);


-- public.alpha_product definition

-- Drop table

-- DROP TABLE public.alpha_product;

CREATE TABLE public.alpha_product (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	product_name varchar(255) NULL,
	is_active bool DEFAULT true NOT NULL,
	product_code varchar(255) NULL,
	status varchar(50) DEFAULT 'active'::character varying NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT product_pkey PRIMARY KEY (id),
	CONSTRAINT product_unique UNIQUE (product_name)
);

-- Table Triggers

create trigger update_product_updated_at before
update
    on
    public.alpha_product for each row execute function update_product_updated_at();


-- public.app_user definition

-- Drop table

-- DROP TABLE public.app_user;

CREATE TABLE public.app_user (
	id text NOT NULL,
	email text NOT NULL,
	display_name text NULL,
	created_at timestamp DEFAULT now() NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	CONSTRAINT app_user_email_key UNIQUE (email),
	CONSTRAINT app_user_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_app_user_active ON public.app_user USING btree (is_active) WHERE (is_active = true);
CREATE INDEX idx_app_user_email ON public.app_user USING btree (email);


-- public.audit_alerts definition

-- Drop table

-- DROP TABLE public.audit_alerts;

CREATE TABLE public.audit_alerts (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	event_type varchar(50) NOT NULL,
	severity varchar(20) NOT NULL,
	conditions jsonb NULL,
	enabled bool DEFAULT true NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT audit_alerts_pkey PRIMARY KEY (id)
);


-- public.audit_events definition

-- Drop table

-- DROP TABLE public.audit_events;

CREATE TABLE public.audit_events (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	"timestamp" timestamptz DEFAULT now() NOT NULL,
	event_type varchar(50) NOT NULL,
	severity varchar(20) DEFAULT 'medium'::character varying NOT NULL,
	user_id varchar(255) NULL,
	tenant_id varchar(255) NULL,
	session_id varchar(255) NULL,
	resource_id varchar(255) NULL,
	resource_type varchar(100) NULL,
	"action" varchar(100) NULL,
	ip_address inet NULL,
	user_agent text NULL,
	request_id varchar(255) NULL,
	details jsonb NULL,
	old_values jsonb NULL,
	new_values jsonb NULL,
	success bool DEFAULT true NOT NULL,
	error_message text NULL,
	compliance_flags _text NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT audit_events_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_audit_events_event_type ON public.audit_events USING btree (event_type);
CREATE INDEX idx_audit_events_ip_address ON public.audit_events USING btree (ip_address);
CREATE INDEX idx_audit_events_resource_id ON public.audit_events USING btree (resource_id);
CREATE INDEX idx_audit_events_resource_type ON public.audit_events USING btree (resource_type);
CREATE INDEX idx_audit_events_severity ON public.audit_events USING btree (severity);
CREATE INDEX idx_audit_events_success ON public.audit_events USING btree (success);
CREATE INDEX idx_audit_events_tenant_id ON public.audit_events USING btree (tenant_id);
CREATE INDEX idx_audit_events_tenant_time ON public.audit_events USING btree (tenant_id, "timestamp" DESC);
CREATE INDEX idx_audit_events_timestamp ON public.audit_events USING btree ("timestamp" DESC);
CREATE INDEX idx_audit_events_type_time ON public.audit_events USING btree (event_type, "timestamp" DESC);
CREATE INDEX idx_audit_events_user_id ON public.audit_events USING btree (user_id);
CREATE INDEX idx_audit_events_user_time ON public.audit_events USING btree (user_id, "timestamp" DESC);


-- public.audit_logs definition

-- Drop table

-- DROP TABLE public.audit_logs;

CREATE TABLE public.audit_logs (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	user_id uuid NOT NULL,
	datasource_id uuid NOT NULL,
	policy_id uuid NOT NULL,
	access_time timestamptz DEFAULT now() NOT NULL,
	"action" varchar(50) NOT NULL,
	details jsonb NULL,
	instance_datasource_id uuid NULL,
	tenant_id uuid NOT NULL,
	"type" varchar(50) NOT NULL,
	resource_type varchar(50) NOT NULL,
	resource_id uuid NOT NULL,
	request_id varchar(100) NULL,
	"timestamp" timestamptz DEFAULT now() NOT NULL,
	duration int8 NULL,
	status varchar(50) NOT NULL,
	error_message text NULL,
	request_data jsonb NULL,
	response_data jsonb NULL,
	ip_address inet NULL,
	user_agent text NULL,
	CONSTRAINT action_check CHECK (((action)::text = ANY (ARRAY[('allow'::character varying)::text, ('deny'::character varying)::text, ('mask'::character varying)::text]))),
	CONSTRAINT audit_logs_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_audit_logs_resource ON public.audit_logs USING btree (resource_type, resource_id);
CREATE INDEX idx_audit_logs_tenant_timestamp ON public.audit_logs USING btree (tenant_id, "timestamp");
CREATE INDEX idx_audit_logs_user ON public.audit_logs USING btree (user_id);


-- public.audit_retention_policies definition

-- Drop table

-- DROP TABLE public.audit_retention_policies;

CREATE TABLE public.audit_retention_policies (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	event_type varchar(50) NOT NULL,
	retention_days int4 NOT NULL,
	archive_after_days int4 NULL,
	delete_after_days int4 NULL,
	enabled bool DEFAULT true NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT audit_retention_policies_event_type_key UNIQUE (event_type),
	CONSTRAINT audit_retention_policies_pkey PRIMARY KEY (id)
);


-- public.audit_summaries definition

-- Drop table

-- DROP TABLE public.audit_summaries;

CREATE TABLE public.audit_summaries (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	summary_date date NOT NULL,
	tenant_id varchar(255) NULL,
	total_events int8 DEFAULT 0 NOT NULL,
	events_by_type jsonb NULL,
	events_by_severity jsonb NULL,
	events_by_user jsonb NULL,
	critical_events int8 DEFAULT 0 NOT NULL,
	compliance_violations int8 DEFAULT 0 NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT audit_summaries_pkey PRIMARY KEY (id),
	CONSTRAINT audit_summaries_summary_date_tenant_id_key UNIQUE (summary_date, tenant_id)
);


-- public.bundle_change_proposal definition

-- Drop table

-- DROP TABLE public.bundle_change_proposal;

CREATE TABLE public.bundle_change_proposal (
	id uuid NOT NULL,
	bundle_id uuid NULL,
	proposed_version int4 NULL,
	change_type text NULL,
	details jsonb NULL,
	fitness_score float8 NULL,
	risk_score float8 NULL,
	impact jsonb NULL,
	status text NULL,
	created_at timestamptz NULL,
	decided_at timestamptz NULL,
	decided_by text NULL,
	CONSTRAINT bundle_change_proposal_pkey PRIMARY KEY (id)
);


-- public.candidate_bundles definition

-- Drop table

-- DROP TABLE public.candidate_bundles;

CREATE TABLE public.candidate_bundles (
	id text NOT NULL,
	tenant_id text NULL,
	"name" text NULL,
	description text NULL,
	claims jsonb NULL,
	"scope" text NULL,
	score float8 NULL,
	risk float8 NULL,
	explanations jsonb NULL,
	status text NULL,
	created_at timestamptz NULL,
	CONSTRAINT candidate_bundles_pkey PRIMARY KEY (id)
);


-- public.claim_bundle definition

-- Drop table

-- DROP TABLE public.claim_bundle;

CREATE TABLE public.claim_bundle (
	id uuid NOT NULL,
	"name" text NULL,
	"version" int4 NULL,
	"domain" text NULL,
	description text NULL,
	created_by text NULL,
	created_at timestamptz NULL,
	status text NULL,
	risk_level text NULL,
	CONSTRAINT claim_bundle_pkey PRIMARY KEY (id)
);


-- public.claim_bundle_item definition

-- Drop table

-- DROP TABLE public.claim_bundle_item;

CREATE TABLE public.claim_bundle_item (
	id uuid NOT NULL,
	bundle_id uuid NULL,
	model_id uuid NULL,
	"permission" text NULL,
	"scope" jsonb NULL,
	CONSTRAINT claim_bundle_item_pkey PRIMARY KEY (id)
);


-- public.compliance_reports definition

-- Drop table

-- DROP TABLE public.compliance_reports;

CREATE TABLE public.compliance_reports (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	report_type varchar(100) NOT NULL,
	start_date timestamptz NOT NULL,
	end_date timestamptz NOT NULL,
	generated_at timestamptz DEFAULT now() NOT NULL,
	generated_by varchar(255) NOT NULL,
	summary jsonb NULL,
	violations jsonb NULL,
	recommendations _text NULL,
	status varchar(50) DEFAULT 'generated'::character varying NOT NULL,
	file_path text NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT compliance_reports_pkey PRIMARY KEY (id)
);


-- public.connection_pool_metrics definition

-- Drop table

-- DROP TABLE public.connection_pool_metrics;

CREATE TABLE public.connection_pool_metrics (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	pool_name varchar(100) NOT NULL,
	total_connections int4 NOT NULL,
	active_connections int4 NOT NULL,
	idle_connections int4 NOT NULL,
	waiting_requests int4 NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	CONSTRAINT connection_pool_metrics_pkey PRIMARY KEY (id)
);


-- public.data_access_log definition

-- Drop table

-- DROP TABLE public.data_access_log;

CREATE TABLE public.data_access_log (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	user_id varchar(255) NOT NULL,
	tenant_id varchar(255) NULL,
	session_id varchar(255) NULL,
	resource_type varchar(100) NOT NULL,
	resource_id varchar(255) NOT NULL,
	"action" varchar(50) NOT NULL,
	ip_address inet NULL,
	user_agent text NULL,
	request_id varchar(255) NULL,
	query_parameters jsonb NULL,
	accessed_fields _text NULL,
	record_count int4 NULL,
	access_time timestamptz DEFAULT now() NOT NULL,
	success bool DEFAULT true NOT NULL,
	error_message text NULL,
	CONSTRAINT data_access_log_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_data_access_log_resource ON public.data_access_log USING btree (resource_type, resource_id);
CREATE INDEX idx_data_access_log_time ON public.data_access_log USING btree (access_time DESC);
CREATE INDEX idx_data_access_log_user_time ON public.data_access_log USING btree (user_id, access_time DESC);


-- public.drift_reports definition

-- Drop table

-- DROP TABLE public.drift_reports;

CREATE TABLE public.drift_reports (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	generated_at timestamptz NOT NULL,
	schema_hash text NOT NULL,
	severity_summary jsonb NOT NULL,
	changelog_md text NULL,
	changelog_html text NULL,
	raw_report jsonb NOT NULL,
	CONSTRAINT drift_reports_pkey PRIMARY KEY (id)
);


-- public.engagement_notifications definition

-- Drop table

-- DROP TABLE public.engagement_notifications;

CREATE TABLE public.engagement_notifications (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	user_id varchar(255) NOT NULL,
	"type" varchar(100) NOT NULL,
	title varchar(500) NOT NULL,
	message text NOT NULL,
	rich_content jsonb NULL,
	priority int4 DEFAULT 2 NULL,
	channels _text DEFAULT ARRAY['in_app'::text] NOT NULL,
	status varchar(50) DEFAULT 'draft'::character varying NULL,
	scheduled_at timestamptz NULL,
	sent_at timestamptz NULL,
	read_at timestamptz NULL,
	clicked_at timestamptz NULL,
	dismissed_at timestamptz NULL,
	expires_at timestamptz NULL,
	created_by varchar(255) NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	engagement_score numeric(5, 4) DEFAULT 0 NULL,
	user_segment varchar(100) NULL,
	ab_test_variant varchar(100) NULL,
	template_id varchar(255) NULL,
	personalization jsonb NULL,
	actions jsonb NULL,
	cta jsonb NULL,
	CONSTRAINT engagement_notifications_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_engagement_notifications_created_at ON public.engagement_notifications USING btree (created_at);
CREATE INDEX idx_engagement_notifications_scheduled_at ON public.engagement_notifications USING btree (scheduled_at);
CREATE INDEX idx_engagement_notifications_status ON public.engagement_notifications USING btree (status);
CREATE INDEX idx_engagement_notifications_type ON public.engagement_notifications USING btree (type);
CREATE INDEX idx_engagement_notifications_user_id ON public.engagement_notifications USING btree (user_id);

-- Table Triggers

create trigger update_engagement_notifications_updated_at before
update
    on
    public.engagement_notifications for each row execute function update_engagement_notifications_updated_at();


-- public.explorer_saved_query definition

-- Drop table

-- DROP TABLE public.explorer_saved_query;

CREATE TABLE public.explorer_saved_query (
	id uuid NOT NULL,
	created_at timestamp DEFAULT now() NOT NULL,
	updated_at timestamp DEFAULT now() NOT NULL,
	owner_user_id text NOT NULL,
	owner_tenant_id text NOT NULL,
	"name" text NOT NULL,
	description text NULL,
	tags _text NULL,
	view_name text NOT NULL,
	query jsonb NOT NULL,
	viz_config jsonb NULL,
	last_run_at timestamp NULL,
	last_duration_ms int4 NULL,
	CONSTRAINT explorer_saved_query_pkey PRIMARY KEY (id)
);


-- public.fabric_defn_audit definition

-- Drop table

-- DROP TABLE public.fabric_defn_audit;

CREATE TABLE public.fabric_defn_audit (
	audit_id bigserial NOT NULL,
	defn_id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	model_key text NOT NULL,
	"version" int4 NOT NULL,
	"action" text NOT NULL,
	"at" timestamptz DEFAULT now() NOT NULL,
	actor_id uuid NOT NULL,
	before_doc jsonb NULL,
	after_doc jsonb NULL,
	CONSTRAINT fabric_defn_audit_action_check CHECK ((action = ANY (ARRAY['create'::text, 'update'::text, 'publish'::text, 'archive'::text]))),
	CONSTRAINT fabric_defn_audit_pkey PRIMARY KEY (audit_id)
);


-- public.fabric_defn_index definition

-- Drop table

-- DROP TABLE public.fabric_defn_index;

CREATE TABLE public.fabric_defn_index (
	defn_id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	model_key text NOT NULL,
	"version" int4 NOT NULL,
	kind text NOT NULL,
	"name" text NOT NULL,
	"type" text NULL,
	relationship public.join_relationship NULL,
	title text NULL,
	description text NULL,
	"sql" text NULL,
	CONSTRAINT fabric_defn_index_kind_check CHECK ((kind = ANY (ARRAY['dimension'::text, 'measure'::text, 'join'::text]))),
	CONSTRAINT fabric_defn_index_tenant_id_model_key_version_kind_name_key UNIQUE (tenant_id, model_key, version, kind, name)
);


-- public.integration_logs definition

-- Drop table

-- DROP TABLE public.integration_logs;

CREATE TABLE public.integration_logs (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	integration_id uuid NOT NULL,
	event_id uuid NOT NULL,
	request_id uuid NOT NULL,
	execution_start timestamptz NOT NULL,
	execution_end timestamptz NOT NULL,
	duration_ms int4 NOT NULL,
	status varchar(20) NOT NULL,
	status_code int4 NULL,
	error_message text NULL,
	request_body text NULL,
	response_body text NULL,
	attempt_count int4 DEFAULT 1 NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT integration_logs_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_integration_logs_event ON public.integration_logs USING btree (event_id);
CREATE INDEX idx_integration_logs_execution_start ON public.integration_logs USING btree (execution_start);
CREATE INDEX idx_integration_logs_integration ON public.integration_logs USING btree (integration_id);
CREATE INDEX idx_integration_logs_status ON public.integration_logs USING btree (status);
CREATE INDEX idx_integration_logs_tenant ON public.integration_logs USING btree (tenant_id);


-- public.message_templates definition

-- Drop table

-- DROP TABLE public.message_templates;

CREATE TABLE public.message_templates (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	teams_json text NOT NULL,
	slack_json text NOT NULL,
	tenant_id uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT message_templates_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_message_templates_tenant_id ON public.message_templates USING btree (tenant_id);


-- public.model_upgrade_audit definition

-- Drop table

-- DROP TABLE public.model_upgrade_audit;

CREATE TABLE public.model_upgrade_audit (
	id uuid NOT NULL,
	diff_id uuid NULL,
	model_name text NOT NULL,
	field_path text NULL,
	rule_id text NULL,
	provenance text NULL,
	decision text NULL,
	reviewer text NULL,
	reason text NULL,
	event_type text NOT NULL,
	decided_at timestamp DEFAULT now() NOT NULL,
	CONSTRAINT model_upgrade_audit_event_type_check CHECK ((event_type = ANY (ARRAY['upgrade'::text, 'tuning'::text, 'preagg'::text]))),
	CONSTRAINT model_upgrade_audit_pkey PRIMARY KEY (id)
);


-- public.notification_analytics definition

-- Drop table

-- DROP TABLE public.notification_analytics;

CREATE TABLE public.notification_analytics (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	notification_id varchar(255) NOT NULL,
	user_id varchar(255) NOT NULL,
	event_type varchar(50) NOT NULL,
	event_timestamp timestamptz DEFAULT now() NULL,
	user_agent text NULL,
	ip_address inet NULL,
	device_type varchar(100) NULL,
	"location" varchar(255) NULL,
	session_id varchar(255) NULL,
	additional_metadata jsonb NULL,
	CONSTRAINT notification_analytics_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_notification_analytics_event_type ON public.notification_analytics USING btree (event_type);
CREATE INDEX idx_notification_analytics_notification_id ON public.notification_analytics USING btree (notification_id);
CREATE INDEX idx_notification_analytics_timestamp ON public.notification_analytics USING btree (event_timestamp);
CREATE INDEX idx_notification_analytics_user_id ON public.notification_analytics USING btree (user_id);


-- public.notification_campaigns definition

-- Drop table

-- DROP TABLE public.notification_campaigns;

CREATE TABLE public.notification_campaigns (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	"type" varchar(100) NOT NULL,
	status varchar(50) DEFAULT 'draft'::character varying NULL,
	target_users _text NULL,
	user_segment varchar(100) NULL,
	steps jsonb DEFAULT '[]'::jsonb NOT NULL,
	created_by varchar(255) NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT notification_campaigns_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_notification_campaigns_status ON public.notification_campaigns USING btree (status);
CREATE INDEX idx_notification_campaigns_type ON public.notification_campaigns USING btree (type);

-- Table Triggers

create trigger update_notification_campaigns_updated_at before
update
    on
    public.notification_campaigns for each row execute function update_notification_campaigns_updated_at();


-- public.notification_templates definition

-- Drop table

-- DROP TABLE public.notification_templates;

CREATE TABLE public.notification_templates (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	"name" varchar(255) NOT NULL,
	"type" varchar(100) NOT NULL,
	subject varchar(500) NULL,
	title varchar(500) NOT NULL,
	message text NOT NULL,
	rich_content jsonb NULL,
	variables _text NULL,
	channels _text DEFAULT ARRAY['in_app'::text] NOT NULL,
	created_by varchar(255) NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT notification_templates_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_notification_templates_type ON public.notification_templates USING btree (type);

-- Table Triggers

create trigger update_notification_templates_updated_at before
update
    on
    public.notification_templates for each row execute function update_notification_templates_updated_at();


-- public.performance_metrics definition

-- Drop table

-- DROP TABLE public.performance_metrics;

CREATE TABLE public.performance_metrics (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_id varchar(255) NOT NULL,
	metric_name varchar(100) NOT NULL,
	metric_value numeric(15, 6) NOT NULL,
	labels jsonb NULL,
	collected_at timestamptz DEFAULT now() NULL,
	CONSTRAINT performance_metrics_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_performance_metrics_name_collected ON public.performance_metrics USING btree (metric_name, collected_at DESC);
CREATE INDEX idx_performance_metrics_tenant_collected ON public.performance_metrics USING btree (tenant_id, collected_at DESC);


-- public.permissions definition

-- Drop table

-- DROP TABLE public.permissions;

CREATE TABLE public.permissions (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	"name" varchar(255) NOT NULL,
	description varchar(255) NULL,
	resource_type varchar(50) NOT NULL,
	"action" varchar(50) NOT NULL,
	is_system bool DEFAULT false NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT permissions_pkey PRIMARY KEY (id),
	CONSTRAINT uq_name UNIQUE (name)
);


-- public.policies definition

-- Drop table

-- DROP TABLE public.policies;

CREATE TABLE public.policies (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	"name" text NOT NULL,
	rules jsonb NULL,
	start_date timestamp NULL,
	end_date timestamp NULL,
	schedule jsonb NULL,
	location_rules jsonb NULL,
	priority int4 DEFAULT 0 NOT NULL,
	active bool DEFAULT true NOT NULL,
	created_at timestamp DEFAULT now() NOT NULL,
	updated_at timestamp DEFAULT now() NOT NULL,
	CONSTRAINT policies_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_policies_active_priority ON public.policies USING btree (active, priority DESC) WHERE (active = true);


-- public.policy_evaluation definition

-- Drop table

-- DROP TABLE public.policy_evaluation;

CREATE TABLE public.policy_evaluation (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	run_id uuid NOT NULL,
	policy_id text NOT NULL,
	policy_version int4 NOT NULL,
	evaluated_at timestamptz DEFAULT now() NOT NULL,
	"result" jsonb NOT NULL,
	decision text NOT NULL,
	exit_code int4 NOT NULL,
	actor text NULL,
	context jsonb NULL,
	CONSTRAINT policy_evaluation_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_policy_evaluation_run_id ON public.policy_evaluation USING btree (run_id);
CREATE INDEX policy_evaluation_run_id_idx ON public.policy_evaluation USING btree (run_id);


-- public.policy_set definition

-- Drop table

-- DROP TABLE public.policy_set;

CREATE TABLE public.policy_set (
	id text NOT NULL,
	"version" int4 NOT NULL,
	title text NULL,
	"source" text NULL,
	"owner" text NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	spec jsonb NOT NULL,
	active bool DEFAULT true NOT NULL,
	CONSTRAINT policy_set_pkey PRIMARY KEY (id)
);


-- public.policy_version_history definition

-- Drop table

-- DROP TABLE public.policy_version_history;

CREATE TABLE public.policy_version_history (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	policy_id text NOT NULL,
	"version" int4 NOT NULL,
	spec jsonb NOT NULL,
	author text NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	change_summary text NULL,
	CONSTRAINT policy_version_history_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_policy_version_history_policy_id_version ON public.policy_version_history USING btree (policy_id, version);
CREATE INDEX policy_version_history_policy_id_version_idx ON public.policy_version_history USING btree (policy_id, version);


-- public.pop_dashboards definition

-- Drop table

-- DROP TABLE public.pop_dashboards;

CREATE TABLE public.pop_dashboards (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	"name" text NOT NULL,
	description text NULL,
	owner_user_id text NOT NULL,
	config jsonb NOT NULL,
	default_filters jsonb NULL,
	refresh_schedule text NULL,
	is_public bool DEFAULT false NOT NULL,
	allowed_users _text NULL,
	allowed_groups _text NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT pop_dashboards_pkey PRIMARY KEY (id)
);

-- Table Triggers

create trigger pop_dashboards_updated_at before
update
    on
    public.pop_dashboards for each row execute function update_pop_updated_at();


-- public.pop_metrics definition

-- Drop table

-- DROP TABLE public.pop_metrics;

CREATE TABLE public.pop_metrics (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	"name" text NOT NULL,
	display_name text NOT NULL,
	description text NULL,
	"domain" text NOT NULL,
	category text NOT NULL,
	metric_type text NOT NULL,
	base_query text NOT NULL,
	aggregation_function text NOT NULL,
	date_column text NOT NULL,
	value_column text NOT NULL,
	granularity text DEFAULT 'month'::text NOT NULL,
	comparison_periods jsonb DEFAULT '["previous_period", "year_over_year"]'::jsonb NOT NULL,
	owner_user_id text NOT NULL,
	steward_group text NOT NULL,
	data_source text NOT NULL,
	schema_name text NOT NULL,
	table_name text NOT NULL,
	sla_freshness_hours int4 DEFAULT 24 NOT NULL,
	sla_completeness_threshold numeric(5, 2) DEFAULT 0.95 NOT NULL,
	data_quality_checks jsonb NULL,
	status text DEFAULT 'draft'::text NOT NULL,
	golden_path bool DEFAULT false NOT NULL,
	"version" int4 DEFAULT 1 NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	created_by text NOT NULL,
	updated_by text NULL,
	CONSTRAINT pop_metrics_name_version_key UNIQUE (name, version),
	CONSTRAINT pop_metrics_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_pop_metrics_domain ON public.pop_metrics USING btree (domain);
CREATE INDEX idx_pop_metrics_golden_path ON public.pop_metrics USING btree (golden_path);
CREATE INDEX idx_pop_metrics_owner ON public.pop_metrics USING btree (owner_user_id);
CREATE INDEX idx_pop_metrics_status ON public.pop_metrics USING btree (status);

-- Table Triggers

create trigger pop_metrics_updated_at before
update
    on
    public.pop_metrics for each row execute function update_pop_updated_at();


-- public.prepared_statement_metrics definition

-- Drop table

-- DROP TABLE public.prepared_statement_metrics;

CREATE TABLE public.prepared_statement_metrics (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	query_hash varchar(64) NOT NULL,
	query_text text NOT NULL,
	execution_count int8 DEFAULT 0 NULL,
	total_time_ms int8 DEFAULT 0 NULL,
	avg_time_ms numeric(10, 2) DEFAULT 0 NULL,
	last_executed timestamptz NULL,
	created_at timestamptz DEFAULT now() NULL,
	CONSTRAINT prepared_statement_metrics_pkey PRIMARY KEY (id),
	CONSTRAINT prepared_statement_metrics_query_hash_key UNIQUE (query_hash)
);


-- public.private_markets_bundles definition

-- Drop table

-- DROP TABLE public.private_markets_bundles;

CREATE TABLE public.private_markets_bundles (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	bundle_id varchar(100) NOT NULL,
	"name" varchar(255) NOT NULL,
	audience varchar(50) NOT NULL,
	"version" varchar(50) NOT NULL,
	modules jsonb DEFAULT '[]'::jsonb NULL,
	metrics jsonb DEFAULT '[]'::jsonb NULL,
	governance jsonb DEFAULT '{}'::jsonb NULL,
	is_active bool DEFAULT true NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT private_markets_bundles_audience_check CHECK (((audience)::text = ANY ((ARRAY['lp'::character varying, 'gp'::character varying, 'fof'::character varying])::text[]))),
	CONSTRAINT private_markets_bundles_bundle_id_key UNIQUE (bundle_id),
	CONSTRAINT private_markets_bundles_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_pm_bundles_audience ON public.private_markets_bundles USING btree (audience);


-- public.private_markets_funds definition

-- Drop table

-- DROP TABLE public.private_markets_funds;

CREATE TABLE public.private_markets_funds (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	fund_id varchar(100) NOT NULL,
	"name" varchar(255) NOT NULL,
	vintage int4 NOT NULL,
	manager varchar(255) NOT NULL,
	strategy varchar(255) NOT NULL,
	geography varchar(255) NOT NULL,
	status varchar(50) DEFAULT 'active'::character varying NULL,
	description text NULL,
	target_size numeric(20, 2) NULL,
	committed_capital numeric(20, 2) NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT private_markets_funds_fund_id_key UNIQUE (fund_id),
	CONSTRAINT private_markets_funds_pkey PRIMARY KEY (id),
	CONSTRAINT private_markets_funds_status_check CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'liquidated'::character varying, 'realizing'::character varying])::text[])))
);
CREATE INDEX idx_pm_funds_manager ON public.private_markets_funds USING btree (manager);
CREATE INDEX idx_pm_funds_strategy ON public.private_markets_funds USING btree (strategy);


-- public.private_markets_users definition

-- Drop table

-- DROP TABLE public.private_markets_users;

CREATE TABLE public.private_markets_users (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	email varchar(255) NOT NULL,
	"name" varchar(255) NOT NULL,
	"role" varchar(50) NOT NULL,
	organization varchar(255) NULL,
	permissions jsonb DEFAULT '[]'::jsonb NULL,
	is_active bool DEFAULT true NULL,
	last_login_at timestamptz NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT private_markets_users_email_key UNIQUE (email),
	CONSTRAINT private_markets_users_pkey PRIMARY KEY (id),
	CONSTRAINT private_markets_users_role_check CHECK (((role)::text = ANY ((ARRAY['lp'::character varying, 'gp'::character varying, 'fof'::character varying, 'steward'::character varying])::text[])))
);
CREATE INDEX idx_pm_users_email ON public.private_markets_users USING btree (email);
CREATE INDEX idx_pm_users_role ON public.private_markets_users USING btree (role);


-- public.rule_config_changelog definition

-- Drop table

-- DROP TABLE public.rule_config_changelog;

CREATE TABLE public.rule_config_changelog (
	id uuid NOT NULL,
	rule_id text NOT NULL,
	old_aggressiveness numeric NULL,
	new_aggressiveness numeric NULL,
	old_auto_accept bool NULL,
	new_auto_accept bool NULL,
	"scope" text NOT NULL,
	reason text NOT NULL,
	triggered_by text NOT NULL,
	triggered_at timestamp DEFAULT now() NOT NULL,
	CONSTRAINT rule_config_changelog_pkey PRIMARY KEY (id)
);


-- public.schema_migrations definition

-- Drop table

-- DROP TABLE public.schema_migrations;

CREATE TABLE public.schema_migrations (
	"version" varchar(255) NOT NULL,
	"name" varchar(255) NOT NULL,
	applied_at timestamptz DEFAULT now() NULL,
	checksum varchar(64) NULL,
	CONSTRAINT schema_migrations_pkey PRIMARY KEY (version)
);


-- public.template_registry definition

-- Drop table

-- DROP TABLE public.template_registry;

CREATE TABLE public.template_registry (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	template_name varchar(255) NULL,
	template_type varchar(100) NULL,
	description text NULL,
	template_data jsonb NULL,
	"version" varchar(50) DEFAULT '1.0.0'::character varying NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	created_by varchar(255) NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	node_id varchar(255) NOT NULL,
	node_type varchar(100) NULL,
	"domain" varchar(100) NULL,
	category varchar(100) NULL,
	subcategory varchar(100) NULL,
	calc_type varchar(100) NULL,
	tags _text NULL,
	lineage _text NULL,
	status varchar(50) DEFAULT 'draft'::character varying NULL,
	schema_hash varchar(255) NULL,
	"template" jsonb NULL,
	"owner" varchar(255) NULL,
	CONSTRAINT template_registry_node_id_key UNIQUE (node_id),
	CONSTRAINT template_registry_pkey PRIMARY KEY (id),
	CONSTRAINT template_registry_template_name_key UNIQUE (template_name)
);
CREATE INDEX idx_template_registry_active ON public.template_registry USING btree (is_active);
CREATE INDEX idx_template_registry_category ON public.template_registry USING btree (category);
CREATE INDEX idx_template_registry_domain ON public.template_registry USING btree (domain);
CREATE INDEX idx_template_registry_node_id ON public.template_registry USING btree (node_id);
CREATE INDEX idx_template_registry_owner ON public.template_registry USING btree (owner);
CREATE INDEX idx_template_registry_status ON public.template_registry USING btree (status);
CREATE INDEX idx_template_registry_tags ON public.template_registry USING gin (tags);
CREATE INDEX idx_template_registry_type ON public.template_registry USING btree (template_type);

-- Table Triggers

create trigger update_template_registry_updated_at before
update
    on
    public.template_registry for each row execute function update_template_registry_updated_at();


-- public.template_versions definition

-- Drop table

-- DROP TABLE public.template_versions;

CREATE TABLE public.template_versions (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	node_id varchar(255) NOT NULL,
	"version" varchar(50) NOT NULL,
	schema_hash varchar(255) NULL,
	"template" jsonb NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT template_versions_node_id_version_key UNIQUE (node_id, version),
	CONSTRAINT template_versions_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_template_versions_node_id ON public.template_versions USING btree (node_id);
CREATE INDEX idx_template_versions_version ON public.template_versions USING btree (version);


-- public.tenant_ip_whitelist definition

-- Drop table

-- DROP TABLE public.tenant_ip_whitelist;

CREATE TABLE public.tenant_ip_whitelist (
	tenant_id uuid NOT NULL,
	ip_address varchar(45) NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT tenant_ip_whitelist_pkey PRIMARY KEY (tenant_id, ip_address)
);


-- public.tenants definition

-- Drop table

-- DROP TABLE public.tenants;

CREATE TABLE public.tenants (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	"name" varchar(255) NULL,
	is_active bool DEFAULT true NOT NULL,
	tenant_code varchar(255) NULL,
	display_name varchar(255) NOT NULL,
	description text NULL,
	status varchar(50) DEFAULT 'active'::character varying NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	gold_copy bool DEFAULT false NULL,
	CONSTRAINT tenants_pkey PRIMARY KEY (id),
	CONSTRAINT tenants_unique UNIQUE (name),
	CONSTRAINT tenants_unique_1 UNIQUE (tenant_code)
);
CREATE INDEX idx_tenants_created ON public.tenants USING btree (created_at DESC);
CREATE INDEX idx_tenants_name ON public.tenants USING btree (name);

-- Table Triggers

create trigger update_tenants_updated_at before
update
    on
    public.tenants for each row execute function update_timestamp();


-- public.user_engagement_profiles definition

-- Drop table

-- DROP TABLE public.user_engagement_profiles;

CREATE TABLE public.user_engagement_profiles (
	user_id varchar(255) NOT NULL,
	total_notifications int4 DEFAULT 0 NULL,
	opened_notifications int4 DEFAULT 0 NULL,
	clicked_notifications int4 DEFAULT 0 NULL,
	dismissed_notifications int4 DEFAULT 0 NULL,
	avg_open_rate numeric(5, 4) DEFAULT 0 NULL,
	avg_click_rate numeric(5, 4) DEFAULT 0 NULL,
	last_activity timestamptz DEFAULT now() NULL,
	engagement_score numeric(5, 4) DEFAULT 0 NULL,
	segment varchar(100) DEFAULT 'new_user'::character varying NULL,
	preferred_channels _text DEFAULT ARRAY['in_app'::text] NULL,
	preferred_times _text DEFAULT ARRAY['morning'::text, 'afternoon'::text] NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT user_engagement_profiles_pkey PRIMARY KEY (user_id)
);
CREATE INDEX idx_user_engagement_profiles_score ON public.user_engagement_profiles USING btree (engagement_score);
CREATE INDEX idx_user_engagement_profiles_segment ON public.user_engagement_profiles USING btree (segment);

-- Table Triggers

create trigger update_user_engagement_profiles_updated_at before
update
    on
    public.user_engagement_profiles for each row execute function update_updated_at_column();


-- public.user_notification_preferences definition

-- Drop table

-- DROP TABLE public.user_notification_preferences;

CREATE TABLE public.user_notification_preferences (
	user_id varchar(255) NOT NULL,
	email_enabled bool DEFAULT true NULL,
	sms_enabled bool DEFAULT false NULL,
	push_enabled bool DEFAULT true NULL,
	in_app_enabled bool DEFAULT true NULL,
	quiet_hours_start time NULL,
	quiet_hours_end time NULL,
	timezone varchar(100) DEFAULT 'UTC'::character varying NULL,
	channel_preferences jsonb DEFAULT '{}'::jsonb NULL,
	type_preferences jsonb DEFAULT '{}'::jsonb NULL,
	frequency_preferences jsonb DEFAULT '{}'::jsonb NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT user_notification_preferences_pkey PRIMARY KEY (user_id)
);

-- Table Triggers

create trigger update_user_notification_preferences_updated_at before
update
    on
    public.user_notification_preferences for each row execute function update_updated_at_column();


-- public.user_sessions definition

-- Drop table

-- DROP TABLE public.user_sessions;

CREATE TABLE public.user_sessions (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	user_id varchar(255) NOT NULL,
	tenant_id varchar(255) NULL,
	session_id varchar(255) NOT NULL,
	ip_address inet NULL,
	user_agent text NULL,
	login_time timestamptz DEFAULT now() NOT NULL,
	logout_time timestamptz NULL,
	last_activity timestamptz DEFAULT now() NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT user_sessions_pkey PRIMARY KEY (id),
	CONSTRAINT user_sessions_session_id_key UNIQUE (session_id)
);
CREATE INDEX idx_user_sessions_active ON public.user_sessions USING btree (is_active) WHERE (is_active = true);
CREATE INDEX idx_user_sessions_session_id ON public.user_sessions USING btree (session_id);
CREATE INDEX idx_user_sessions_user_id ON public.user_sessions USING btree (user_id);


-- public.users definition

-- Drop table

-- DROP TABLE public.users;

CREATE TABLE public.users (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	username varchar(255) NOT NULL,
	password_hash varchar(255) NULL,
	email varchar(255) NULL,
	firstname varchar(255) NULL,
	lastname varchar(255) NULL,
	tenant_id uuid NULL,
	first_name varchar(255) NULL,
	last_name varchar(255) NULL,
	status varchar(50) DEFAULT 'active'::character varying NOT NULL,
	last_login timestamptz NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	CONSTRAINT users_pkey PRIMARY KEY (id),
	CONSTRAINT users_username_key UNIQUE (username)
);

-- Table Triggers

create trigger update_users_updated_at before
update
    on
    public.users for each row execute function update_timestamp();


-- public.active_requests definition

-- Drop table

-- DROP TABLE public.active_requests;

CREATE TABLE public.active_requests (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	request_id varchar(255) NOT NULL,
	resource_type varchar(50) NOT NULL,
	resource_id uuid NOT NULL,
	user_id uuid NOT NULL,
	start_time timestamptz DEFAULT now() NOT NULL,
	status varchar(50) DEFAULT 'running'::character varying NOT NULL,
	progress int4 NULL,
	CONSTRAINT active_requests_pkey PRIMARY KEY (id),
	CONSTRAINT active_requests_status_check CHECK (((status)::text = ANY (ARRAY[('running'::character varying)::text, ('completed'::character varying)::text, ('failed'::character varying)::text, ('cancelled'::character varying)::text]))),
	CONSTRAINT active_requests_tenant_request_key UNIQUE (tenant_id, request_id),
	CONSTRAINT active_requests_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
	CONSTRAINT active_requests_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id)
);
CREATE INDEX idx_active_requests_status ON public.active_requests USING btree (tenant_id, status);
CREATE INDEX idx_active_requests_tenant ON public.active_requests USING btree (tenant_id);


-- public.api_groups definition

-- Drop table

-- DROP TABLE public.api_groups;

CREATE TABLE public.api_groups (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(100) NOT NULL,
	display_name varchar(200) NULL,
	description text NULL,
	parent_group_id uuid NULL,
	is_active bool DEFAULT true NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT api_groups_pkey PRIMARY KEY (id),
	CONSTRAINT api_groups_tenant_id_name_key UNIQUE (tenant_id, name),
	CONSTRAINT api_groups_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT api_groups_parent_group_id_fkey FOREIGN KEY (parent_group_id) REFERENCES public.api_groups(id),
	CONSTRAINT api_groups_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id),
	CONSTRAINT api_groups_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_api_groups_tenant_id ON public.api_groups USING btree (tenant_id);


-- public.api_keys definition

-- Drop table

-- DROP TABLE public.api_keys;

CREATE TABLE public.api_keys (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	key_hash varchar(255) NOT NULL,
	description text NULL,
	permissions jsonb NOT NULL,
	expires_at timestamptz NULL,
	is_active bool DEFAULT true NOT NULL,
	created_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	last_used_at timestamptz NULL,
	CONSTRAINT api_keys_key_hash_key UNIQUE (key_hash),
	CONSTRAINT api_keys_pkey PRIMARY KEY (id),
	CONSTRAINT api_keys_tenant_name_key UNIQUE (tenant_id, name),
	CONSTRAINT api_keys_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT api_keys_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_api_keys_expires ON public.api_keys USING btree (tenant_id, expires_at, is_active);
CREATE INDEX idx_api_keys_permissions_gin ON public.api_keys USING gin (permissions);
CREATE INDEX idx_api_keys_tenant_active ON public.api_keys USING btree (tenant_id, is_active);
CREATE INDEX idx_api_keys_tenant_id ON public.api_keys USING btree (tenant_id);

-- Table Triggers

create trigger update_api_keys_updated_at before
update
    on
    public.api_keys for each row execute function update_timestamp();


-- public.api_workflow_approvals definition

-- Drop table

-- DROP TABLE public.api_workflow_approvals;

CREATE TABLE public.api_workflow_approvals (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_id uuid NOT NULL,
	request_id varchar(50) NOT NULL,
	request_type varchar(50) NOT NULL,
	requested_by uuid NOT NULL,
	status varchar(20) NOT NULL,
	resource_id uuid NULL,
	resource_type varchar(50) NULL,
	details jsonb NULL,
	approver_id uuid NULL,
	approval_step int4 DEFAULT 1 NULL,
	"comments" text NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT api_workflow_approvals_pkey PRIMARY KEY (id),
	CONSTRAINT api_workflow_approvals_status_check CHECK (((status)::text = ANY (ARRAY[('PENDING'::character varying)::text, ('APPROVED'::character varying)::text, ('REJECTED'::character varying)::text]))),
	CONSTRAINT api_workflow_approvals_approver_id_fkey FOREIGN KEY (approver_id) REFERENCES public.users(id),
	CONSTRAINT api_workflow_approvals_requested_by_fkey FOREIGN KEY (requested_by) REFERENCES public.users(id),
	CONSTRAINT api_workflow_approvals_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id)
);
CREATE INDEX idx_api_workflow_approvals_tenant_id_status ON public.api_workflow_approvals USING btree (tenant_id, status);


-- public.apis definition

-- Drop table

-- DROP TABLE public.apis;

CREATE TABLE public.apis (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	group_id uuid NULL,
	cloned_from_id uuid NULL,
	"name" varchar(255) NOT NULL,
	display_name varchar(255) NOT NULL,
	description text NULL,
	base_path varchar(255) NULL,
	api_type varchar(50) NOT NULL,
	visibility varchar(50) DEFAULT 'private'::character varying NOT NULL,
	"version" varchar(50) DEFAULT '1.0.0'::character varying NOT NULL,
	status varchar(50) DEFAULT 'draft'::character varying NOT NULL,
	specification jsonb NULL,
	is_core bool DEFAULT false NOT NULL,
	created_at timestamptz NOT NULL,
	updated_at timestamptz NOT NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	CONSTRAINT apis_api_type_check CHECK (((api_type)::text = ANY ((ARRAY['REST'::character varying, 'GraphQL'::character varying, 'OpenAPI'::character varying])::text[]))),
	CONSTRAINT apis_pkey PRIMARY KEY (id),
	CONSTRAINT apis_status_check CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'active'::character varying, 'deprecated'::character varying, 'retired'::character varying])::text[]))),
	CONSTRAINT apis_visibility_check CHECK (((visibility)::text = ANY ((ARRAY['public'::character varying, 'private'::character varying, 'tenant-specific'::character varying])::text[]))),
	CONSTRAINT apis_cloned_from_id_fkey FOREIGN KEY (cloned_from_id) REFERENCES public.apis(id) ON DELETE SET NULL,
	CONSTRAINT apis_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.api_groups(id) ON DELETE SET NULL,
	CONSTRAINT apis_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_apis_group_id ON public.apis USING btree (group_id);
CREATE INDEX idx_apis_name ON public.apis USING btree (name);
CREATE INDEX idx_apis_tenant_id ON public.apis USING btree (tenant_id);


-- public.asset definition

-- Drop table

-- DROP TABLE public.asset;

CREATE TABLE public.asset (
	id uuid NOT NULL,
	tenant_id uuid NULL,
	"name" text NOT NULL,
	asset_type text NOT NULL,
	"domain" text NOT NULL,
	certified bool DEFAULT false NOT NULL,
	sensitivity text DEFAULT 'medium'::text NOT NULL,
	created_at timestamp DEFAULT now() NOT NULL,
	CONSTRAINT asset_pkey PRIMARY KEY (id),
	CONSTRAINT asset_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_asset_cert ON public.asset USING btree (tenant_id, certified);
CREATE INDEX idx_asset_created ON public.asset USING btree (created_at DESC);
CREATE INDEX idx_asset_sensitivity ON public.asset USING btree (sensitivity);
CREATE INDEX idx_asset_tenant_domain ON public.asset USING btree (tenant_id, domain);
CREATE INDEX idx_asset_type ON public.asset USING btree (asset_type);


-- public.broker_apis definition

-- Drop table

-- DROP TABLE public.broker_apis;

CREATE TABLE public.broker_apis (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	api_id uuid NOT NULL,
	api_name varchar(255) NOT NULL,
	broker_id uuid NOT NULL,
	broker_name varchar(255) NOT NULL,
	status varchar(50) NOT NULL,
	created_at timestamp NOT NULL,
	updated_at timestamp NOT NULL,
	CONSTRAINT broker_apis_pkey PRIMARY KEY (id),
	CONSTRAINT broker_apis_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_broker_apis_api_id ON public.broker_apis USING btree (api_id);
CREATE INDEX idx_broker_apis_tenant_id ON public.broker_apis USING btree (tenant_id);


-- public.broker_events definition

-- Drop table

-- DROP TABLE public.broker_events;

CREATE TABLE public.broker_events (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	broker_id uuid NOT NULL,
	broker_name varchar(255) NOT NULL,
	event_name varchar(255) NOT NULL,
	event_description text NULL,
	"schema" jsonb NULL,
	created_at timestamp NOT NULL,
	updated_at timestamp NOT NULL,
	CONSTRAINT broker_events_pkey PRIMARY KEY (id),
	CONSTRAINT broker_events_tenant_id_event_name_key UNIQUE (tenant_id, event_name),
	CONSTRAINT broker_events_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_broker_events_tenant_id ON public.broker_events USING btree (tenant_id);


-- public.customers definition

-- Drop table

-- DROP TABLE public.customers;

CREATE TABLE public.customers (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	email varchar(255) NOT NULL,
	phone varchar(50) NULL,
	address text NULL,
	status varchar(50) DEFAULT 'active'::character varying NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamp NOT NULL,
	updated_at timestamp NOT NULL,
	CONSTRAINT customers_pkey PRIMARY KEY (id),
	CONSTRAINT customers_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);


-- public.drift_log_entries definition

-- Drop table

-- DROP TABLE public.drift_log_entries;

CREATE TABLE public.drift_log_entries (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	report_id uuid NULL,
	severity text NULL,
	qualified_path text NOT NULL,
	explanation text NOT NULL,
	CONSTRAINT drift_log_entries_pkey PRIMARY KEY (id),
	CONSTRAINT drift_log_entries_severity_check CHECK ((severity = ANY (ARRAY['breaking'::text, 'medium'::text, 'low'::text]))),
	CONSTRAINT drift_log_entries_report_id_fkey FOREIGN KEY (report_id) REFERENCES public.drift_reports(id) ON DELETE CASCADE
);


-- public.event_subscriptions definition

-- Drop table

-- DROP TABLE public.event_subscriptions;

CREATE TABLE public.event_subscriptions (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	event_id uuid NOT NULL,
	event_name varchar(255) NOT NULL,
	callback_url text NOT NULL,
	status varchar(50) NOT NULL,
	created_at timestamp NOT NULL,
	updated_at timestamp NOT NULL,
	CONSTRAINT event_subscriptions_pkey PRIMARY KEY (id),
	CONSTRAINT event_subscriptions_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.broker_events(id) ON DELETE CASCADE,
	CONSTRAINT event_subscriptions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_event_subscriptions_event_id ON public.event_subscriptions USING btree (event_id);
CREATE INDEX idx_event_subscriptions_tenant_id ON public.event_subscriptions USING btree (tenant_id);


-- public.exposed_apis definition

-- Drop table

-- DROP TABLE public.exposed_apis;

CREATE TABLE public.exposed_apis (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	api_id uuid NOT NULL,
	broker_api_id varchar(255) NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	is_active bool DEFAULT true NOT NULL,
	exposed_at timestamptz NOT NULL,
	last_synced_at timestamptz NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz NOT NULL,
	updated_at timestamptz NOT NULL,
	CONSTRAINT exposed_apis_pkey PRIMARY KEY (id),
	CONSTRAINT exposed_apis_tenant_id_api_id_key UNIQUE (tenant_id, api_id),
	CONSTRAINT exposed_apis_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.apis(id) ON DELETE CASCADE,
	CONSTRAINT exposed_apis_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_exposed_apis_tenant_id ON public.exposed_apis USING btree (tenant_id);


-- public.integration_audit_logs definition

-- Drop table

-- DROP TABLE public.integration_audit_logs;

CREATE TABLE public.integration_audit_logs (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	"type" varchar(50) NOT NULL,
	"action" varchar(50) NOT NULL,
	resource_type varchar(50) NOT NULL,
	resource_id uuid NOT NULL,
	user_id uuid NOT NULL,
	"timestamp" timestamptz DEFAULT now() NOT NULL,
	duration int8 NULL,
	status varchar(50) NOT NULL,
	error_message text NULL,
	request_data jsonb NULL,
	response_data jsonb NULL,
	ip_address varchar(50) NULL,
	user_agent text NULL,
	CONSTRAINT integration_audit_logs_pkey PRIMARY KEY (id),
	CONSTRAINT integration_audit_logs_status_check CHECK (((status)::text = ANY (ARRAY[('success'::character varying)::text, ('error'::character varying)::text, ('warning'::character varying)::text, ('info'::character varying)::text]))),
	CONSTRAINT integration_audit_logs_type_check CHECK (((type)::text = ANY (ARRAY[('integration_execution'::character varying)::text, ('business_rule_execution'::character varying)::text, ('api_request'::character varying)::text, ('security_event'::character varying)::text]))),
	CONSTRAINT integration_audit_logs_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
	CONSTRAINT integration_audit_logs_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id)
);
CREATE INDEX idx_integration_audit_logs_request_gin ON public.integration_audit_logs USING gin (request_data);
CREATE INDEX idx_integration_audit_logs_resource ON public.integration_audit_logs USING btree (tenant_id, resource_type, resource_id);
CREATE INDEX idx_integration_audit_logs_response_gin ON public.integration_audit_logs USING gin (response_data);
CREATE INDEX idx_integration_audit_logs_status ON public.integration_audit_logs USING btree (tenant_id, status);
CREATE INDEX idx_integration_audit_logs_tenant ON public.integration_audit_logs USING btree (tenant_id);
CREATE INDEX idx_integration_audit_logs_timestamp ON public.integration_audit_logs USING btree (tenant_id, "timestamp");
CREATE INDEX idx_integration_audit_logs_type_action ON public.integration_audit_logs USING btree (tenant_id, type, action);
CREATE INDEX idx_integration_audit_logs_user ON public.integration_audit_logs USING btree (tenant_id, user_id);


-- public.integration_configs definition

-- Drop table

-- DROP TABLE public.integration_configs;

CREATE TABLE public.integration_configs (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	event_id uuid NOT NULL,
	broker_endpoint varchar(1024) NOT NULL,
	headers text NULL,
	is_active bool DEFAULT true NOT NULL,
	retry_count int4 DEFAULT 3 NOT NULL,
	retry_delay int4 DEFAULT 60 NOT NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT integration_configs_pkey PRIMARY KEY (id),
	CONSTRAINT integration_configs_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT integration_configs_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_integration_configs_tenant ON public.integration_configs USING btree (tenant_id);
CREATE INDEX idx_integration_configs_tenant_event ON public.integration_configs USING btree (tenant_id, event_id);


-- public.integrations definition

-- Drop table

-- DROP TABLE public.integrations;

CREATE TABLE public.integrations (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	display_name varchar(255) NOT NULL,
	description text NULL,
	"type" varchar(50) NOT NULL,
	endpoint varchar(1024) NOT NULL,
	methods jsonb NOT NULL,
	required_permissions jsonb NOT NULL,
	rate_limits jsonb NOT NULL,
	timeout int4 NOT NULL,
	retry_policy jsonb NOT NULL,
	config jsonb NULL,
	status varchar(50) DEFAULT 'active'::character varying NOT NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT integrations_pkey PRIMARY KEY (id),
	CONSTRAINT integrations_status_check CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('inactive'::character varying)::text, ('draft'::character varying)::text, ('deprecated'::character varying)::text]))),
	CONSTRAINT integrations_tenant_name_key UNIQUE (tenant_id, name),
	CONSTRAINT integrations_type_check CHECK (((type)::text = ANY (ARRAY[('rest'::character varying)::text, ('kafka'::character varying)::text, ('azure'::character varying)::text]))),
	CONSTRAINT integrations_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT integrations_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
	CONSTRAINT integrations_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_integrations_config_gin ON public.integrations USING gin (config);
CREATE INDEX idx_integrations_description_trgm ON public.integrations USING gin (description gin_trgm_ops);
CREATE INDEX idx_integrations_methods_gin ON public.integrations USING gin (methods);
CREATE INDEX idx_integrations_name ON public.integrations USING btree (tenant_id, name);
CREATE INDEX idx_integrations_required_permissions_gin ON public.integrations USING gin (required_permissions);
CREATE INDEX idx_integrations_status ON public.integrations USING btree (tenant_id, status);
CREATE INDEX idx_integrations_tenant_type ON public.integrations USING btree (tenant_id, type);

-- Table Triggers

create trigger update_integration_timestamp before
update
    on
    public.integrations for each row execute function update_timestamp();
create trigger update_integrations_updated_at before
update
    on
    public.integrations for each row execute function update_timestamp();


-- public.ip_whitelist definition

-- Drop table

-- DROP TABLE public.ip_whitelist;

CREATE TABLE public.ip_whitelist (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	ip_address varchar(50) NOT NULL,
	description text NULL,
	created_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT ip_whitelist_pkey PRIMARY KEY (id),
	CONSTRAINT ip_whitelist_tenant_ip_key UNIQUE (tenant_id, ip_address),
	CONSTRAINT ip_whitelist_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT ip_whitelist_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_ip_whitelist_tenant ON public.ip_whitelist USING btree (tenant_id);


-- public.ip_whitelist_entries definition

-- Drop table

-- DROP TABLE public.ip_whitelist_entries;

CREATE TABLE public.ip_whitelist_entries (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_id uuid NULL,
	ip_address text NOT NULL,
	description text NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT ip_whitelist_entries_pkey PRIMARY KEY (id),
	CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_ip_whitelist_global_ip_unique ON public.ip_whitelist_entries USING btree (ip_address) WHERE (tenant_id IS NULL);
CREATE INDEX idx_ip_whitelist_tenant_id ON public.ip_whitelist_entries USING btree (tenant_id);
CREATE UNIQUE INDEX idx_ip_whitelist_tenant_ip_unique ON public.ip_whitelist_entries USING btree (tenant_id, ip_address) WHERE (tenant_id IS NOT NULL);


-- public.message_targets definition

-- Drop table

-- DROP TABLE public.message_targets;

CREATE TABLE public.message_targets (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	"type" varchar(50) NOT NULL,
	webhook_url text NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	template_id uuid NULL,
	tenant_id uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT message_targets_pkey PRIMARY KEY (id),
	CONSTRAINT fk_template FOREIGN KEY (template_id) REFERENCES public.message_templates(id) ON DELETE SET NULL
);
CREATE INDEX idx_message_targets_tenant_id ON public.message_targets USING btree (tenant_id);


-- public.metadata_tables definition

-- Drop table

-- DROP TABLE public.metadata_tables;

CREATE TABLE public.metadata_tables (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	datasource_id uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	display_name varchar(255) NOT NULL,
	description text NULL,
	schema_name varchar(255) DEFAULT 'public'::character varying NOT NULL,
	is_view bool DEFAULT false NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	is_system bool DEFAULT false NOT NULL,
	CONSTRAINT metadata_tables_pkey PRIMARY KEY (id),
	CONSTRAINT metadata_tables_tenant_name_key UNIQUE (tenant_id, name),
	CONSTRAINT metadata_tables_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT metadata_tables_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
	CONSTRAINT metadata_tables_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_metadata_tables_datasource ON public.metadata_tables USING btree (datasource_id);
CREATE INDEX idx_metadata_tables_description_trgm ON public.metadata_tables USING gin (description gin_trgm_ops);
CREATE INDEX idx_metadata_tables_name ON public.metadata_tables USING btree (tenant_id, name);
CREATE INDEX idx_metadata_tables_tenant_active ON public.metadata_tables USING btree (tenant_id, is_active);

-- Table Triggers

create trigger update_metadata_table_timestamp before
update
    on
    public.metadata_tables for each row execute function update_timestamp();
create trigger update_metadata_tables_updated_at before
update
    on
    public.metadata_tables for each row execute function update_timestamp();


-- public.orders definition

-- Drop table

-- DROP TABLE public.orders;

CREATE TABLE public.orders (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	customer_id uuid NOT NULL,
	order_date timestamp NOT NULL,
	total_amount numeric(10, 2) NOT NULL,
	status varchar(50) DEFAULT 'pending'::character varying NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamp NOT NULL,
	updated_at timestamp NOT NULL,
	CONSTRAINT orders_pkey PRIMARY KEY (id),
	CONSTRAINT orders_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE,
	CONSTRAINT orders_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);


-- public.policy_violation definition

-- Drop table

-- DROP TABLE public.policy_violation;

CREATE TABLE public.policy_violation (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	evaluation_id uuid NOT NULL,
	rule_id text NOT NULL,
	code text NOT NULL,
	severity text NOT NULL,
	on_violation text NOT NULL,
	message text NOT NULL,
	change_id text NOT NULL,
	object_fqn text NULL,
	details jsonb NULL,
	"explain" jsonb NULL,
	CONSTRAINT policy_violation_pkey PRIMARY KEY (id),
	CONSTRAINT policy_violation_evaluation_id_fkey FOREIGN KEY (evaluation_id) REFERENCES public.policy_evaluation(id) ON DELETE CASCADE
);
CREATE INDEX idx_policy_violation_evaluation_id ON public.policy_violation USING btree (evaluation_id);
CREATE INDEX policy_violation_evaluation_id_idx ON public.policy_violation USING btree (evaluation_id);


-- public.pop_computations definition

-- Drop table

-- DROP TABLE public.pop_computations;

CREATE TABLE public.pop_computations (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	metric_id uuid NULL,
	period_start date NOT NULL,
	period_end date NOT NULL,
	granularity text NOT NULL,
	period_label text NOT NULL,
	current_value numeric(20, 6) NULL,
	previous_value numeric(20, 6) NULL,
	delta numeric(20, 6) NULL,
	percent_change numeric(10, 4) NULL,
	record_count int4 NULL,
	last_updated timestamptz DEFAULT now() NULL,
	computation_status text DEFAULT 'success'::text NOT NULL,
	CONSTRAINT pop_computations_metric_id_period_start_period_end_granular_key UNIQUE (metric_id, period_start, period_end, granularity),
	CONSTRAINT pop_computations_pkey PRIMARY KEY (id),
	CONSTRAINT pop_computations_metric_id_fkey FOREIGN KEY (metric_id) REFERENCES public.pop_metrics(id) ON DELETE CASCADE
);
CREATE INDEX idx_pop_computations_granularity ON public.pop_computations USING btree (granularity);
CREATE INDEX idx_pop_computations_metric_period ON public.pop_computations USING btree (metric_id, period_start, period_end);


-- public.pop_dashboard_widgets definition

-- Drop table

-- DROP TABLE public.pop_dashboard_widgets;

CREATE TABLE public.pop_dashboard_widgets (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	dashboard_id uuid NULL,
	widget_type text NOT NULL,
	title text NOT NULL,
	"position" jsonb NOT NULL,
	config jsonb NOT NULL,
	metric_ids _uuid NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT pop_dashboard_widgets_pkey PRIMARY KEY (id),
	CONSTRAINT pop_dashboard_widgets_dashboard_id_fkey FOREIGN KEY (dashboard_id) REFERENCES public.pop_dashboards(id) ON DELETE CASCADE
);


-- public.pop_metric_tags definition

-- Drop table

-- DROP TABLE public.pop_metric_tags;

CREATE TABLE public.pop_metric_tags (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	metric_id uuid NULL,
	tag_name text NOT NULL,
	tag_value text NULL,
	created_at timestamptz DEFAULT now() NULL,
	CONSTRAINT pop_metric_tags_metric_id_tag_name_key UNIQUE (metric_id, tag_name),
	CONSTRAINT pop_metric_tags_pkey PRIMARY KEY (id),
	CONSTRAINT pop_metric_tags_metric_id_fkey FOREIGN KEY (metric_id) REFERENCES public.pop_metrics(id) ON DELETE CASCADE
);


-- public.pop_steward_reviews definition

-- Drop table

-- DROP TABLE public.pop_steward_reviews;

CREATE TABLE public.pop_steward_reviews (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	metric_id uuid NULL,
	review_period_start date NOT NULL,
	review_period_end date NOT NULL,
	reviewer_user_id text NOT NULL,
	review_type text NOT NULL,
	overall_rating text NULL,
	review_notes text NULL,
	action_items jsonb NULL,
	status text DEFAULT 'in_progress'::text NOT NULL,
	due_date date NULL,
	completed_at timestamptz NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT pop_steward_reviews_pkey PRIMARY KEY (id),
	CONSTRAINT pop_steward_reviews_metric_id_fkey FOREIGN KEY (metric_id) REFERENCES public.pop_metrics(id) ON DELETE CASCADE
);
CREATE INDEX idx_pop_steward_reviews_metric ON public.pop_steward_reviews USING btree (metric_id);
CREATE INDEX idx_pop_steward_reviews_reviewer ON public.pop_steward_reviews USING btree (reviewer_user_id);
CREATE INDEX idx_pop_steward_reviews_status ON public.pop_steward_reviews USING btree (status);

-- Table Triggers

create trigger pop_steward_reviews_updated_at before
update
    on
    public.pop_steward_reviews for each row execute function update_pop_updated_at();


-- public.private_markets_fund_metrics definition

-- Drop table

-- DROP TABLE public.private_markets_fund_metrics;

CREATE TABLE public.private_markets_fund_metrics (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	fund_id varchar(100) NOT NULL,
	as_of_date date NOT NULL,
	tvpi numeric(10, 4) NULL,
	rvpi numeric(10, 4) NULL,
	irr numeric(10, 6) NULL,
	xirr numeric(10, 6) NULL,
	pme numeric(10, 4) NULL,
	paid_in_capital numeric(20, 2) NULL,
	distributions numeric(20, 2) NULL,
	residual_value numeric(20, 2) NULL,
	nav numeric(20, 2) NULL,
	dpi numeric(10, 4) NULL,
	multiple numeric(10, 4) NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT private_markets_fund_metrics_fund_id_as_of_date_key UNIQUE (fund_id, as_of_date),
	CONSTRAINT private_markets_fund_metrics_pkey PRIMARY KEY (id),
	CONSTRAINT private_markets_fund_metrics_fund_id_fkey FOREIGN KEY (fund_id) REFERENCES public.private_markets_funds(fund_id) ON DELETE CASCADE
);
CREATE INDEX idx_pm_fund_metrics_date ON public.private_markets_fund_metrics USING btree (as_of_date);
CREATE INDEX idx_pm_fund_metrics_fund_id ON public.private_markets_fund_metrics USING btree (fund_id);


-- public.private_markets_refresh_tokens definition

-- Drop table

-- DROP TABLE public.private_markets_refresh_tokens;

CREATE TABLE public.private_markets_refresh_tokens (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	user_id uuid NOT NULL,
	"token" varchar(255) NOT NULL,
	expires_at timestamptz NOT NULL,
	is_revoked bool DEFAULT false NULL,
	created_at timestamptz DEFAULT now() NULL,
	revoked_at timestamptz NULL,
	CONSTRAINT private_markets_refresh_tokens_pkey PRIMARY KEY (id),
	CONSTRAINT private_markets_refresh_tokens_token_key UNIQUE (token),
	CONSTRAINT private_markets_refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.private_markets_users(id) ON DELETE CASCADE
);
CREATE INDEX idx_pm_refresh_tokens_expires_at ON public.private_markets_refresh_tokens USING btree (expires_at);
CREATE INDEX idx_pm_refresh_tokens_token ON public.private_markets_refresh_tokens USING btree (token);
CREATE INDEX idx_pm_refresh_tokens_user_id ON public.private_markets_refresh_tokens USING btree (user_id);


-- public.private_markets_sessions definition

-- Drop table

-- DROP TABLE public.private_markets_sessions;

CREATE TABLE public.private_markets_sessions (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	user_id uuid NOT NULL,
	session_token varchar(255) NOT NULL,
	refresh_token varchar(255) NOT NULL,
	expires_at timestamptz NOT NULL,
	refresh_expires_at timestamptz NOT NULL,
	ip_address inet NULL,
	user_agent text NULL,
	is_active bool DEFAULT true NULL,
	created_at timestamptz DEFAULT now() NULL,
	last_activity_at timestamptz DEFAULT now() NULL,
	CONSTRAINT private_markets_sessions_pkey PRIMARY KEY (id),
	CONSTRAINT private_markets_sessions_refresh_token_key UNIQUE (refresh_token),
	CONSTRAINT private_markets_sessions_session_token_key UNIQUE (session_token),
	CONSTRAINT private_markets_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.private_markets_users(id) ON DELETE CASCADE
);
CREATE INDEX idx_pm_sessions_expires_at ON public.private_markets_sessions USING btree (expires_at);
CREATE INDEX idx_pm_sessions_refresh_token ON public.private_markets_sessions USING btree (refresh_token);
CREATE INDEX idx_pm_sessions_token ON public.private_markets_sessions USING btree (session_token);
CREATE INDEX idx_pm_sessions_user_id ON public.private_markets_sessions USING btree (user_id);


-- public.private_markets_user_auth definition

-- Drop table

-- DROP TABLE public.private_markets_user_auth;

CREATE TABLE public.private_markets_user_auth (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	user_id uuid NOT NULL,
	password_hash varchar(255) NOT NULL,
	salt varchar(255) NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT private_markets_user_auth_pkey PRIMARY KEY (id),
	CONSTRAINT private_markets_user_auth_user_id_key UNIQUE (user_id),
	CONSTRAINT private_markets_user_auth_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.private_markets_users(id) ON DELETE CASCADE
);
CREATE INDEX idx_pm_user_auth_user_id ON public.private_markets_user_auth USING btree (user_id);


-- public.private_markets_user_preferences definition

-- Drop table

-- DROP TABLE public.private_markets_user_preferences;

CREATE TABLE public.private_markets_user_preferences (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	user_id uuid NOT NULL,
	bundle_id varchar(100) NOT NULL,
	dashboard_config jsonb DEFAULT '{}'::jsonb NULL,
	favorite_funds jsonb DEFAULT '[]'::jsonb NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT private_markets_user_preferences_pkey PRIMARY KEY (id),
	CONSTRAINT private_markets_user_preferences_user_id_key UNIQUE (user_id),
	CONSTRAINT private_markets_user_preferences_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.private_markets_users(id) ON DELETE CASCADE
);


-- public."role" definition

-- Drop table

-- DROP TABLE public."role";

CREATE TABLE public."role" (
	id uuid NOT NULL,
	tenant_id uuid NULL,
	"name" text NOT NULL,
	CONSTRAINT role_pkey PRIMARY KEY (id),
	CONSTRAINT role_tenant_id_name_key UNIQUE (tenant_id, name),
	CONSTRAINT role_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_role_tenant_name ON public.role USING btree (tenant_id, name);


-- public.role_claim definition

-- Drop table

-- DROP TABLE public.role_claim;

CREATE TABLE public.role_claim (
	id uuid NOT NULL,
	role_id uuid NULL,
	asset_id uuid NULL,
	"permission" text NOT NULL,
	"scope" _text DEFAULT '{}'::text[] NOT NULL,
	CONSTRAINT role_claim_pkey PRIMARY KEY (id),
	CONSTRAINT role_claim_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.asset(id) ON DELETE CASCADE,
	CONSTRAINT role_claim_role_id_fkey FOREIGN KEY (role_id) REFERENCES public."role"(id) ON DELETE CASCADE
);
CREATE INDEX idx_role_claim_asset ON public.role_claim USING btree (asset_id);
CREATE INDEX idx_role_claim_permission ON public.role_claim USING btree (permission);
CREATE INDEX idx_role_claim_role ON public.role_claim USING btree (role_id);


-- public.role_member definition

-- Drop table

-- DROP TABLE public.role_member;

CREATE TABLE public.role_member (
	role_id uuid NOT NULL,
	user_id text NOT NULL,
	tenant_id uuid NOT NULL,
	CONSTRAINT role_member_pkey PRIMARY KEY (role_id, user_id, tenant_id),
	CONSTRAINT role_member_role_id_fkey FOREIGN KEY (role_id) REFERENCES public."role"(id) ON DELETE CASCADE,
	CONSTRAINT role_member_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
	CONSTRAINT role_member_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.app_user(id) ON DELETE CASCADE
);
CREATE INDEX idx_role_member_role ON public.role_member USING btree (role_id);
CREATE INDEX idx_role_member_user ON public.role_member USING btree (user_id, tenant_id);


-- public.roles definition

-- Drop table

-- DROP TABLE public.roles;

CREATE TABLE public.roles (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	rolename varchar(255) NOT NULL,
	description varchar(255) NULL,
	is_active bool DEFAULT true NOT NULL,
	is_alpha bool DEFAULT false NOT NULL,
	tenant_id uuid NULL,
	is_system bool DEFAULT false NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT roles_pkey PRIMARY KEY (id),
	CONSTRAINT roles_unique UNIQUE (tenant_id, rolename),
	CONSTRAINT roles_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_active_roles_tenant ON public.roles USING btree (tenant_id, updated_at DESC) WHERE (is_active = true);
CREATE INDEX idx_roles_tenant_active ON public.roles USING btree (tenant_id, updated_at DESC) WHERE (is_active = true);

-- Table Triggers

create trigger update_roles_updated_at before
update
    on
    public.roles for each row execute function update_timestamp();


-- public.tenant_connections definition

-- Drop table

-- DROP TABLE public.tenant_connections;

CREATE TABLE public.tenant_connections (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	connection_name varchar(255) NOT NULL,
	database_type varchar(50) NOT NULL,
	dsn text NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT tenant_connections_pkey PRIMARY KEY (id),
	CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

-- Table Triggers

create trigger update_tenant_connections_updated_at before
update
    on
    public.tenant_connections for each row execute function update_timestamp();


-- public.tenant_instance definition

-- Drop table

-- DROP TABLE public.tenant_instance;

CREATE TABLE public.tenant_instance (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	instance_name varchar(255) NOT NULL,
	config jsonb NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	url varchar(255) NULL,
	display_name varchar(255) NOT NULL,
	description text NULL,
	status varchar(50) DEFAULT 'active'::character varying NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT tenant_instance_pkey PRIMARY KEY (id),
	CONSTRAINT tenant_instance_unique UNIQUE (tenant_id, instance_name),
	CONSTRAINT tenant_instance_unique_1 UNIQUE (url),
	CONSTRAINT tenant_instance_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

-- Table Triggers

create trigger update_tenant_instance_updated_at before
update
    on
    public.tenant_instance for each row execute function update_timestamp();


-- public.tenant_product definition

-- Drop table

-- DROP TABLE public.tenant_product;

CREATE TABLE public.tenant_product (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	datasource_id uuid NOT NULL,
	alpha_product_id uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	"version" float4 NOT NULL,
	is_active bool DEFAULT false NOT NULL,
	CONSTRAINT tenant_product_pkey PRIMARY KEY (id),
	CONSTRAINT tenant_product_uniq UNIQUE (datasource_id, alpha_product_id),
	CONSTRAINT tenant_product_product_fk FOREIGN KEY (alpha_product_id) REFERENCES public.alpha_product(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT tenant_product_tenant_instance_fk FOREIGN KEY (datasource_id) REFERENCES public.tenant_instance(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);


-- public.tenant_product_datasource definition

-- Drop table

-- DROP TABLE public.tenant_product_datasource;

CREATE TABLE public.tenant_product_datasource (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_product_id uuid NOT NULL,
	alpha_datasource_id uuid NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	config jsonb NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	source_name varchar NULL,
	chart bytea NULL,
	CONSTRAINT tenant_product_datasource_pkey PRIMARY KEY (id),
	CONSTRAINT tenant_product_datasource_source_uniq UNIQUE (tenant_product_id, source_name),
	CONSTRAINT tenant_product_datasource_alpha_datasource_fk FOREIGN KEY (alpha_datasource_id) REFERENCES public.alpha_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT tenant_product_datasource_tenant_product_fk FOREIGN KEY (tenant_product_id) REFERENCES public.tenant_product(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);


-- public.tenant_user definition

-- Drop table

-- DROP TABLE public.tenant_user;

CREATE TABLE public.tenant_user (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	user_id uuid NOT NULL,
	CONSTRAINT tenant_user_pkey PRIMARY KEY (id),
	CONSTRAINT tenant_user_unique UNIQUE (tenant_id, user_id),
	CONSTRAINT tenant_user_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT tenant_user_users_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);


-- public.user_role definition

-- Drop table

-- DROP TABLE public.user_role;

CREATE TABLE public.user_role (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	user_id uuid NOT NULL,
	role_id uuid NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	CONSTRAINT user_role_pkey PRIMARY KEY (id),
	CONSTRAINT user_role_unique UNIQUE (user_id, role_id),
	CONSTRAINT user_role_roles_fk FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT user_role_users_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);


-- public.user_tenant definition

-- Drop table

-- DROP TABLE public.user_tenant;

CREATE TABLE public.user_tenant (
	user_id text NOT NULL,
	tenant_id uuid NOT NULL,
	CONSTRAINT user_tenant_pkey PRIMARY KEY (user_id, tenant_id),
	CONSTRAINT user_tenant_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
	CONSTRAINT user_tenant_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.app_user(id) ON DELETE CASCADE
);
CREATE INDEX idx_user_tenant_tenant ON public.user_tenant USING btree (tenant_id);
CREATE INDEX idx_user_tenant_user ON public.user_tenant USING btree (user_id);


-- public.api_definitions definition

-- Drop table

-- DROP TABLE public.api_definitions;

CREATE TABLE public.api_definitions (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(100) NOT NULL,
	display_name varchar(200) NULL,
	description text NULL,
	api_type varchar(20) NOT NULL,
	"version" varchar(20) NOT NULL,
	status varchar(20) NOT NULL,
	visibility varchar(20) NOT NULL,
	base_path varchar(200) NULL,
	group_id uuid NULL,
	specification jsonb NULL,
	is_core bool DEFAULT false NULL,
	cloned_from_id uuid NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	deprecated_at timestamptz NULL,
	sunset_at timestamptz NULL,
	successor_api_id uuid NULL,
	CONSTRAINT api_definitions_api_type_check CHECK (((api_type)::text = ANY (ARRAY[('REST'::character varying)::text, ('GRAPHQL'::character varying)::text, ('OPENAPI'::character varying)::text]))),
	CONSTRAINT api_definitions_pkey PRIMARY KEY (id),
	CONSTRAINT api_definitions_status_check CHECK (((status)::text = ANY (ARRAY[('DRAFT'::character varying)::text, ('ACTIVE'::character varying)::text, ('DEPRECATED'::character varying)::text, ('RETIRED'::character varying)::text]))),
	CONSTRAINT api_definitions_tenant_id_name_version_key UNIQUE (tenant_id, name, version),
	CONSTRAINT api_definitions_visibility_check CHECK (((visibility)::text = ANY (ARRAY[('PUBLIC'::character varying)::text, ('PRIVATE'::character varying)::text, ('TENANT_SPECIFIC'::character varying)::text]))),
	CONSTRAINT api_definitions_cloned_from_id_fkey FOREIGN KEY (cloned_from_id) REFERENCES public.api_definitions(id),
	CONSTRAINT api_definitions_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT api_definitions_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.api_groups(id),
	CONSTRAINT api_definitions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id),
	CONSTRAINT api_definitions_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_api_definitions_group_id ON public.api_definitions USING btree (group_id);
CREATE INDEX idx_api_definitions_tenant_id ON public.api_definitions USING btree (tenant_id);


-- public.api_documentation definition

-- Drop table

-- DROP TABLE public.api_documentation;

CREATE TABLE public.api_documentation (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	api_id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	content_type varchar(50) DEFAULT 'MARKDOWN'::character varying NULL,
	"content" text NULL,
	"section" varchar(100) DEFAULT 'OVERVIEW'::character varying NULL,
	order_index int4 DEFAULT 0 NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT api_documentation_content_type_check CHECK (((content_type)::text = ANY (ARRAY[('MARKDOWN'::character varying)::text, ('HTML'::character varying)::text, ('ASCIIDOC'::character varying)::text]))),
	CONSTRAINT api_documentation_pkey PRIMARY KEY (id),
	CONSTRAINT api_documentation_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id) ON DELETE CASCADE,
	CONSTRAINT api_documentation_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT api_documentation_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id),
	CONSTRAINT api_documentation_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_api_documentation_api_id ON public.api_documentation USING btree (api_id);


-- public.api_endpoints definition

-- Drop table

-- DROP TABLE public.api_endpoints;

CREATE TABLE public.api_endpoints (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	api_id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	"path" varchar(200) NOT NULL,
	http_method varchar(10) NULL,
	operation_id varchar(100) NULL,
	summary varchar(200) NULL,
	description text NULL,
	request_schema jsonb NULL,
	response_schema jsonb NULL,
	deprecated bool DEFAULT false NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT api_endpoints_api_id_path_http_method_key UNIQUE (api_id, path, http_method),
	CONSTRAINT api_endpoints_http_method_check CHECK (((http_method)::text = ANY (ARRAY[('GET'::character varying)::text, ('POST'::character varying)::text, ('PUT'::character varying)::text, ('DELETE'::character varying)::text, ('PATCH'::character varying)::text, ('OPTIONS'::character varying)::text, ('HEAD'::character varying)::text]))),
	CONSTRAINT api_endpoints_pkey PRIMARY KEY (id),
	CONSTRAINT api_endpoints_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id) ON DELETE CASCADE,
	CONSTRAINT api_endpoints_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT api_endpoints_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id),
	CONSTRAINT api_endpoints_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_api_endpoints_api_id ON public.api_endpoints USING btree (api_id);


-- public.api_hooks definition

-- Drop table

-- DROP TABLE public.api_hooks;

CREATE TABLE public.api_hooks (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(100) NOT NULL,
	display_name varchar(200) NULL,
	description text NULL,
	event_type varchar(50) NOT NULL,
	api_id uuid NULL,
	endpoint_id uuid NULL,
	script text NULL,
	execution_order int4 DEFAULT 100 NULL,
	is_active bool DEFAULT true NULL,
	execution_location varchar(20) DEFAULT 'SERVER'::character varying NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT api_hooks_event_type_check CHECK (((event_type)::text = ANY ((ARRAY['pre-request'::character varying, 'post-request'::character varying, 'user.created'::character varying, 'user.updated'::character varying, 'user.deleted'::character varying])::text[]))),
	CONSTRAINT api_hooks_execution_location_check CHECK (((execution_location)::text = 'gateway'::text)),
	CONSTRAINT api_hooks_pkey PRIMARY KEY (id),
	CONSTRAINT api_hooks_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id),
	CONSTRAINT api_hooks_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT api_hooks_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES public.api_endpoints(id),
	CONSTRAINT api_hooks_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id),
	CONSTRAINT api_hooks_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_api_hooks_api_id ON public.api_hooks USING btree (api_id);
CREATE INDEX idx_api_hooks_endpoint_id ON public.api_hooks USING btree (endpoint_id);


-- public.api_metrics definition

-- Drop table

-- DROP TABLE public.api_metrics;

CREATE TABLE public.api_metrics (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	"timestamp" timestamptz DEFAULT now() NOT NULL,
	api_id uuid NULL,
	endpoint_id uuid NULL,
	response_time int4 NULL,
	status_code int4 NULL,
	request_size int4 NULL,
	response_size int4 NULL,
	client_ip varchar(45) NULL,
	user_id uuid NULL,
	error_type varchar(100) NULL,
	error_message text NULL,
	CONSTRAINT api_metrics_pkey PRIMARY KEY (id, "timestamp"),
	CONSTRAINT api_metrics_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id),
	CONSTRAINT api_metrics_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES public.api_endpoints(id),
	CONSTRAINT api_metrics_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id),
	CONSTRAINT api_metrics_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id)
)
PARTITION BY RANGE ("timestamp");
CREATE INDEX idx_api_metrics_api_id_timestamp ON ONLY public.api_metrics USING btree (api_id, "timestamp");
CREATE INDEX idx_api_metrics_tenant_id_timestamp ON ONLY public.api_metrics USING btree (tenant_id, "timestamp");


-- public.api_metrics_current_month definition

CREATE TABLE public.api_metrics_current_month PARTITION OF public.api_metrics  FOR VALUES FROM ('2025-07-01 00:00:00-04') TO ('2025-08-01 00:00:00-04');
CREATE INDEX api_metrics_current_month_api_id_timestamp_idx ON public.api_metrics_current_month USING btree (api_id, "timestamp");
CREATE INDEX api_metrics_current_month_tenant_id_timestamp_idx ON public.api_metrics_current_month USING btree (tenant_id, "timestamp");


-- public.api_metrics_next_month definition

CREATE TABLE public.api_metrics_next_month PARTITION OF public.api_metrics  FOR VALUES FROM ('2025-08-01 00:00:00-04') TO ('2025-09-01 00:00:00-04');
CREATE INDEX api_metrics_next_month_api_id_timestamp_idx ON public.api_metrics_next_month USING btree (api_id, "timestamp");
CREATE INDEX api_metrics_next_month_tenant_id_timestamp_idx ON public.api_metrics_next_month USING btree (tenant_id, "timestamp");


-- public.api_security_configs definition

-- Drop table

-- DROP TABLE public.api_security_configs;

CREATE TABLE public.api_security_configs (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	api_id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	auth_methods jsonb DEFAULT '["API_KEY"]'::jsonb NULL,
	rate_limit_requests int4 DEFAULT 60 NULL,
	rate_limit_period varchar(10) DEFAULT 'MINUTE'::character varying NULL,
	ip_whitelist _text NULL,
	require_approval bool DEFAULT false NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT api_security_configs_api_id_tenant_id_key UNIQUE (api_id, tenant_id),
	CONSTRAINT api_security_configs_pkey PRIMARY KEY (id),
	CONSTRAINT api_security_configs_rate_limit_period_check CHECK (((rate_limit_period)::text = ANY (ARRAY[('SECOND'::character varying)::text, ('MINUTE'::character varying)::text, ('HOUR'::character varying)::text, ('DAY'::character varying)::text]))),
	CONSTRAINT api_security_configs_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id) ON DELETE CASCADE,
	CONSTRAINT api_security_configs_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT api_security_configs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id),
	CONSTRAINT api_security_configs_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_api_security_configs_api_id ON public.api_security_configs USING btree (api_id);


-- public.business_rules definition

-- Drop table

-- DROP TABLE public.business_rules;

CREATE TABLE public.business_rules (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	table_id uuid NOT NULL,
	event_type varchar(50) NOT NULL,
	script text NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	execution_order int4 DEFAULT 100 NOT NULL,
	execution_location varchar(50) DEFAULT 'server'::character varying NOT NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	"version" int4 DEFAULT 1 NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT business_rules_event_type_check CHECK (((event_type)::text = ANY (ARRAY[('BeforeInsert'::character varying)::text, ('AfterInsert'::character varying)::text, ('BeforeUpdate'::character varying)::text, ('AfterUpdate'::character varying)::text, ('BeforeDelete'::character varying)::text, ('AfterDelete'::character varying)::text, ('RowInit'::character varying)::text, ('RowSelect'::character varying)::text, ('RowEdit'::character varying)::text]))),
	CONSTRAINT business_rules_execution_location_check CHECK (((execution_location)::text = ANY (ARRAY[('server'::character varying)::text, ('client'::character varying)::text, ('both'::character varying)::text]))),
	CONSTRAINT business_rules_pkey PRIMARY KEY (id),
	CONSTRAINT business_rules_tenant_name_key UNIQUE (tenant_id, name),
	CONSTRAINT business_rules_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT business_rules_table_fk FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE,
	CONSTRAINT business_rules_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
	CONSTRAINT business_rules_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_business_rules_description_trgm ON public.business_rules USING gin (description gin_trgm_ops);
CREATE INDEX idx_business_rules_name ON public.business_rules USING btree (tenant_id, name);
CREATE INDEX idx_business_rules_table_event ON public.business_rules USING btree (table_id, event_type, is_active);
CREATE INDEX idx_business_rules_tenant_active ON public.business_rules USING btree (tenant_id, is_active);

-- Table Triggers

create trigger update_business_rule_timestamp before
update
    on
    public.business_rules for each row execute function update_timestamp();
create trigger update_business_rules_updated_at before
update
    on
    public.business_rules for each row execute function update_timestamp();


-- public.catalog_node_type definition

-- Drop table

-- DROP TABLE public.catalog_node_type;

CREATE TABLE public.catalog_node_type (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_dataource_id uuid NULL,
	catalog_type_name varchar NOT NULL,
	description text NULL,
	is_active bool DEFAULT true NULL,
	parent_type_id uuid NULL,
	config jsonb NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	tenant_id uuid NULL,
	core_id uuid NULL,
	CONSTRAINT catalog_node_type_pkey PRIMARY KEY (id),
	CONSTRAINT catalog_node_type_catalog_node_type_fk FOREIGN KEY (parent_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT catalog_node_type_tenant_product_datasource_fk FOREIGN KEY (tenant_dataource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT catalog_node_type_tenants_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);


-- public.fabric_defn definition

-- Drop table

-- DROP TABLE public.fabric_defn;

CREATE TABLE public.fabric_defn (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_id uuid NOT NULL,
	tenant_datasource_id uuid NOT NULL,
	model_key text NOT NULL,
	"version" int4 NOT NULL,
	status public.fabric_status DEFAULT 'draft'::fabric_status NOT NULL,
	is_current bool DEFAULT false NOT NULL,
	title text NULL,
	description text NULL,
	source_config jsonb NOT NULL,
	resolved_config jsonb NOT NULL,
	created_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	published_at timestamptz NULL,
	checksum_sha256 bytea NULL,
	updated_at timestamptz NULL,
	CONSTRAINT fabric_defn_pkey PRIMARY KEY (id),
	CONSTRAINT fabric_defn_tenant_id_model_key_version_key UNIQUE (tenant_id, model_key, version),
	CONSTRAINT fk_fabric_defn_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
	CONSTRAINT fk_fabric_defn_tenant_datasource FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE
);
CREATE INDEX idx_fabric_defn_is_current ON public.fabric_defn USING btree (is_current);
CREATE INDEX idx_fabric_defn_model_key ON public.fabric_defn USING btree (model_key);
CREATE INDEX idx_fabric_defn_tenant_datasource_id ON public.fabric_defn USING btree (tenant_datasource_id);
CREATE INDEX idx_fabric_defn_tenant_id ON public.fabric_defn USING btree (tenant_id);

-- Table Triggers

create trigger fabric_defn_refresh_index_trigger after
insert
    or
update
    on
    public.fabric_defn for each row execute function fabric_defn_refresh_index();


-- public.integration_credentials definition

-- Drop table

-- DROP TABLE public.integration_credentials;

CREATE TABLE public.integration_credentials (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	integration_id uuid NOT NULL,
	credential_type varchar(50) NOT NULL,
	credential_name varchar(255) NOT NULL,
	credentials jsonb NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT integration_credentials_integration_name_key UNIQUE (integration_id, credential_name),
	CONSTRAINT integration_credentials_pkey PRIMARY KEY (id),
	CONSTRAINT integration_credentials_type_check CHECK (((credential_type)::text = ANY (ARRAY[('api_key'::character varying)::text, ('oauth2'::character varying)::text, ('basic_auth'::character varying)::text, ('certificate'::character varying)::text, ('aws'::character varying)::text, ('azure'::character varying)::text, ('gcp'::character varying)::text, ('kafka'::character varying)::text]))),
	CONSTRAINT integration_credentials_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT integration_credentials_integration_fk FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE,
	CONSTRAINT integration_credentials_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_integration_credentials_credentials_gin ON public.integration_credentials USING gin (credentials);
CREATE INDEX idx_integration_credentials_integration_id ON public.integration_credentials USING btree (integration_id);

-- Table Triggers

create trigger update_integration_credentials_updated_at before
update
    on
    public.integration_credentials for each row execute function update_timestamp();


-- public.integration_metrics definition

-- Drop table

-- DROP TABLE public.integration_metrics;

CREATE TABLE public.integration_metrics (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	integration_id uuid NOT NULL,
	"timestamp" timestamptz DEFAULT now() NOT NULL,
	requests_count int4 DEFAULT 0 NOT NULL,
	success_count int4 DEFAULT 0 NOT NULL,
	error_count int4 DEFAULT 0 NOT NULL,
	total_duration int8 DEFAULT 0 NOT NULL,
	avg_duration float8 DEFAULT 0 NOT NULL,
	min_duration int8 DEFAULT 0 NOT NULL,
	max_duration int8 DEFAULT 0 NOT NULL,
	CONSTRAINT integration_metrics_pkey PRIMARY KEY (id),
	CONSTRAINT integration_metrics_integration_fk FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE,
	CONSTRAINT integration_metrics_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);
CREATE INDEX idx_integration_metrics_tenant_integration ON public.integration_metrics USING btree (tenant_id, integration_id);
CREATE INDEX idx_integration_metrics_timestamp ON public.integration_metrics USING btree (tenant_id, "timestamp");


-- public.integration_versions definition

-- Drop table

-- DROP TABLE public.integration_versions;

CREATE TABLE public.integration_versions (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	integration_id uuid NOT NULL,
	"version" int4 NOT NULL,
	config jsonb NOT NULL,
	"comment" text NULL,
	created_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT integration_versions_integration_id_version_key UNIQUE (integration_id, version),
	CONSTRAINT integration_versions_pkey PRIMARY KEY (id),
	CONSTRAINT integration_versions_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT integration_versions_integration_fk FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE
);
CREATE INDEX idx_integration_versions_integration_id ON public.integration_versions USING btree (integration_id);


-- public.message_delivery_logs definition

-- Drop table

-- DROP TABLE public.message_delivery_logs;

CREATE TABLE public.message_delivery_logs (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	message_id uuid NOT NULL,
	target_id uuid NOT NULL,
	status varchar(50) NOT NULL,
	status_code int4 NULL,
	error_message text NULL,
	attempt_count int4 DEFAULT 1 NOT NULL,
	request_body text NULL,
	response_body text NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT message_delivery_logs_pkey PRIMARY KEY (id),
	CONSTRAINT fk_target FOREIGN KEY (target_id) REFERENCES public.message_targets(id) ON DELETE CASCADE
);
CREATE INDEX idx_message_delivery_logs_message_id ON public.message_delivery_logs USING btree (message_id);
CREATE INDEX idx_message_delivery_logs_target_id ON public.message_delivery_logs USING btree (target_id);


-- public.metadata_columns definition

-- Drop table

-- DROP TABLE public.metadata_columns;

CREATE TABLE public.metadata_columns (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	table_id uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	display_name varchar(255) NOT NULL,
	description text NULL,
	data_type varchar(50) NOT NULL,
	is_required bool DEFAULT false NOT NULL,
	is_primary_key bool DEFAULT false NOT NULL,
	is_foreign_key bool DEFAULT false NOT NULL,
	reference_table_id uuid NULL,
	reference_column_id uuid NULL,
	default_value text NULL,
	max_length int4 NULL,
	"precision" int4 NULL,
	"scale" int4 NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT metadata_columns_pkey PRIMARY KEY (id),
	CONSTRAINT metadata_columns_table_name_key UNIQUE (table_id, name),
	CONSTRAINT metadata_columns_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT metadata_columns_reference_column_fk FOREIGN KEY (reference_column_id) REFERENCES public.metadata_columns(id),
	CONSTRAINT metadata_columns_reference_table_fk FOREIGN KEY (reference_table_id) REFERENCES public.metadata_tables(id),
	CONSTRAINT metadata_columns_table_fk FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE,
	CONSTRAINT metadata_columns_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_metadata_columns_reference_table ON public.metadata_columns USING btree (reference_table_id);
CREATE INDEX idx_metadata_columns_table_id ON public.metadata_columns USING btree (table_id);

-- Table Triggers

create trigger update_metadata_column_timestamp before
update
    on
    public.metadata_columns for each row execute function update_timestamp();
create trigger update_metadata_columns_updated_at before
update
    on
    public.metadata_columns for each row execute function update_timestamp();


-- public.metadata_fields definition

-- Drop table

-- DROP TABLE public.metadata_fields;

CREATE TABLE public.metadata_fields (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	table_id uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	display_name varchar(255) NOT NULL,
	description text NULL,
	data_type varchar(50) NOT NULL,
	is_required bool DEFAULT false NULL,
	is_key bool DEFAULT false NULL,
	default_value text NULL,
	min_value text NULL,
	max_value text NULL,
	regex_pattern text NULL,
	list_of_values jsonb NULL,
	ui_properties jsonb NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamp NOT NULL,
	updated_at timestamp NOT NULL,
	CONSTRAINT metadata_fields_pkey PRIMARY KEY (id),
	CONSTRAINT metadata_fields_table_id_name_key UNIQUE (table_id, name),
	CONSTRAINT metadata_fields_table_id_fkey FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE,
	CONSTRAINT metadata_fields_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);


-- public.metadata_relationships definition

-- Drop table

-- DROP TABLE public.metadata_relationships;

CREATE TABLE public.metadata_relationships (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	source_table uuid NOT NULL,
	source_column uuid NOT NULL,
	target_table uuid NOT NULL,
	target_column uuid NOT NULL,
	relation_type varchar(50) NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT metadata_relationships_pkey PRIMARY KEY (id),
	CONSTRAINT metadata_relationships_tenant_id_name_key UNIQUE (tenant_id, name),
	CONSTRAINT metadata_relationships_source_column_fk FOREIGN KEY (source_column) REFERENCES public.metadata_columns(id) ON DELETE CASCADE,
	CONSTRAINT metadata_relationships_source_table_fk FOREIGN KEY (source_table) REFERENCES public.metadata_tables(id) ON DELETE CASCADE,
	CONSTRAINT metadata_relationships_target_column_fk FOREIGN KEY (target_column) REFERENCES public.metadata_columns(id) ON DELETE CASCADE,
	CONSTRAINT metadata_relationships_target_table_fk FOREIGN KEY (target_table) REFERENCES public.metadata_tables(id) ON DELETE CASCADE,
	CONSTRAINT metadata_relationships_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

-- Table Triggers

create trigger update_metadata_relationships_updated_at before
update
    on
    public.metadata_relationships for each row execute function update_timestamp();


-- public.pop_anomalies definition

-- Drop table

-- DROP TABLE public.pop_anomalies;

CREATE TABLE public.pop_anomalies (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	metric_id uuid NULL,
	computation_id uuid NULL,
	anomaly_type text NOT NULL,
	severity text NOT NULL,
	confidence numeric(5, 4) NULL,
	z_score numeric(10, 4) NULL,
	expected_value numeric(20, 6) NULL,
	expected_range_min numeric(20, 6) NULL,
	expected_range_max numeric(20, 6) NULL,
	actual_value numeric(20, 6) NULL,
	detection_method text NOT NULL,
	detection_params jsonb NULL,
	detected_at timestamptz DEFAULT now() NULL,
	status text DEFAULT 'open'::text NOT NULL,
	resolved_at timestamptz NULL,
	resolved_by text NULL,
	resolution_notes text NULL,
	CONSTRAINT pop_anomalies_metric_id_computation_id_anomaly_type_key UNIQUE (metric_id, computation_id, anomaly_type),
	CONSTRAINT pop_anomalies_pkey PRIMARY KEY (id),
	CONSTRAINT pop_anomalies_computation_id_fkey FOREIGN KEY (computation_id) REFERENCES public.pop_computations(id) ON DELETE CASCADE,
	CONSTRAINT pop_anomalies_metric_id_fkey FOREIGN KEY (metric_id) REFERENCES public.pop_metrics(id) ON DELETE CASCADE
);
CREATE INDEX idx_pop_anomalies_metric ON public.pop_anomalies USING btree (metric_id);
CREATE INDEX idx_pop_anomalies_severity ON public.pop_anomalies USING btree (severity);
CREATE INDEX idx_pop_anomalies_status ON public.pop_anomalies USING btree (status);

-- Table Triggers

create trigger create_anomaly_review_trigger after
insert
    on
    public.pop_anomalies for each row execute function create_anomaly_review();


-- public.pop_steward_comments definition

-- Drop table

-- DROP TABLE public.pop_steward_comments;

CREATE TABLE public.pop_steward_comments (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	review_id uuid NULL,
	anomaly_id uuid NULL,
	commenter_user_id text NOT NULL,
	comment_type text NOT NULL,
	comment_text text NOT NULL,
	parent_comment_id uuid NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT pop_steward_comments_pkey PRIMARY KEY (id),
	CONSTRAINT pop_steward_comments_anomaly_id_fkey FOREIGN KEY (anomaly_id) REFERENCES public.pop_anomalies(id) ON DELETE SET NULL,
	CONSTRAINT pop_steward_comments_parent_comment_id_fkey FOREIGN KEY (parent_comment_id) REFERENCES public.pop_steward_comments(id) ON DELETE CASCADE,
	CONSTRAINT pop_steward_comments_review_id_fkey FOREIGN KEY (review_id) REFERENCES public.pop_steward_reviews(id) ON DELETE CASCADE
);


-- public.role_integration_permissions definition

-- Drop table

-- DROP TABLE public.role_integration_permissions;

CREATE TABLE public.role_integration_permissions (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	role_id uuid NOT NULL,
	integration_id uuid NOT NULL,
	can_execute bool DEFAULT false NOT NULL,
	can_update bool DEFAULT false NOT NULL,
	can_delete bool DEFAULT false NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT role_integration_permissions_pkey PRIMARY KEY (id),
	CONSTRAINT role_integration_permissions_role_integration_key UNIQUE (role_id, integration_id),
	CONSTRAINT role_integration_permissions_integration_fk FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE,
	CONSTRAINT role_integration_permissions_role_fk FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE
);
CREATE INDEX idx_role_integration_permissions_integration ON public.role_integration_permissions USING btree (integration_id);
CREATE INDEX idx_role_integration_permissions_role ON public.role_integration_permissions USING btree (role_id);


-- public.role_permissions definition

-- Drop table

-- DROP TABLE public.role_permissions;

CREATE TABLE public.role_permissions (
	role_id uuid NOT NULL,
	permission_id uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT role_permissions_pkey PRIMARY KEY (role_id, permission_id),
	CONSTRAINT role_permissions_permission_fk FOREIGN KEY (permission_id) REFERENCES public.permissions(id) ON DELETE CASCADE,
	CONSTRAINT role_permissions_role_fk FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE
);


-- public.tenant_chart definition

-- Drop table

-- DROP TABLE public.tenant_chart;

CREATE TABLE public.tenant_chart (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_datasource_id uuid NOT NULL,
	chart_name varchar NOT NULL,
	is_active bool DEFAULT true NOT NULL,
	config jsonb NULL,
	chart bytea NULL,
	cloned_from uuid NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT tenant_chart_pkey PRIMARY KEY (id),
	CONSTRAINT tenant_chart_unique UNIQUE (tenant_datasource_id, chart_name),
	CONSTRAINT tenant_chart_tenant_chart_fk FOREIGN KEY (cloned_from) REFERENCES public.tenant_chart(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT tenant_chart_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_tenant_chart_alpha_datasource_id ON public.tenant_chart USING btree (tenant_datasource_id, chart_name);
CREATE INDEX idx_tenant_chart_cloned_from ON public.tenant_chart USING btree (cloned_from);
CREATE INDEX tenant_chart_alpha_datasource_id_idx ON public.tenant_chart USING btree (tenant_datasource_id, chart_name);
CREATE INDEX tenant_chart_cloned_from_idx ON public.tenant_chart USING btree (cloned_from);


-- public.api_access_rules definition

-- Drop table

-- DROP TABLE public.api_access_rules;

CREATE TABLE public.api_access_rules (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_id uuid NOT NULL,
	"name" varchar(100) NOT NULL,
	description text NULL,
	api_id uuid NULL,
	endpoint_id uuid NULL,
	role_id uuid NULL,
	"permission" varchar(20) NOT NULL,
	resources _text NULL,
	is_active bool DEFAULT true NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT api_access_rules_permission_check CHECK (((permission)::text = ANY (ARRAY[('READ'::character varying)::text, ('WRITE'::character varying)::text, ('ADMIN'::character varying)::text, ('DENY'::character varying)::text]))),
	CONSTRAINT api_access_rules_pkey PRIMARY KEY (id),
	CONSTRAINT api_access_rules_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id),
	CONSTRAINT api_access_rules_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT api_access_rules_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES public.api_endpoints(id),
	CONSTRAINT api_access_rules_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id),
	CONSTRAINT api_access_rules_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id),
	CONSTRAINT api_access_rules_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)
);
CREATE INDEX idx_api_access_rules_api_id ON public.api_access_rules USING btree (api_id);
CREATE INDEX idx_api_access_rules_role_id ON public.api_access_rules USING btree (role_id);


-- public.api_audit_logs definition

-- Drop table

-- DROP TABLE public.api_audit_logs;

CREATE TABLE public.api_audit_logs (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_id uuid NOT NULL,
	"timestamp" timestamptz DEFAULT now() NULL,
	event_type varchar(50) NOT NULL,
	user_id uuid NULL,
	api_id uuid NULL,
	endpoint_id uuid NULL,
	resource varchar(255) NULL,
	"action" varchar(50) NULL,
	status varchar(20) NULL,
	details jsonb NULL,
	ip_address varchar(45) NULL,
	user_agent text NULL,
	CONSTRAINT api_audit_logs_pkey PRIMARY KEY (id),
	CONSTRAINT api_audit_logs_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id),
	CONSTRAINT api_audit_logs_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES public.api_endpoints(id),
	CONSTRAINT api_audit_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id),
	CONSTRAINT api_audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id)
);
CREATE INDEX idx_api_audit_logs_tenant_id_timestamp ON public.api_audit_logs USING btree (tenant_id, "timestamp");


-- public.business_rule_versions definition

-- Drop table

-- DROP TABLE public.business_rule_versions;

CREATE TABLE public.business_rule_versions (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	business_rule_id uuid NOT NULL,
	script text NOT NULL,
	"version" int4 NOT NULL,
	"comment" text NULL,
	created_by uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT business_rule_versions_pkey PRIMARY KEY (id),
	CONSTRAINT business_rule_versions_rule_version_key UNIQUE (business_rule_id, version),
	CONSTRAINT business_rule_versions_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id),
	CONSTRAINT business_rule_versions_rule_fk FOREIGN KEY (business_rule_id) REFERENCES public.business_rules(id) ON DELETE CASCADE
);
CREATE INDEX idx_business_rule_versions_rule_id ON public.business_rule_versions USING btree (business_rule_id);



-- public.catalog_edge_type definition (canonicalized)

-- Drop table

-- DROP TABLE public.catalog_edge_type;

CREATE TABLE public.catalog_edge_type (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	edge_type_name varchar NOT NULL,
	description text NULL,
	source_node_type_id uuid NOT NULL,
	target_node_type_id uuid NOT NULL,
	config jsonb NULL,
	is_active bool DEFAULT true NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	tenant_id uuid NOT NULL,
	core_id uuid NULL,
	CONSTRAINT catalog_edge_type_pkey PRIMARY KEY (id),
	CONSTRAINT catalog_edge_type_catalog_node_type_fk FOREIGN KEY (source_node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT catalog_edge_type_catalog_node_type_fk_1 FOREIGN KEY (target_node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT catalog_edge_type_tenants_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);


-- public.catalog_node definition

-- Drop table

-- DROP TABLE public.catalog_node;

CREATE TABLE public.catalog_node (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_datasource_id uuid NOT NULL,
	node_type_id uuid NOT NULL,
	node_name varchar NULL,
	description text NULL,
	properties jsonb NULL,
	qualified_path varchar NOT NULL,
	parent_id uuid NULL,
	created_at timestamptz NULL,
	updated_at timestamptz NULL,
	tenant_id uuid NOT NULL,
	core_id uuid NULL,
	is_alpha bool DEFAULT false NULL,
	schema_hash text NULL,
	lineage jsonb NULL,
	data_quality_contract jsonb NULL,
	sla jsonb NULL,
	steward_group text NULL,
	golden_path bool NULL,
	CONSTRAINT catalog_node_pk PRIMARY KEY (id),
	CONSTRAINT catalog_node_unique UNIQUE (tenant_datasource_id, node_type_id, qualified_path),
	CONSTRAINT catalog_node_catalog_node_fk FOREIGN KEY (core_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT catalog_node_catalog_node_type_fk FOREIGN KEY (node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT catalog_node_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT catalog_node_tenants_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX catalog_node_node_type_id_idx ON public.catalog_node USING btree (node_type_id);
CREATE INDEX catalog_node_tenant_datasource_id_idx ON public.catalog_node USING btree (tenant_datasource_id);


-- public.metadata_events definition

-- Drop table

-- DROP TABLE public.metadata_events;

CREATE TABLE public.metadata_events (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	table_id uuid NOT NULL,
	field_id uuid NULL,
	event_type varchar(50) NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	script text NOT NULL,
	is_active bool DEFAULT true NULL,
	execution_order int4 DEFAULT 0 NULL,
	execution_location varchar(50) DEFAULT 'server'::character varying NULL,
	created_by uuid NOT NULL,
	updated_by uuid NOT NULL,
	"version" int4 DEFAULT 1 NOT NULL,
	created_at timestamp NOT NULL,
	updated_at timestamp NOT NULL,
	CONSTRAINT metadata_events_pkey PRIMARY KEY (id),
	CONSTRAINT metadata_events_tenant_id_table_id_field_id_event_type_name_key UNIQUE (tenant_id, table_id, field_id, event_type, name),
	CONSTRAINT metadata_events_field_id_fkey FOREIGN KEY (field_id) REFERENCES public.metadata_fields(id) ON DELETE SET NULL,
	CONSTRAINT metadata_events_table_id_fkey FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE,
	CONSTRAINT metadata_events_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);


-- public.catalog_edge definition

-- Drop table

-- DROP TABLE public.catalog_edge;

CREATE TABLE public.catalog_edge (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	tenant_datasource_id uuid NOT NULL,
	source_node_id uuid NOT NULL,
	target_node_id uuid NOT NULL,
	relationship_type varchar NOT NULL,
	properties jsonb NULL,
	created_at timestamptz NULL,
	edge_type_id uuid NULL,
	updated_at timestamptz NULL,
	tenant_id uuid NULL,
	core_id uuid NULL,
	CONSTRAINT catalog_edge_pk PRIMARY KEY (id),
	CONSTRAINT catalog_edge_unique UNIQUE (tenant_datasource_id, source_node_id, edge_type_id, target_node_id),
	CONSTRAINT catalog_edge_catalog_edge_type_fk FOREIGN KEY (edge_type_id) REFERENCES public.catalog_edge_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT catalog_edge_catalog_sourcee_node_fk FOREIGN KEY (source_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT catalog_edge_catalog_target_node_fk FOREIGN KEY (target_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX catalog_edge_tenant_datasource_id_idx ON public.catalog_edge USING btree (tenant_datasource_id, source_node_id, edge_type_id, target_node_id);


-- public.metadata_event_logs definition

-- Drop table

-- DROP TABLE public.metadata_event_logs;

CREATE TABLE public.metadata_event_logs (
	id uuid NOT NULL,
	tenant_id uuid NOT NULL,
	event_id uuid NOT NULL,
	request_id uuid NOT NULL,
	user_id uuid NULL,
	execution_start timestamp NOT NULL,
	execution_end timestamp NOT NULL,
	duration_ms int4 NOT NULL,
	status varchar(50) NOT NULL,
	error_message text NULL,
	input_data jsonb NULL,
	output_data jsonb NULL,
	created_at timestamp NOT NULL,
	CONSTRAINT metadata_event_logs_pkey PRIMARY KEY (id),
	CONSTRAINT metadata_event_logs_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.metadata_events(id) ON DELETE CASCADE,
	CONSTRAINT metadata_event_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);


-- public.metadata_event_versions definition

-- Drop table

-- DROP TABLE public.metadata_event_versions;

CREATE TABLE public.metadata_event_versions (
	id uuid NOT NULL,
	event_id uuid NOT NULL,
	script text NOT NULL,
	"version" int4 NOT NULL,
	"comment" text NULL,
	created_by uuid NOT NULL,
	created_at timestamp NOT NULL,
	CONSTRAINT metadata_event_versions_pkey PRIMARY KEY (id),
	CONSTRAINT metadata_event_versions_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.metadata_events(id) ON DELETE CASCADE
);


-- public.business_rule_summary_view source

CREATE OR REPLACE VIEW public.business_rule_summary_view
AS SELECT br.id,
    br.tenant_id,
    br.name,
    br.description,
    br.table_id,
    mt.name AS table_name,
    br.event_type,
    br.is_active,
    br.execution_order,
    br.execution_location,
    br.version,
    br.created_at,
    br.updated_at,
    t.name AS tenant_name,
    u1.username AS created_by_username,
    u2.username AS updated_by_username
   FROM business_rules br
     JOIN metadata_tables mt ON br.table_id = mt.id
     JOIN tenants t ON br.tenant_id = t.id
     JOIN users u1 ON br.created_by = u1.id
     JOIN users u2 ON br.updated_by = u2.id;


-- public.catalog_edge_vw source

CREATE OR REPLACE VIEW public.catalog_edge_vw
AS SELECT ns.id AS subject_node_id,
    ns.node_name AS subject_node_name,
	ce.relationship_type,
	ce.id AS edge_id,
	cet.edge_type_name AS edge_type_name,
	-- Expose edge-type level configuration (JSON) as `edge_defn` so callers
	-- can differentiate type-level config from instance `properties`.
	cet.config AS edge_defn,
    no.node_name AS object_node_name,
    no.id AS object_node_id
   FROM catalog_edge ce
	JOIN catalog_node ns ON ns.id = ce.source_node_id
	JOIN catalog_node no ON no.id = ce.target_node_id
	LEFT JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id;


-- public.catalog_node_vw source

CREATE OR REPLACE VIEW public.catalog_node_vw
AS SELECT cn.tenant_datasource_id,
    tpd.source_name,
    cn.id AS node_id,
    cn.id AS id, -- Alias for compatibility
    cn.node_name,
    cnt.catalog_type_name,
	-- Expose the node-type level configuration (JSON) as `catalog_defn` so
	-- callers can differentiate type-level config from instance `properties`.
	COALESCE(cnt.config, jsonb_build_object('properties', cnt.properties)) AS catalog_defn,
    cn.node_type_id,
    cn.description,
    cn.qualified_path,
    cn.properties,
    cn.parent_id
   FROM catalog_node cn
     JOIN catalog_node_type cnt ON cnt.id = cn.node_type_id
     JOIN tenant_product_datasource tpd ON tpd.id = cn.tenant_datasource_id;


-- public.integration_summary_view source

CREATE OR REPLACE VIEW public.integration_summary_view
AS SELECT i.id,
    i.tenant_id,
    i.name,
    i.display_name,
    i.type,
    i.status,
    i.created_at,
    i.updated_at,
    COALESCE(m.requests_count, 0::bigint) AS total_requests,
    COALESCE(m.success_count, 0::bigint) AS successful_requests,
    COALESCE(m.error_count, 0::bigint) AS failed_requests,
    COALESCE(m.avg_duration, 0::double precision) AS avg_duration_ms,
    t.name AS tenant_name,
    u.username AS created_by_username
   FROM integrations i
     LEFT JOIN ( SELECT integration_metrics.integration_id,
            sum(integration_metrics.requests_count) AS requests_count,
            sum(integration_metrics.success_count) AS success_count,
            sum(integration_metrics.error_count) AS error_count,
            avg(integration_metrics.avg_duration) AS avg_duration
           FROM integration_metrics
          WHERE integration_metrics."timestamp" > (now() - '30 days'::interval)
          GROUP BY integration_metrics.integration_id) m ON i.id = m.integration_id
     JOIN tenants t ON i.tenant_id = t.id
     JOIN users u ON i.created_by = u.id;


-- public.pg_stat_statements source

CREATE OR REPLACE VIEW public.pg_stat_statements
AS SELECT userid,
    dbid,
    toplevel,
    queryid,
    query,
    plans,
    total_plan_time,
    min_plan_time,
    max_plan_time,
    mean_plan_time,
    stddev_plan_time,
    calls,
    total_exec_time,
    min_exec_time,
    max_exec_time,
    mean_exec_time,
    stddev_exec_time,
    rows,
    shared_blks_hit,
    shared_blks_read,
    shared_blks_dirtied,
    shared_blks_written,
    local_blks_hit,
    local_blks_read,
    local_blks_dirtied,
    local_blks_written,
    temp_blks_read,
    temp_blks_written,
    shared_blk_read_time,
    shared_blk_write_time,
    local_blk_read_time,
    local_blk_write_time,
    temp_blk_read_time,
    temp_blk_write_time,
    wal_records,
    wal_fpi,
    wal_bytes,
    jit_functions,
    jit_generation_time,
    jit_inlining_count,
    jit_inlining_time,
    jit_optimization_count,
    jit_optimization_time,
    jit_emission_count,
    jit_emission_time,
    jit_deform_count,
    jit_deform_time,
    stats_since,
    minmax_stats_since
   FROM pg_stat_statements(true) pg_stat_statements(userid, dbid, toplevel, queryid, query, plans, total_plan_time, min_plan_time, max_plan_time, mean_plan_time, stddev_plan_time, calls, total_exec_time, min_exec_time, max_exec_time, mean_exec_time, stddev_exec_time, rows, shared_blks_hit, shared_blks_read, shared_blks_dirtied, shared_blks_written, local_blks_hit, local_blks_read, local_blks_dirtied, local_blks_written, temp_blks_read, temp_blks_written, shared_blk_read_time, shared_blk_write_time, local_blk_read_time, local_blk_write_time, temp_blk_read_time, temp_blk_write_time, wal_records, wal_fpi, wal_bytes, jit_functions, jit_generation_time, jit_inlining_count, jit_inlining_time, jit_optimization_count, jit_optimization_time, jit_emission_count, jit_emission_time, jit_deform_count, jit_deform_time, stats_since, minmax_stats_since);


-- public.pg_stat_statements_info source

CREATE OR REPLACE VIEW public.pg_stat_statements_info
AS SELECT dealloc,
    stats_reset
   FROM pg_stat_statements_info() pg_stat_statements_info(dealloc, stats_reset);


-- public.pop_anomaly_summary source

CREATE OR REPLACE VIEW public.pop_anomaly_summary
AS SELECT m.domain,
    m.category,
    a.severity,
    a.anomaly_type,
    count(*) AS anomaly_count,
    max(a.detected_at) AS latest_detection,
    array_agg(DISTINCT m.name) AS affected_metrics
   FROM pop_anomalies a
     JOIN pop_metrics m ON a.metric_id = m.id
  WHERE a.status = 'open'::text
  GROUP BY m.domain, m.category, a.severity, a.anomaly_type
  ORDER BY m.domain, m.category, a.severity;


-- public.pop_metrics_with_latest source

CREATE OR REPLACE VIEW public.pop_metrics_with_latest
AS SELECT m.id,
    m.name,
    m.display_name,
    m.description,
    m.domain,
    m.category,
    m.metric_type,
    m.base_query,
    m.aggregation_function,
    m.date_column,
    m.value_column,
    m.granularity,
    m.comparison_periods,
    m.owner_user_id,
    m.steward_group,
    m.data_source,
    m.schema_name,
    m.table_name,
    m.sla_freshness_hours,
    m.sla_completeness_threshold,
    m.data_quality_checks,
    m.status,
    m.golden_path,
    m.version,
    m.created_at,
    m.updated_at,
    m.created_by,
    m.updated_by,
    c.current_value,
    c.previous_value,
    c.delta,
    c.percent_change,
    c.period_start,
    c.period_end,
    c.last_updated AS last_computed_at,
        CASE
            WHEN a.id IS NOT NULL THEN true
            ELSE false
        END AS has_anomalies,
    count(a.id) AS anomaly_count
   FROM pop_metrics m
     LEFT JOIN pop_computations c ON m.id = c.metric_id AND c.id = (( SELECT pop_computations.id
           FROM pop_computations
          WHERE pop_computations.metric_id = m.id
          ORDER BY pop_computations.period_end DESC, pop_computations.last_updated DESC
         LIMIT 1))
     LEFT JOIN pop_anomalies a ON m.id = a.metric_id AND a.status = 'open'::text
  WHERE m.status = 'active'::text
  GROUP BY m.id, c.id, c.current_value, c.previous_value, c.delta, c.percent_change, c.period_start, c.period_end, c.last_updated, a.id;



-- DROP FUNCTION public._fabric_defn_index_put(uuid, uuid, text, int4, text, text, text, join_relationship, text, text, text);

CREATE OR REPLACE FUNCTION public._fabric_defn_index_put(p_defn_id uuid, p_tenant_id uuid, p_model_key text, p_version integer, p_kind text, p_name text, p_type text, p_relationship join_relationship, p_title text, p_description text, p_sql text)
 RETURNS void
 LANGUAGE sql
AS $function$
  insert into fabric_defn_index
    (defn_id, tenant_id, model_key, version, kind, name, type, relationship, title, description, sql)
  values
    (p_defn_id, p_tenant_id, p_model_key, p_version, p_kind, p_name, p_type, p_relationship, p_title, p_description, p_sql)
  on conflict (tenant_id, model_key, version, kind, name) do update
    set type = excluded.type,
        relationship = excluded.relationship,
        title = excluded.title,
        description = excluded.description,
        sql = excluded.sql;
$function$
;

-- DROP FUNCTION public.armor(bytea);

CREATE OR REPLACE FUNCTION public.armor(bytea)
 RETURNS text
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_armor$function$
;

-- DROP FUNCTION public.armor(bytea, _text, _text);

CREATE OR REPLACE FUNCTION public.armor(bytea, text[], text[])
 RETURNS text
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_armor$function$
;

-- DROP FUNCTION public.clone_api_for_tenant(uuid, uuid, varchar, uuid);

CREATE OR REPLACE FUNCTION public.clone_api_for_tenant(p_source_api_id uuid, p_target_tenant_id uuid, p_new_name character varying, p_user_id uuid)
 RETURNS uuid
 LANGUAGE plpgsql
AS $function$
DECLARE
    v_source_api RECORD;
    v_new_api_id UUID;
    v_endpoint RECORD;
    v_new_endpoint_id UUID;
    v_hook RECORD;
    v_security RECORD;
    v_doc RECORD;
BEGIN
    -- Get source API details
    SELECT * INTO v_source_api FROM public.api_definitions WHERE id = p_source_api_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Source API not found';
    END IF;
    
    -- Create new API definition
    INSERT INTO public.api_definitions (
        tenant_id, name, display_name, description, api_type, version, status,
        visibility, base_path, group_id, specification, is_core, cloned_from_id,
        created_by, updated_by
    ) VALUES (
        p_target_tenant_id, p_new_name, v_source_api.display_name, v_source_api.description,
        v_source_api.api_type, v_source_api.version, 'DRAFT', 'TENANT_SPECIFIC',
        v_source_api.base_path, v_source_api.group_id, v_source_api.specification,
        false, p_source_api_id, p_user_id, p_user_id
    ) RETURNING id INTO v_new_api_id;
    
    -- Clone endpoints
    FOR v_endpoint IN SELECT * FROM public.api_endpoints WHERE api_id = p_source_api_id LOOP
        INSERT INTO public.api_endpoints (
            api_id, tenant_id, path, http_method, operation_id, summary,
            description, request_schema, response_schema, deprecated,
            created_by, updated_by
        ) VALUES (
            v_new_api_id, p_target_tenant_id, v_endpoint.path, v_endpoint.http_method,
            v_endpoint.operation_id, v_endpoint.summary, v_endpoint.description,
            v_endpoint.request_schema, v_endpoint.response_schema, v_endpoint.deprecated,
            p_user_id, p_user_id
        ) RETURNING id INTO v_new_endpoint_id;
        
        -- Clone hooks for this endpoint
        FOR v_hook IN SELECT * FROM public.api_hooks WHERE endpoint_id = v_endpoint.id LOOP
            INSERT INTO public.api_hooks (
                tenant_id, name, display_name, description, event_type,
                api_id, endpoint_id, script, execution_order, is_active,
                execution_location, created_by, updated_by
            ) VALUES (
                p_target_tenant_id, v_hook.name, v_hook.display_name, v_hook.description,
                v_hook.event_type, v_new_api_id, v_new_endpoint_id, v_hook.script,
                v_hook.execution_order, v_hook.is_active, v_hook.execution_location,
                p_user_id, p_user_id
            );
        END LOOP;
    END LOOP;
    
    -- Clone API-level hooks
    FOR v_hook IN SELECT * FROM public.api_hooks WHERE api_id = p_source_api_id AND endpoint_id IS NULL LOOP
        INSERT INTO public.api_hooks (
            tenant_id, name, display_name, description, event_type,
            api_id, endpoint_id, script, execution_order, is_active,
            execution_location, created_by, updated_by
        ) VALUES (
            p_target_tenant_id, v_hook.name, v_hook.display_name, v_hook.description,
            v_hook.event_type, v_new_api_id, v_hook.endpoint_id, v_hook.script,
            v_hook.execution_order, v_hook.is_active, v_hook.execution_location,
            p_user_id, p_user_id
        );
    END LOOP;

    -- Clone security configurations
    FOR v_security IN SELECT * FROM public.api_security_configs WHERE api_id = p_source_api_id LOOP
        INSERT INTO public.api_security_configs (
            api_id, tenant_id, auth_methods, rate_limit_requests, rate_limit_period,
            ip_whitelist, require_approval, created_by, updated_by
        ) VALUES (
            v_new_api_id, p_target_tenant_id, v_security.auth_methods, v_security.rate_limit_requests,
            v_security.rate_limit_period, v_security.ip_whitelist, v_security.require_approval,
            p_user_id, p_user_id
        );
    END LOOP;

    -- Clone documentation
    FOR v_doc IN SELECT * FROM public.api_documentation WHERE api_id = p_source_api_id LOOP
        INSERT INTO public.api_documentation (
            api_id, tenant_id, content_type, content, section, order_index,
            created_by, updated_by
        ) VALUES (
            v_new_api_id, p_target_tenant_id, v_doc.content_type, v_doc.content,
            v_doc.section, v_doc.order_index, p_user_id, p_user_id
        );
    END LOOP;

    RETURN v_new_api_id;
END;
$function$
;

-- DROP FUNCTION public.create_anomaly_review();

CREATE OR REPLACE FUNCTION public.create_anomaly_review()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    -- Only create review for high/critical anomalies
    IF NEW.severity IN ('high', 'critical') AND NEW.status = 'open' THEN
        INSERT INTO public.pop_steward_reviews (
            metric_id,
            review_period_start,
            review_period_end,
            reviewer_user_id,
            review_type,
            due_date
        )
        SELECT
            NEW.metric_id,
            c.period_start,
            c.period_end,
            m.owner_user_id,
            'anomaly_investigation',
            NOW() + INTERVAL '7 days'
        FROM public.pop_computations c
        JOIN public.pop_metrics m ON c.metric_id = m.id
        WHERE c.id = NEW.computation_id
        ON CONFLICT DO NOTHING;
    END IF;

    RETURN NEW;
END;
$function$
;

-- DROP FUNCTION public.crypt(text, text);

CREATE OR REPLACE FUNCTION public.crypt(text, text)
 RETURNS text
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_crypt$function$
;

-- DROP FUNCTION public.dearmor(text);

CREATE OR REPLACE FUNCTION public.dearmor(text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_dearmor$function$
;

-- DROP FUNCTION public.decrypt(bytea, bytea, text);

CREATE OR REPLACE FUNCTION public.decrypt(bytea, bytea, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_decrypt$function$
;

-- DROP FUNCTION public.decrypt_iv(bytea, bytea, bytea, text);

CREATE OR REPLACE FUNCTION public.decrypt_iv(bytea, bytea, bytea, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_decrypt_iv$function$
;

-- DROP FUNCTION public.digest(bytea, text);

CREATE OR REPLACE FUNCTION public.digest(bytea, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_digest$function$
;

-- DROP FUNCTION public.digest(text, text);

CREATE OR REPLACE FUNCTION public.digest(text, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_digest$function$
;

-- DROP FUNCTION public.encrypt(bytea, bytea, text);

CREATE OR REPLACE FUNCTION public.encrypt(bytea, bytea, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_encrypt$function$
;

-- DROP FUNCTION public.encrypt_iv(bytea, bytea, bytea, text);

CREATE OR REPLACE FUNCTION public.encrypt_iv(bytea, bytea, bytea, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_encrypt_iv$function$
;

-- DROP FUNCTION public.execute_business_rules(uuid, text, text, jsonb);

CREATE OR REPLACE FUNCTION public.execute_business_rules(p_tenant_id uuid, p_table_name text, p_event_type text, p_data jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
AS $function$
DECLARE
    v_rule RECORD;
    v_result JSONB;
    v_modified_data JSONB := p_data;
BEGIN
    FOR v_rule IN
        SELECT br.id, br.script
        FROM business_rules br
        JOIN metadata_tables mt ON br.table_id = mt.id
        WHERE br.tenant_id = p_tenant_id
          AND mt.name = p_table_name
          AND br.event_type = p_event_type
          AND br.is_active = TRUE
        ORDER BY br.execution_order
    LOOP
        INSERT INTO audit_logs (
            id, tenant_id, type, action, resource_type, resource_id,
            status, request_data
        )
        VALUES (
            uuid_generate_v4(), p_tenant_id, 'business_rule_execution', 'execute',
            'business_rule', v_rule.id, 'success', p_data
        );
    END LOOP;
    RETURN v_modified_data;
END;
$function$
;

-- DROP FUNCTION public.fabric_defn_refresh_index();

CREATE OR REPLACE FUNCTION public.fabric_defn_refresh_index()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
declare
  v jsonb;
  -- generic fields we expect in each item
  it_name text; it_type text; it_title text; it_desc text; it_sql text; it_rel join_relationship;
begin
  -- Clear existing index rows for this definition
  delete from fabric_defn_index where defn_id = new.id;

  -- Helper to extract array items from a JSON path and write to index
  -- We index both core.* and custom.* arrays; resolved_config should already have overrides applied.
  -- Dimensions
  for v in
    select jsonb_array_elements(coalesce(new.resolved_config #> '{core,dimensions}', '[]'::jsonb))
    union all
    select jsonb_array_elements(coalesce(new.resolved_config #> '{custom,dimensions}', '[]'::jsonb))
  loop
    it_name := (v->>'name');
    it_type := (v->>'type');
    it_title := (v->>'title');
    it_desc := (v->>'description');
    it_sql := (v->>'sql');
    perform _fabric_defn_index_put(new.id, new.tenant_id, new.model_key, new.version,
                                   'dimension', it_name, it_type, null, it_title, it_desc, it_sql);
  end loop;

  -- Measures
  for v in
    select jsonb_array_elements(coalesce(new.resolved_config #> '{core,measures}', '[]'::jsonb))
    union all
    select jsonb_array_elements(coalesce(new.resolved_config #> '{custom,measures}', '[]'::jsonb))
  loop
    it_name := (v->>'name');
    it_type := (v->>'type');
    it_title := (v->>'title');
    it_desc := (v->>'description');
    it_sql := (v->>'sql');
    perform _fabric_defn_index_put(new.id, new.tenant_id, new.model_key, new.version,
                                   'measure', it_name, it_type, null, it_title, it_desc, it_sql);
  end loop;

  -- Joins
  for v in
    select jsonb_array_elements(coalesce(new.resolved_config #> '{core,joins}', '[]'::jsonb))
    union all
    select jsonb_array_elements(coalesce(new.resolved_config #> '{custom,joins}', '[]'::jsonb))
  loop
    it_name := (v->>'name');
    it_rel := (v->>'relationship')::join_relationship;
    it_title := (v->>'title');
    it_desc := (v->>'description');
    it_sql := (v->>'sql');
    perform _fabric_defn_index_put(new.id, new.tenant_id, new.model_key, new.version,
                                   'join', it_name, null, it_rel, it_title, it_desc, it_sql);
  end loop;

  return new;
end;
$function$
;

-- DROP FUNCTION public.gen_random_bytes(int4);

CREATE OR REPLACE FUNCTION public.gen_random_bytes(integer)
 RETURNS bytea
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_random_bytes$function$
;

-- DROP FUNCTION public.gen_random_uuid();

CREATE OR REPLACE FUNCTION public.gen_random_uuid()
 RETURNS uuid
 LANGUAGE c
 PARALLEL SAFE
AS '$libdir/pgcrypto', $function$pg_random_uuid$function$
;

-- DROP FUNCTION public.gen_salt(text, int4);

CREATE OR REPLACE FUNCTION public.gen_salt(text, integer)
 RETURNS text
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_gen_salt_rounds$function$
;

-- DROP FUNCTION public.gen_salt(text);

CREATE OR REPLACE FUNCTION public.gen_salt(text)
 RETURNS text
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_gen_salt$function$
;

-- DROP FUNCTION public.gin_extract_query_trgm(text, internal, int2, internal, internal, internal, internal);

CREATE OR REPLACE FUNCTION public.gin_extract_query_trgm(text, internal, smallint, internal, internal, internal, internal)
 RETURNS internal
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gin_extract_query_trgm$function$
;

-- DROP FUNCTION public.gin_extract_value_trgm(text, internal);

CREATE OR REPLACE FUNCTION public.gin_extract_value_trgm(text, internal)
 RETURNS internal
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gin_extract_value_trgm$function$
;

-- DROP FUNCTION public.gin_trgm_consistent(internal, int2, text, int4, internal, internal, internal, internal);

CREATE OR REPLACE FUNCTION public.gin_trgm_consistent(internal, smallint, text, integer, internal, internal, internal, internal)
 RETURNS boolean
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gin_trgm_consistent$function$
;

-- DROP FUNCTION public.gin_trgm_triconsistent(internal, int2, text, int4, internal, internal, internal);

CREATE OR REPLACE FUNCTION public.gin_trgm_triconsistent(internal, smallint, text, integer, internal, internal, internal)
 RETURNS "char"
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gin_trgm_triconsistent$function$
;

-- DROP FUNCTION public.gtrgm_compress(internal);

CREATE OR REPLACE FUNCTION public.gtrgm_compress(internal)
 RETURNS internal
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_compress$function$
;

-- DROP FUNCTION public.gtrgm_consistent(internal, text, int2, oid, internal);

CREATE OR REPLACE FUNCTION public.gtrgm_consistent(internal, text, smallint, oid, internal)
 RETURNS boolean
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_consistent$function$
;

-- DROP FUNCTION public.gtrgm_decompress(internal);

CREATE OR REPLACE FUNCTION public.gtrgm_decompress(internal)
 RETURNS internal
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_decompress$function$
;

-- DROP FUNCTION public.gtrgm_distance(internal, text, int2, oid, internal);

CREATE OR REPLACE FUNCTION public.gtrgm_distance(internal, text, smallint, oid, internal)
 RETURNS double precision
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_distance$function$
;

-- DROP FUNCTION public.gtrgm_in(cstring);

CREATE OR REPLACE FUNCTION public.gtrgm_in(cstring)
 RETURNS gtrgm
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_in$function$
;

-- DROP FUNCTION public.gtrgm_options(internal);

CREATE OR REPLACE FUNCTION public.gtrgm_options(internal)
 RETURNS void
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE
AS '$libdir/pg_trgm', $function$gtrgm_options$function$
;

-- DROP FUNCTION public.gtrgm_out(gtrgm);

CREATE OR REPLACE FUNCTION public.gtrgm_out(gtrgm)
 RETURNS cstring
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_out$function$
;

-- DROP FUNCTION public.gtrgm_penalty(internal, internal, internal);

CREATE OR REPLACE FUNCTION public.gtrgm_penalty(internal, internal, internal)
 RETURNS internal
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_penalty$function$
;

-- DROP FUNCTION public.gtrgm_picksplit(internal, internal);

CREATE OR REPLACE FUNCTION public.gtrgm_picksplit(internal, internal)
 RETURNS internal
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_picksplit$function$
;

-- DROP FUNCTION public.gtrgm_same(gtrgm, gtrgm, internal);

CREATE OR REPLACE FUNCTION public.gtrgm_same(gtrgm, gtrgm, internal)
 RETURNS internal
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_same$function$
;

-- DROP FUNCTION public.gtrgm_union(internal, internal);

CREATE OR REPLACE FUNCTION public.gtrgm_union(internal, internal)
 RETURNS gtrgm
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$gtrgm_union$function$
;

-- DROP FUNCTION public.hmac(text, text, text);

CREATE OR REPLACE FUNCTION public.hmac(text, text, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_hmac$function$
;

-- DROP FUNCTION public.hmac(bytea, bytea, text);

CREATE OR REPLACE FUNCTION public.hmac(bytea, bytea, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pg_hmac$function$
;

-- DROP FUNCTION public.pg_stat_statements(in bool, out oid, out oid, out bool, out int8, out text, out int8, out float8, out float8, out float8, out float8, out float8, out int8, out float8, out float8, out float8, out float8, out float8, out int8, out int8, out int8, out int8, out int8, out int8, out int8, out int8, out int8, out int8, out int8, out float8, out float8, out float8, out float8, out float8, out float8, out int8, out int8, out numeric, out int8, out float8, out int8, out float8, out int8, out float8, out int8, out float8, out int8, out float8, out timestamptz, out timestamptz);

CREATE OR REPLACE FUNCTION public.pg_stat_statements(showtext boolean, OUT userid oid, OUT dbid oid, OUT toplevel boolean, OUT queryid bigint, OUT query text, OUT plans bigint, OUT total_plan_time double precision, OUT min_plan_time double precision, OUT max_plan_time double precision, OUT mean_plan_time double precision, OUT stddev_plan_time double precision, OUT calls bigint, OUT total_exec_time double precision, OUT min_exec_time double precision, OUT max_exec_time double precision, OUT mean_exec_time double precision, OUT stddev_exec_time double precision, OUT rows bigint, OUT shared_blks_hit bigint, OUT shared_blks_read bigint, OUT shared_blks_dirtied bigint, OUT shared_blks_written bigint, OUT local_blks_hit bigint, OUT local_blks_read bigint, OUT local_blks_dirtied bigint, OUT local_blks_written bigint, OUT temp_blks_read bigint, OUT temp_blks_written bigint, OUT shared_blk_read_time double precision, OUT shared_blk_write_time double precision, OUT local_blk_read_time double precision, OUT local_blk_write_time double precision, OUT temp_blk_read_time double precision, OUT temp_blk_write_time double precision, OUT wal_records bigint, OUT wal_fpi bigint, OUT wal_bytes numeric, OUT jit_functions bigint, OUT jit_generation_time double precision, OUT jit_inlining_count bigint, OUT jit_inlining_time double precision, OUT jit_optimization_count bigint, OUT jit_optimization_time double precision, OUT jit_emission_count bigint, OUT jit_emission_time double precision, OUT jit_deform_count bigint, OUT jit_deform_time double precision, OUT stats_since timestamp with time zone, OUT minmax_stats_since timestamp with time zone)
 RETURNS SETOF record
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pg_stat_statements', $function$pg_stat_statements_1_11$function$
;

-- DROP FUNCTION public.pg_stat_statements_info(out int8, out timestamptz);

CREATE OR REPLACE FUNCTION public.pg_stat_statements_info(OUT dealloc bigint, OUT stats_reset timestamp with time zone)
 RETURNS record
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pg_stat_statements', $function$pg_stat_statements_info$function$
;

-- DROP FUNCTION public.pg_stat_statements_reset(oid, oid, int8, bool);

CREATE OR REPLACE FUNCTION public.pg_stat_statements_reset(userid oid DEFAULT 0, dbid oid DEFAULT 0, queryid bigint DEFAULT 0, minmax_only boolean DEFAULT false)
 RETURNS timestamp with time zone
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pg_stat_statements', $function$pg_stat_statements_reset_1_11$function$
;

-- DROP FUNCTION public.pgp_armor_headers(in text, out text, out text);

CREATE OR REPLACE FUNCTION public.pgp_armor_headers(text, OUT key text, OUT value text)
 RETURNS SETOF record
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_armor_headers$function$
;

-- DROP FUNCTION public.pgp_key_id(bytea);

CREATE OR REPLACE FUNCTION public.pgp_key_id(bytea)
 RETURNS text
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_key_id_w$function$
;

-- DROP FUNCTION public.pgp_pub_decrypt(bytea, bytea);

CREATE OR REPLACE FUNCTION public.pgp_pub_decrypt(bytea, bytea)
 RETURNS text
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_decrypt_text$function$
;

-- DROP FUNCTION public.pgp_pub_decrypt(bytea, bytea, text);

CREATE OR REPLACE FUNCTION public.pgp_pub_decrypt(bytea, bytea, text)
 RETURNS text
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_decrypt_text$function$
;

-- DROP FUNCTION public.pgp_pub_decrypt(bytea, bytea, text, text);

CREATE OR REPLACE FUNCTION public.pgp_pub_decrypt(bytea, bytea, text, text)
 RETURNS text
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_decrypt_text$function$
;

-- DROP FUNCTION public.pgp_pub_decrypt_bytea(bytea, bytea);

CREATE OR REPLACE FUNCTION public.pgp_pub_decrypt_bytea(bytea, bytea)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_decrypt_bytea$function$
;

-- DROP FUNCTION public.pgp_pub_decrypt_bytea(bytea, bytea, text, text);

CREATE OR REPLACE FUNCTION public.pgp_pub_decrypt_bytea(bytea, bytea, text, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_decrypt_bytea$function$
;

-- DROP FUNCTION public.pgp_pub_decrypt_bytea(bytea, bytea, text);

CREATE OR REPLACE FUNCTION public.pgp_pub_decrypt_bytea(bytea, bytea, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_decrypt_bytea$function$
;

-- DROP FUNCTION public.pgp_pub_encrypt(text, bytea);

CREATE OR REPLACE FUNCTION public.pgp_pub_encrypt(text, bytea)
 RETURNS bytea
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_encrypt_text$function$
;

-- DROP FUNCTION public.pgp_pub_encrypt(text, bytea, text);

CREATE OR REPLACE FUNCTION public.pgp_pub_encrypt(text, bytea, text)
 RETURNS bytea
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_encrypt_text$function$
;

-- DROP FUNCTION public.pgp_pub_encrypt_bytea(bytea, bytea, text);

CREATE OR REPLACE FUNCTION public.pgp_pub_encrypt_bytea(bytea, bytea, text)
 RETURNS bytea
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_encrypt_bytea$function$
;

-- DROP FUNCTION public.pgp_pub_encrypt_bytea(bytea, bytea);

CREATE OR REPLACE FUNCTION public.pgp_pub_encrypt_bytea(bytea, bytea)
 RETURNS bytea
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_pub_encrypt_bytea$function$
;

-- DROP FUNCTION public.pgp_sym_decrypt(bytea, text);

CREATE OR REPLACE FUNCTION public.pgp_sym_decrypt(bytea, text)
 RETURNS text
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_sym_decrypt_text$function$
;

-- DROP FUNCTION public.pgp_sym_decrypt(bytea, text, text);

CREATE OR REPLACE FUNCTION public.pgp_sym_decrypt(bytea, text, text)
 RETURNS text
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_sym_decrypt_text$function$
;

-- DROP FUNCTION public.pgp_sym_decrypt_bytea(bytea, text, text);

CREATE OR REPLACE FUNCTION public.pgp_sym_decrypt_bytea(bytea, text, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_sym_decrypt_bytea$function$
;

-- DROP FUNCTION public.pgp_sym_decrypt_bytea(bytea, text);

CREATE OR REPLACE FUNCTION public.pgp_sym_decrypt_bytea(bytea, text)
 RETURNS bytea
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_sym_decrypt_bytea$function$
;

-- DROP FUNCTION public.pgp_sym_encrypt(text, text);

CREATE OR REPLACE FUNCTION public.pgp_sym_encrypt(text, text)
 RETURNS bytea
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_sym_encrypt_text$function$
;

-- DROP FUNCTION public.pgp_sym_encrypt(text, text, text);

CREATE OR REPLACE FUNCTION public.pgp_sym_encrypt(text, text, text)
 RETURNS bytea
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_sym_encrypt_text$function$
;

-- DROP FUNCTION public.pgp_sym_encrypt_bytea(bytea, text);

CREATE OR REPLACE FUNCTION public.pgp_sym_encrypt_bytea(bytea, text)
 RETURNS bytea
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_sym_encrypt_bytea$function$
;

-- DROP FUNCTION public.pgp_sym_encrypt_bytea(bytea, text, text);

CREATE OR REPLACE FUNCTION public.pgp_sym_encrypt_bytea(bytea, text, text)
 RETURNS bytea
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/pgcrypto', $function$pgp_sym_encrypt_bytea$function$
;

-- DROP FUNCTION public.set_limit(float4);

CREATE OR REPLACE FUNCTION public.set_limit(real)
 RETURNS real
 LANGUAGE c
 STRICT
AS '$libdir/pg_trgm', $function$set_limit$function$
;

-- DROP FUNCTION public.set_tenant_context(uuid);

CREATE OR REPLACE FUNCTION public.set_tenant_context(p_tenant_id uuid)
 RETURNS void
 LANGUAGE plpgsql
AS $function$
BEGIN
    PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, FALSE);
END;
$function$
;

-- DROP FUNCTION public.show_limit();

CREATE OR REPLACE FUNCTION public.show_limit()
 RETURNS real
 LANGUAGE c
 STABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$show_limit$function$
;

-- DROP FUNCTION public.show_trgm(text);

CREATE OR REPLACE FUNCTION public.show_trgm(text)
 RETURNS text[]
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$show_trgm$function$
;

-- DROP FUNCTION public.similarity(text, text);

CREATE OR REPLACE FUNCTION public.similarity(text, text)
 RETURNS real
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$similarity$function$
;

-- DROP FUNCTION public.similarity_dist(text, text);

CREATE OR REPLACE FUNCTION public.similarity_dist(text, text)
 RETURNS real
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$similarity_dist$function$
;

-- DROP FUNCTION public.similarity_op(text, text);

CREATE OR REPLACE FUNCTION public.similarity_op(text, text)
 RETURNS boolean
 LANGUAGE c
 STABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$similarity_op$function$
;

-- DROP FUNCTION public.strict_word_similarity(text, text);

CREATE OR REPLACE FUNCTION public.strict_word_similarity(text, text)
 RETURNS real
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$strict_word_similarity$function$
;

-- DROP FUNCTION public.strict_word_similarity_commutator_op(text, text);

CREATE OR REPLACE FUNCTION public.strict_word_similarity_commutator_op(text, text)
 RETURNS boolean
 LANGUAGE c
 STABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$strict_word_similarity_commutator_op$function$
;

-- DROP FUNCTION public.strict_word_similarity_dist_commutator_op(text, text);

CREATE OR REPLACE FUNCTION public.strict_word_similarity_dist_commutator_op(text, text)
 RETURNS real
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$strict_word_similarity_dist_commutator_op$function$
;

-- DROP FUNCTION public.strict_word_similarity_dist_op(text, text);

CREATE OR REPLACE FUNCTION public.strict_word_similarity_dist_op(text, text)
 RETURNS real
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$strict_word_similarity_dist_op$function$
;

-- DROP FUNCTION public.strict_word_similarity_op(text, text);

CREATE OR REPLACE FUNCTION public.strict_word_similarity_op(text, text)
 RETURNS boolean
 LANGUAGE c
 STABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$strict_word_similarity_op$function$
;

-- DROP FUNCTION public.update_engagement_notifications_updated_at();

CREATE OR REPLACE FUNCTION public.update_engagement_notifications_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$
;

-- DROP FUNCTION public.update_notification_campaigns_updated_at();

CREATE OR REPLACE FUNCTION public.update_notification_campaigns_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$
;

-- DROP FUNCTION public.update_notification_templates_updated_at();

CREATE OR REPLACE FUNCTION public.update_notification_templates_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$
;

-- DROP FUNCTION public.update_pop_updated_at();

CREATE OR REPLACE FUNCTION public.update_pop_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$
;

-- DROP FUNCTION public.update_product_updated_at();

CREATE OR REPLACE FUNCTION public.update_product_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$
;

-- DROP FUNCTION public.update_template_registry_updated_at();

CREATE OR REPLACE FUNCTION public.update_template_registry_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$
;

-- DROP FUNCTION public.update_timestamp();

CREATE OR REPLACE FUNCTION public.update_timestamp()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$
;

-- DROP FUNCTION public.update_updated_at_column();

CREATE OR REPLACE FUNCTION public.update_updated_at_column()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$
;

-- DROP FUNCTION public.uuid_generate_v1();

CREATE OR REPLACE FUNCTION public.uuid_generate_v1()
 RETURNS uuid
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_generate_v1$function$
;

-- DROP FUNCTION public.uuid_generate_v1mc();

CREATE OR REPLACE FUNCTION public.uuid_generate_v1mc()
 RETURNS uuid
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_generate_v1mc$function$
;

-- DROP FUNCTION public.uuid_generate_v3(uuid, text);

CREATE OR REPLACE FUNCTION public.uuid_generate_v3(namespace uuid, name text)
 RETURNS uuid
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_generate_v3$function$
;

-- DROP FUNCTION public.uuid_generate_v4();

CREATE OR REPLACE FUNCTION public.uuid_generate_v4()
 RETURNS uuid
 LANGUAGE c
 PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_generate_v4$function$
;

-- DROP FUNCTION public.uuid_generate_v5(uuid, text);

CREATE OR REPLACE FUNCTION public.uuid_generate_v5(namespace uuid, name text)
 RETURNS uuid
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_generate_v5$function$
;

-- DROP FUNCTION public.uuid_nil();

CREATE OR REPLACE FUNCTION public.uuid_nil()
 RETURNS uuid
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_nil$function$
;

-- DROP FUNCTION public.uuid_ns_dns();

CREATE OR REPLACE FUNCTION public.uuid_ns_dns()
 RETURNS uuid
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_ns_dns$function$
;

-- DROP FUNCTION public.uuid_ns_oid();

CREATE OR REPLACE FUNCTION public.uuid_ns_oid()
 RETURNS uuid
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_ns_oid$function$
;

-- DROP FUNCTION public.uuid_ns_url();

CREATE OR REPLACE FUNCTION public.uuid_ns_url()
 RETURNS uuid
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_ns_url$function$
;

-- DROP FUNCTION public.uuid_ns_x500();

CREATE OR REPLACE FUNCTION public.uuid_ns_x500()
 RETURNS uuid
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/uuid-ossp', $function$uuid_ns_x500$function$
;

-- DROP FUNCTION public.word_similarity(text, text);

CREATE OR REPLACE FUNCTION public.word_similarity(text, text)
 RETURNS real
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$word_similarity$function$
;

-- DROP FUNCTION public.word_similarity_commutator_op(text, text);

CREATE OR REPLACE FUNCTION public.word_similarity_commutator_op(text, text)
 RETURNS boolean
 LANGUAGE c
 STABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$word_similarity_commutator_op$function$
;

-- DROP FUNCTION public.word_similarity_dist_commutator_op(text, text);

CREATE OR REPLACE FUNCTION public.word_similarity_dist_commutator_op(text, text)
 RETURNS real
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$word_similarity_dist_commutator_op$function$
;

-- DROP FUNCTION public.word_similarity_dist_op(text, text);

CREATE OR REPLACE FUNCTION public.word_similarity_dist_op(text, text)
 RETURNS real
 LANGUAGE c
 IMMUTABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$word_similarity_dist_op$function$
;

-- DROP FUNCTION public.word_similarity_op(text, text);

CREATE OR REPLACE FUNCTION public.word_similarity_op(text, text)
 RETURNS boolean
 LANGUAGE c
 STABLE PARALLEL SAFE STRICT
AS '$libdir/pg_trgm', $function$word_similarity_op$function$
;