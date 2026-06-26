-- DROP SCHEMA public;

CREATE SCHEMA public AUTHORIZATION pg_database_owner;

COMMENT ON SCHEMA public IS 'standard public schema';

-- public.tenants definition
CREATE TABLE public.tenants (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    CONSTRAINT tenants_pk PRIMARY KEY (id)
);

-- public.tenant_product_datasource definition
CREATE TABLE public.tenant_product_datasource (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT tenant_product_datasource_pk PRIMARY KEY (id),
    CONSTRAINT tenant_product_datasource_tenants_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

-- public.categories definition

-- Drop table

-- DROP TABLE public.categories;

CREATE TABLE public.categories (
	category_id int2 NOT NULL,
	category_name varchar(15) NOT NULL,
	description text NULL,
	picture bytea NULL,
	CONSTRAINT pk_categories PRIMARY KEY (category_id)
);


-- public.customer_demographics definition

-- Drop table

-- DROP TABLE public.customer_demographics;

CREATE TABLE public.customer_demographics (
	customer_type_id varchar(5) NOT NULL,
	customer_desc text NULL,
	CONSTRAINT pk_customer_demographics PRIMARY KEY (customer_type_id)
);


-- public.customers definition

-- Drop table

-- DROP TABLE public.customers;

CREATE TABLE public.customers (
	customer_id varchar(5) NOT NULL,
	company_name varchar(40) NOT NULL,
	contact_name varchar(30) NULL,
	contact_title varchar(30) NULL,
	address varchar(60) NULL,
	city varchar(15) NULL,
	region varchar(15) NULL,
	postal_code varchar(10) NULL,
	country varchar(15) NULL,
	phone varchar(24) NULL,
	fax varchar(24) NULL,
	CONSTRAINT pk_customers PRIMARY KEY (customer_id)
);


-- public.dim_country_cd definition

-- Drop table

-- DROP TABLE public.dim_country_cd;

CREATE TABLE public.dim_country_cd (
	cntry_cd varchar NOT NULL,
	cntry_name varchar NULL,
	CONSTRAINT dim_country_cd_pk PRIMARY KEY (cntry_cd)
);


-- public.ref_service_codes definition

-- Drop table

-- DROP TABLE public.ref_service_codes;

CREATE TABLE public.ref_service_codes (
	service_code varchar NULL,
	service_code_name varchar NULL,
	CONSTRAINT ref_service_codes_unique UNIQUE (service_code)
);


-- public.region definition

-- Drop table

-- DROP TABLE public.region;

CREATE TABLE public.region (
	region_id int2 NOT NULL,
	region_description varchar(60) NOT NULL,
	CONSTRAINT pk_region PRIMARY KEY (region_id)
);


-- public.shippers definition

-- Drop table

-- DROP TABLE public.shippers;

CREATE TABLE public.shippers (
	shipper_id int2 NOT NULL,
	company_name varchar(40) NOT NULL,
	phone varchar(24) NULL,
	CONSTRAINT pk_shippers PRIMARY KEY (shipper_id)
);


-- public.suppliers definition

-- Drop table

-- DROP TABLE public.suppliers;

CREATE TABLE public.suppliers (
	supplier_id int2 NOT NULL,
	company_name varchar(40) NOT NULL,
	contact_name varchar(30) NULL,
	contact_title varchar(30) NULL,
	address varchar(60) NULL,
	city varchar(15) NULL,
	region varchar(15) NULL,
	postal_code varchar(10) NULL,
	country varchar(15) NULL,
	phone varchar(24) NULL,
	fax varchar(24) NULL,
	homepage text NULL,
	CONSTRAINT pk_suppliers PRIMARY KEY (supplier_id)
);


-- public.us_states definition

-- Drop table

-- DROP TABLE public.us_states;

CREATE TABLE public.us_states (
	state_id int2 NOT NULL,
	state_name varchar(100) NULL,
	state_abbr varchar(2) NULL,
	state_region varchar(50) NULL,
	CONSTRAINT pk_usstates PRIMARY KEY (state_id)
);


-- public.customer_customer_demo definition

-- Drop table

-- DROP TABLE public.customer_customer_demo;

CREATE TABLE public.customer_customer_demo (
	customer_id varchar(5) NOT NULL,
	customer_type_id varchar(5) NOT NULL,
	CONSTRAINT pk_customer_customer_demo PRIMARY KEY (customer_id, customer_type_id),
	CONSTRAINT fk_customer_customer_demo_customer_demographics FOREIGN KEY (customer_type_id) REFERENCES public.customer_demographics(customer_type_id),
	CONSTRAINT fk_customer_customer_demo_customers FOREIGN KEY (customer_id) REFERENCES public.customers(customer_id)
);


-- public.employees definition

-- Drop table

-- DROP TABLE public.employees;

CREATE TABLE public.employees (
	employee_id int2 NOT NULL,
	last_name varchar(20) NOT NULL,
	first_name varchar(10) NOT NULL,
	title varchar(30) NULL,
	title_of_courtesy varchar(25) NULL,
	birth_date date NULL,
	hire_date date NULL,
	address varchar(60) NULL,
	city varchar(15) NULL,
	region varchar(15) NULL,
	postal_code varchar(10) NULL,
	country varchar(15) NULL,
	home_phone varchar(24) NULL,
	"extension" varchar(4) NULL,
	photo bytea NULL,
	notes text NULL,
	reports_to int2 NULL,
	photo_path varchar(255) NULL,
	CONSTRAINT pk_employees PRIMARY KEY (employee_id),
	CONSTRAINT fk_employees_employees FOREIGN KEY (reports_to) REFERENCES public.employees(employee_id),
	CONSTRAINT fk_employees_reports_to FOREIGN KEY (reports_to) REFERENCES public.employees(employee_id)
);


-- public.orders definition

-- Drop table

-- DROP TABLE public.orders;

CREATE TABLE public.orders (
	order_id int2 NOT NULL,
	customer_id varchar(5) NULL,
	employee_id int2 NULL,
	order_date date NULL,
	required_date date NULL,
	shipped_date date NULL,
	ship_via int2 NULL,
	freight float4 NULL,
	ship_name varchar(40) NULL,
	ship_address varchar(60) NULL,
	ship_city varchar(15) NULL,
	ship_region varchar(15) NULL,
	ship_postal_code varchar(10) NULL,
	ship_country varchar(15) NULL,
	CONSTRAINT pk_orders PRIMARY KEY (order_id),
	CONSTRAINT orders_customers_fk FOREIGN KEY (customer_id) REFERENCES public.customers(customer_id)
);


-- public.products definition

-- Drop table

-- DROP TABLE public.products;

CREATE TABLE public.products (
	product_id int2 NOT NULL,
	product_name varchar(40) NOT NULL,
	supplier_id int2 NULL,
	category_id int2 NULL,
	quantity_per_unit varchar(20) NULL,
	unit_price float4 NULL,
	units_in_stock int2 NULL,
	units_on_order int2 NULL,
	reorder_level int2 NULL,
	discontinued int4 NOT NULL,
	CONSTRAINT pk_products PRIMARY KEY (product_id),
	CONSTRAINT fk_products_categories FOREIGN KEY (category_id) REFERENCES public.categories(category_id),
	CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES public.categories(category_id),
	CONSTRAINT fk_products_supplier FOREIGN KEY (supplier_id) REFERENCES public.suppliers(supplier_id),
	CONSTRAINT fk_products_suppliers FOREIGN KEY (supplier_id) REFERENCES public.suppliers(supplier_id)
);


-- public.territories definition

-- Drop table

-- DROP TABLE public.territories;

CREATE TABLE public.territories (
	territory_id varchar(20) NOT NULL,
	territory_description varchar(60) NOT NULL,
	region_id int2 NOT NULL,
	CONSTRAINT pk_territories PRIMARY KEY (territory_id),
	CONSTRAINT fk_territories_region FOREIGN KEY (region_id) REFERENCES public.region(region_id)
);


-- public.employee_territories definition

-- Drop table

-- DROP TABLE public.employee_territories;

CREATE TABLE public.employee_territories (
	employee_id int2 NOT NULL,
	territory_id varchar(20) NOT NULL,
	CONSTRAINT employee_territories_pk PRIMARY KEY (employee_id, territory_id),
	CONSTRAINT fk_employee_territories_employees FOREIGN KEY (employee_id) REFERENCES public.employees(employee_id),
	CONSTRAINT fk_employee_territories_territories FOREIGN KEY (territory_id) REFERENCES public.territories(territory_id)
);


-- public.order_details definition

-- Drop table

-- DROP TABLE public.order_details;

CREATE TABLE public.order_details (
	order_id int2 NOT NULL,
	product_id int2 NOT NULL,
	unit_price float4 NOT NULL,
	quantity int2 NOT NULL,
	discount float4 NOT NULL,
	CONSTRAINT pk_order_details PRIMARY KEY (order_id, product_id),
	CONSTRAINT fk_order_details_orders FOREIGN KEY (order_id) REFERENCES public.orders(order_id),
	CONSTRAINT fk_order_details_products FOREIGN KEY (product_id) REFERENCES public.products(product_id)
);


-- public.customers_vw source

CREATE OR REPLACE VIEW public.customers_vw
AS SELECT customer_id,
    company_name,
    contact_name,
    contact_title,
    address,
    city,
    region,
    postal_code,
    country,
    phone,
    fax
   FROM customers;


-- public.employees_vw source

CREATE OR REPLACE VIEW public.employees_vw
AS SELECT employee_id,
    last_name,
    first_name,
    title,
    title_of_courtesy,
    birth_date,
    hire_date,
    address,
    city,
    region,
    postal_code,
    country,
    home_phone,
    extension,
    photo,
    notes,
    reports_to,
    photo_path
   FROM employees;


-- public.orders_vw source

CREATE OR REPLACE VIEW public.orders_vw
AS SELECT order_id,
    customer_id,
    employee_id,
    order_date,
    required_date,
    shipped_date,
    ship_via,
    freight,
    ship_name,
    ship_address,
    ship_city,
    ship_region,
    ship_postal_code,
    ship_country
   FROM orders;

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

-- public.catalog_edge_type definition

-- Drop table

-- DROP TABLE public.catalog_edge_types;

CREATE TABLE public.catalog_edge_types (
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
CONSTRAINT catalog_edge_types_pkey PRIMARY KEY (id),
CONSTRAINT catalog_edge_types_catalog_node_type_fk FOREIGN KEY (source_node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
CONSTRAINT catalog_edge_types_catalog_node_type_fk_1 FOREIGN KEY (target_node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
CONSTRAINT catalog_edge_types_tenants_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

-- public.catalog_node definition
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
    core_id uuid NULL,
    tenant_id uuid NOT NULL,
    is_alpha bool DEFAULT false NULL,
    CONSTRAINT catalog_node_pk PRIMARY KEY (id),
    CONSTRAINT catalog_node_unique UNIQUE (tenant_datasource_id, node_type_id, qualified_path),
    CONSTRAINT catalog_node_catalog_node_type_fk FOREIGN KEY (node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT catalog_node_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT catalog_node_tenants_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX catalog_node_node_type_id_idx ON public.catalog_node USING btree (node_type_id);
CREATE INDEX catalog_node_tenant_datasource_id_idx ON public.catalog_node USING btree (tenant_datasource_id);


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
CONSTRAINT catalog_edge_catalog_edge_types_fk FOREIGN KEY (edge_type_id) REFERENCES public.catalog_edge_types(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
CONSTRAINT catalog_edge_catalog_sourcee_node_fk FOREIGN KEY (source_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
CONSTRAINT catalog_edge_catalog_target_node_fk FOREIGN KEY (target_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX catalog_edge_tenant_datasource_id_idx ON public.catalog_edge USING btree (tenant_datasource_id, source_node_id, edge_type_id, target_node_id);

-- Temporary tables for data staging
CREATE SCHEMA IF NOT EXISTS public_temp;

CREATE TABLE public_temp.catalog_node (
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
core_id uuid NULL,
tenant_id uuid NOT NULL,
is_alpha bool DEFAULT false NULL,
CONSTRAINT temp_catalog_node_pk PRIMARY KEY (id),
CONSTRAINT temp_catalog_node_unique UNIQUE (tenant_datasource_id, node_type_id, qualified_path),
CONSTRAINT temp_catalog_node_catalog_node_type_fk FOREIGN KEY (node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
CONSTRAINT temp_catalog_node_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX temp_catalog_node_node_type_id_idx ON public_temp.catalog_node USING btree (node_type_id);
CREATE INDEX temp_catalog_node_tenant_datasource_id_idx ON public_temp.catalog_node USING btree (tenant_datasource_id);

CREATE TABLE public_temp.catalog_edge (
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
CONSTRAINT temp_catalog_edge_pk PRIMARY KEY (id),
CONSTRAINT temp_catalog_edge_unique UNIQUE (tenant_datasource_id, source_node_id, edge_type_id, target_node_id),
CONSTRAINT temp_catalog_edge_catalog_edge_types_fk FOREIGN KEY (edge_type_id) REFERENCES public.catalog_edge_types(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
CONSTRAINT temp_catalog_edge_catalog_sourcee_node_fk FOREIGN KEY (source_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
CONSTRAINT temp_catalog_edge_catalog_target_node_fk FOREIGN KEY (target_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX temp_catalog_edge_tenant_datasource_id_idx ON public_temp.catalog_edge USING btree (tenant_datasource_id, source_node_id, edge_type_id, target_node_id);

-- public.dashboards definition
CREATE TABLE public.dashboards (
    id varchar(255) NOT NULL,
    name varchar(255) NOT NULL,
    description text NULL,
    widgets jsonb NULL,
    layout varchar(50) NULL DEFAULT 'grid'::character varying,
    theme varchar(50) NULL DEFAULT 'light'::character varying,
    is_public bool NULL DEFAULT false,
    created_by varchar(255) NOT NULL,
    created_at timestamptz NULL DEFAULT now(),
    updated_at timestamptz NULL DEFAULT now(),
    CONSTRAINT dashboards_pk PRIMARY KEY (id)
);

-- Create index for faster queries
CREATE INDEX dashboards_created_by_idx ON public.dashboards USING btree (created_by);
CREATE INDEX dashboards_is_public_idx ON public.dashboards USING btree (is_public);
CREATE INDEX dashboards_updated_at_idx ON public.dashboards USING btree (updated_at);
