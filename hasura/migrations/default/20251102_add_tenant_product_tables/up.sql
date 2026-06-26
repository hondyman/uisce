-- Migration: Add missing tenant/product/datasource tables
-- Creates the table hierarchy required for tenant scope management

-- ============================================================================
-- alpha_datasource table
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.alpha_datasource (
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

-- ============================================================================
-- alpha_product table
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.alpha_product (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	product_name varchar(255) NULL,
	is_active bool DEFAULT true NOT NULL,
	product_code varchar(255) NULL,
	status varchar(50) DEFAULT 'active'::character varying NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT alpha_product_pkey PRIMARY KEY (id),
	CONSTRAINT alpha_product_unique UNIQUE (product_name)
);

-- ============================================================================
-- tenant_instance table
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.tenant_instance (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_id character varying(255) NOT NULL,
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

CREATE TRIGGER update_tenant_instance_updated_at BEFORE UPDATE ON public.tenant_instance
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ============================================================================
-- tenant_product table
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.tenant_product (
	id uuid DEFAULT uuid_generate_v4() NOT NULL,
	tenant_instance_id uuid NOT NULL,
	alpha_product_id uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL,
	version float4 NOT NULL,
	is_active bool DEFAULT false NOT NULL,
	CONSTRAINT tenant_product_pkey PRIMARY KEY (id),
	CONSTRAINT tenant_product_uniq UNIQUE (tenant_instance_id, alpha_product_id),
	CONSTRAINT tenant_product_product_fk FOREIGN KEY (alpha_product_id) REFERENCES public.alpha_product(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
	CONSTRAINT tenant_product_tenant_instance_fk FOREIGN KEY (tenant_instance_id) REFERENCES public.tenant_instance(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

-- ============================================================================
-- tenant_product_datasource table
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.tenant_product_datasource (
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

-- ============================================================================
-- Create product table (alias for alpha_product)
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.product (
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

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_tenant_instance_active ON public.tenant_instance(is_active);
CREATE INDEX IF NOT EXISTS idx_tenant_instance_tenant_id ON public.tenant_instance(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_product_active ON public.tenant_product(is_active);
CREATE INDEX IF NOT EXISTS idx_tenant_product_datasource_active ON public.tenant_product_datasource(is_active);
CREATE INDEX IF NOT EXISTS idx_alpha_product_active ON public.alpha_product(is_active);
CREATE INDEX IF NOT EXISTS idx_alpha_datasource_active ON public.alpha_datasource(is_active);
