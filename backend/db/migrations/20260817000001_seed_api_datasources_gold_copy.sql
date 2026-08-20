-- Migration: Seed API Datasources and Endpoints for Gold Copy Tenant
-- Purpose: Initialize catalog_node_type, catalog_edge_type, and seed Salesforce & ServiceNow API inventories in Gold Copy
-- Date: 2026-08-17

BEGIN;

DO $$
DECLARE
    v_gold_tenant_id UUID;
    
    v_type_ds_id UUID;
    v_type_res_id UUID;
    v_type_ep_id UUID;
    v_type_f_id UUID;

    v_edge_type_res_id UUID;
    v_edge_type_ep_id UUID;
    v_edge_type_f_id UUID;

    v_sf_ds_id UUID := 'a1000000-0000-0000-0000-000000000001';
    v_sf_res_acc UUID := 'a1000000-0000-0000-0000-000000000002';
    v_sf_ep_get_acc UUID := 'a1000000-0000-0000-0000-000000000003';
    v_sf_ep_post_acc UUID := 'a1000000-0000-0000-0000-000000000004';
    v_sf_f_id UUID := 'a1000000-0000-0000-0000-000000000005';
    v_sf_f_name UUID := 'a1000000-0000-0000-0000-000000000006';
    v_sf_f_rev UUID := 'a1000000-0000-0000-0000-000000000007';
    v_sf_f_ind UUID := 'a1000000-0000-0000-0000-000000000008';

    v_now_ds_id UUID := 'a2000000-0000-0000-0000-000000000001';
    v_now_res_inc UUID := 'a2000000-0000-0000-0000-000000000002';
    v_now_ep_get_inc UUID := 'a2000000-0000-0000-0000-000000000003';
    v_now_ep_post_inc UUID := 'a2000000-0000-0000-0000-000000000004';
    v_now_f_sys_id UUID := 'a2000000-0000-0000-0000-000000000005';
    v_now_f_num UUID := 'a2000000-0000-0000-0000-000000000006';
    v_now_f_desc UUID := 'a2000000-0000-0000-0000-000000000007';
    v_now_f_state UUID := 'a2000000-0000-0000-0000-000000000008';
    v_now_f_pri UUID := 'a2000000-0000-0000-0000-000000000009';

    v_sem_cust_id UUID;
BEGIN
    SELECT id INTO v_gold_tenant_id FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant_id IS NULL THEN
        SELECT id INTO v_gold_tenant_id FROM public.tenants ORDER BY created_at LIMIT 1;
    END IF;

    IF v_gold_tenant_id IS NOT NULL THEN
        -- 1. Ensure Node Types
        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config, is_active)
        VALUES
            (v_gold_tenant_id, 'api_datasource', 'API Datasource Connection / Service Base', '{"category": "datasource", "protocol": "http"}'::jsonb, true),
            (v_gold_tenant_id, 'api_resource', 'API Resource / Object Group (e.g. Account, Incident)', '{"category": "resource"}'::jsonb, true),
            (v_gold_tenant_id, 'api_endpoint', 'API Operation / HTTP Endpoint (e.g. GET /sobjects/Account)', '{"category": "endpoint"}'::jsonb, true),
            (v_gold_tenant_id, 'api_field', 'API Payload / Parameter Field', '{"category": "field"}'::jsonb, true)
        ON CONFLICT (tenant_id, catalog_type_name) DO UPDATE SET
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            is_active = EXCLUDED.is_active;

        SELECT id INTO v_type_ds_id FROM catalog_node_types WHERE tenant_id = v_gold_tenant_id AND catalog_type_name = 'api_datasource';
        SELECT id INTO v_type_res_id FROM catalog_node_types WHERE tenant_id = v_gold_tenant_id AND catalog_type_name = 'api_resource';
        SELECT id INTO v_type_ep_id FROM catalog_node_types WHERE tenant_id = v_gold_tenant_id AND catalog_type_name = 'api_endpoint';
        SELECT id INTO v_type_f_id FROM catalog_node_types WHERE tenant_id = v_gold_tenant_id AND catalog_type_name = 'api_field';

        -- 2. Ensure Edge Types
        INSERT INTO catalog_edge_types (tenant_id, edge_type_name, description, is_active)
        VALUES
            (v_gold_tenant_id::text, 'contains_resource', 'Datasource contains API Resource', true),
            (v_gold_tenant_id::text, 'contains_endpoint', 'Resource contains API Endpoint', true),
            (v_gold_tenant_id::text, 'contains_field', 'Endpoint contains Payload Field', true)
        ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET
            description = EXCLUDED.description,
            is_active = EXCLUDED.is_active;

        SELECT id INTO v_edge_type_res_id FROM catalog_edge_types WHERE tenant_id = v_gold_tenant_id::text AND edge_type_name = 'contains_resource';
        SELECT id INTO v_edge_type_ep_id FROM catalog_edge_types WHERE tenant_id = v_gold_tenant_id::text AND edge_type_name = 'contains_endpoint';
        SELECT id INTO v_edge_type_f_id FROM catalog_edge_types WHERE tenant_id = v_gold_tenant_id::text AND edge_type_name = 'contains_field';

        -- -------------------------------------------------------------
        -- A. SALESFORCE REST API
        -- -------------------------------------------------------------
        -- 1. Datasource Node
        INSERT INTO catalog_node (
            id, tenant_id, node_name, qualified_path, node_type_id, description,
            config, properties, is_alpha, is_active, created_at, updated_at
        ) VALUES (
            v_sf_ds_id, v_gold_tenant_id, 'Salesforce REST API', '/salesforce_api',
            v_type_ds_id, 'Enterprise CRM Salesforce REST API service catalog',
            '{"service_type": "salesforce", "default_auth": "oauth2_bearer", "default_base_url": "https://api.salesforce.com"}'::jsonb,
            '{"protocol": "REST", "version": "v58.0"}'::jsonb,
            true, true, NOW(), NOW()
        ) ON CONFLICT (id) DO UPDATE SET node_name = EXCLUDED.node_name, config = EXCLUDED.config, properties = EXCLUDED.properties, node_type_id = EXCLUDED.node_type_id;

        -- 2. Resource Node: Account
        INSERT INTO catalog_node (
            id, tenant_id, node_name, qualified_path, node_type_id, parent_id, description,
            config, properties, is_alpha, is_active, created_at, updated_at
        ) VALUES (
            v_sf_res_acc, v_gold_tenant_id, 'Account', '/salesforce_api/Account',
            v_type_res_id, v_sf_ds_id, 'Salesforce Account entity resource',
            '{"object_name": "Account"}'::jsonb,
            '{"category": "CRM"}'::jsonb,
            true, true, NOW(), NOW()
        ) ON CONFLICT (id) DO UPDATE SET node_name = EXCLUDED.node_name, config = EXCLUDED.config, node_type_id = EXCLUDED.node_type_id;

        -- 3. Endpoint: GET /services/data/v58.0/sobjects/Account
        INSERT INTO catalog_node (
            id, tenant_id, node_name, qualified_path, node_type_id, parent_id, description,
            config, properties, is_alpha, is_active, created_at, updated_at
        ) VALUES (
            v_sf_ep_get_acc, v_gold_tenant_id, 'GET /sobjects/Account', '/salesforce_api/Account/GET_Account',
            v_type_ep_id, v_sf_res_acc, 'Retrieve Accounts query from Salesforce',
            '{"method": "GET", "path_template": "/services/data/v58.0/query/?q=SELECT+Id,Name,AnnualRevenue,Industry+FROM+Account", "response_root": "$.records[*]"}'::jsonb,
            '{"operation": "READ", "data_type": "json_array"}'::jsonb,
            true, true, NOW(), NOW()
        ) ON CONFLICT (id) DO UPDATE SET node_name = EXCLUDED.node_name, config = EXCLUDED.config, node_type_id = EXCLUDED.node_type_id;

        -- 4. Endpoint: POST /services/data/v58.0/sobjects/Account
        INSERT INTO catalog_node (
            id, tenant_id, node_name, qualified_path, node_type_id, parent_id, description,
            config, properties, is_alpha, is_active, created_at, updated_at
        ) VALUES (
            v_sf_ep_post_acc, v_gold_tenant_id, 'POST /sobjects/Account', '/salesforce_api/Account/POST_Account',
            v_type_ep_id, v_sf_res_acc, 'Create a new Account in Salesforce',
            '{"method": "POST", "path_template": "/services/data/v58.0/sobjects/Account", "response_root": "$"}'::jsonb,
            '{"operation": "CREATE", "data_type": "json_object"}'::jsonb,
            true, true, NOW(), NOW()
        ) ON CONFLICT (id) DO UPDATE SET node_name = EXCLUDED.node_name, config = EXCLUDED.config, node_type_id = EXCLUDED.node_type_id;

        -- 5. Fields for GET Account
        INSERT INTO catalog_node (
            id, tenant_id, node_name, qualified_path, node_type_id, parent_id, description,
            config, properties, is_alpha, is_active, created_at, updated_at
        ) VALUES
            (v_sf_f_id, v_gold_tenant_id, 'Id', '/salesforce_api/Account/GET_Account/Id', v_type_f_id, v_sf_ep_get_acc, 'Salesforce Account ID', '{"json_path": "$.Id"}'::jsonb, '{"data_type": "uuid", "is_primary_key": true}'::jsonb, true, true, NOW(), NOW()),
            (v_sf_f_name, v_gold_tenant_id, 'Name', '/salesforce_api/Account/GET_Account/Name', v_type_f_id, v_sf_ep_get_acc, 'Account Company Name', '{"json_path": "$.Name"}'::jsonb, '{"data_type": "varchar"}'::jsonb, true, true, NOW(), NOW()),
            (v_sf_f_rev, v_gold_tenant_id, 'AnnualRevenue', '/salesforce_api/Account/GET_Account/AnnualRevenue', v_type_f_id, v_sf_ep_get_acc, 'Reported Annual Revenue', '{"json_path": "$.AnnualRevenue"}'::jsonb, '{"data_type": "numeric"}'::jsonb, true, true, NOW(), NOW()),
            (v_sf_f_ind, v_gold_tenant_id, 'Industry', '/salesforce_api/Account/GET_Account/Industry', v_type_f_id, v_sf_ep_get_acc, 'Account Industry sector', '{"json_path": "$.Industry"}'::jsonb, '{"data_type": "varchar"}'::jsonb, true, true, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET node_name = EXCLUDED.node_name, config = EXCLUDED.config, properties = EXCLUDED.properties, node_type_id = EXCLUDED.node_type_id;


        -- -------------------------------------------------------------
        -- B. SERVICENOW TABLE API
        -- -------------------------------------------------------------
        -- 1. Datasource Node
        INSERT INTO catalog_node (
            id, tenant_id, node_name, qualified_path, node_type_id, description,
            config, properties, is_alpha, is_active, created_at, updated_at
        ) VALUES (
            v_now_ds_id, v_gold_tenant_id, 'ServiceNow Table API', '/servicenow_api',
            v_type_ds_id, 'ServiceNow ITSM Table API service catalog',
            '{"service_type": "servicenow", "default_auth": "basic_auth", "default_base_url": "https://instance.service-now.com"}'::jsonb,
            '{"protocol": "REST", "version": "v1"}'::jsonb,
            true, true, NOW(), NOW()
        ) ON CONFLICT (id) DO UPDATE SET node_name = EXCLUDED.node_name, config = EXCLUDED.config, properties = EXCLUDED.properties, node_type_id = EXCLUDED.node_type_id;

        -- 2. Resource Node: Incident
        INSERT INTO catalog_node (
            id, tenant_id, node_name, qualified_path, node_type_id, parent_id, description,
            config, properties, is_alpha, is_active, created_at, updated_at
        ) VALUES (
            v_now_res_inc, v_gold_tenant_id, 'incident', '/servicenow_api/incident',
            v_type_res_id, v_now_ds_id, 'ServiceNow Incident Management resource',
            '{"table_name": "incident"}'::jsonb,
            '{"category": "ITSM"}'::jsonb,
            true, true, NOW(), NOW()
        ) ON CONFLICT (id) DO UPDATE SET node_name = EXCLUDED.node_name, config = EXCLUDED.config, node_type_id = EXCLUDED.node_type_id;

        -- 3. Endpoint: GET /api/now/table/incident
        INSERT INTO catalog_node (
            id, tenant_id, node_name, qualified_path, node_type_id, parent_id, description,
            config, properties, is_alpha, is_active, created_at, updated_at
        ) VALUES (
            v_now_ep_get_inc, v_gold_tenant_id, 'GET /api/now/table/incident', '/servicenow_api/incident/GET_incident',
            v_type_ep_id, v_now_res_inc, 'List active incidents from ServiceNow',
            '{"method": "GET", "path_template": "/api/now/table/incident?sysparm_limit=50", "response_root": "$.result[*]"}'::jsonb,
            '{"operation": "READ", "data_type": "json_array"}'::jsonb,
            true, true, NOW(), NOW()
        ) ON CONFLICT (id) DO UPDATE SET node_name = EXCLUDED.node_name, config = EXCLUDED.config, node_type_id = EXCLUDED.node_type_id;

        -- 4. Fields for GET incident
        INSERT INTO catalog_node (
            id, tenant_id, node_name, qualified_path, node_type_id, parent_id, description,
            config, properties, is_alpha, is_active, created_at, updated_at
        ) VALUES
            (v_now_f_sys_id, v_gold_tenant_id, 'sys_id', '/servicenow_api/incident/GET_incident/sys_id', v_type_f_id, v_now_ep_get_inc, 'ServiceNow Unique Record ID', '{"json_path": "$.sys_id"}'::jsonb, '{"data_type": "uuid", "is_primary_key": true}'::jsonb, true, true, NOW(), NOW()),
            (v_now_f_num, v_gold_tenant_id, 'number', '/servicenow_api/incident/GET_incident/number', v_type_f_id, v_now_ep_get_inc, 'Incident Tracking Number (e.g. INC0012345)', '{"json_path": "$.number"}'::jsonb, '{"data_type": "varchar"}'::jsonb, true, true, NOW(), NOW()),
            (v_now_f_desc, v_gold_tenant_id, 'short_description', '/servicenow_api/incident/GET_incident/short_description', v_type_f_id, v_now_ep_get_inc, 'Short description summary', '{"json_path": "$.short_description"}'::jsonb, '{"data_type": "varchar"}'::jsonb, true, true, NOW(), NOW()),
            (v_now_f_state, v_gold_tenant_id, 'state', '/servicenow_api/incident/GET_incident/state', v_type_f_id, v_now_ep_get_inc, 'Incident state code (1=New, 2=In Progress, 6=Resolved)', '{"json_path": "$.state"}'::jsonb, '{"data_type": "int"}'::jsonb, true, true, NOW(), NOW()),
            (v_now_f_pri, v_gold_tenant_id, 'priority', '/servicenow_api/incident/GET_incident/priority', v_type_f_id, v_now_ep_get_inc, 'Urgency priority level (1=Critical, 2=High, 3=Moderate)', '{"json_path": "$.priority"}'::jsonb, '{"data_type": "int"}'::jsonb, true, true, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET node_name = EXCLUDED.node_name, config = EXCLUDED.config, properties = EXCLUDED.properties, node_type_id = EXCLUDED.node_type_id;

        -- -------------------------------------------------------------
        -- C. HIERARCHICAL EDGES (contains_resource, contains_endpoint, contains_field)
        -- -------------------------------------------------------------
        -- Salesforce Edges
        INSERT INTO catalog_edge (id, tenant_id, source_node_id, target_node_id, edge_type_id, relationship_type, created_at, updated_at)
        VALUES
            (gen_random_uuid(), v_gold_tenant_id, v_sf_ds_id, v_sf_res_acc, v_edge_type_res_id, 'contains_resource', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_sf_res_acc, v_sf_ep_get_acc, v_edge_type_ep_id, 'contains_endpoint', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_sf_res_acc, v_sf_ep_post_acc, v_edge_type_ep_id, 'contains_endpoint', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_sf_ep_get_acc, v_sf_f_id, v_edge_type_f_id, 'contains_field', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_sf_ep_get_acc, v_sf_f_name, v_edge_type_f_id, 'contains_field', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_sf_ep_get_acc, v_sf_f_rev, v_edge_type_f_id, 'contains_field', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_sf_ep_get_acc, v_sf_f_ind, v_edge_type_f_id, 'contains_field', NOW(), NOW())
        ON CONFLICT DO NOTHING;

        -- ServiceNow Edges
        INSERT INTO catalog_edge (id, tenant_id, source_node_id, target_node_id, edge_type_id, relationship_type, created_at, updated_at)
        VALUES
            (gen_random_uuid(), v_gold_tenant_id, v_now_ds_id, v_now_res_inc, v_edge_type_res_id, 'contains_resource', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_now_res_inc, v_now_ep_get_inc, v_edge_type_ep_id, 'contains_endpoint', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_now_ep_get_inc, v_now_f_sys_id, v_edge_type_f_id, 'contains_field', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_now_ep_get_inc, v_now_f_num, v_edge_type_f_id, 'contains_field', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_now_ep_get_inc, v_now_f_desc, v_edge_type_f_id, 'contains_field', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_now_ep_get_inc, v_now_f_state, v_edge_type_f_id, 'contains_field', NOW(), NOW()),
            (gen_random_uuid(), v_gold_tenant_id, v_now_ep_get_inc, v_now_f_pri, v_edge_type_f_id, 'contains_field', NOW(), NOW())
        ON CONFLICT DO NOTHING;

        -- -------------------------------------------------------------
        -- D. Link to existing Semantic Terms (e.g. Customer ID)
        -- -------------------------------------------------------------
        SELECT id INTO v_sem_cust_id FROM catalog_node 
        WHERE tenant_id = v_gold_tenant_id 
          AND (node_name ILIKE '%customer id%' OR node_name ILIKE '%customer_id%' OR node_name ILIKE '%account id%')
        LIMIT 1;

        IF v_sem_cust_id IS NOT NULL THEN
            INSERT INTO catalog_edge (id, tenant_id, source_node_id, target_node_id, edge_type_id, relationship_type, created_at, updated_at)
            VALUES (gen_random_uuid(), v_gold_tenant_id, v_sf_f_id, v_sem_cust_id, '0434ca1a-6543-42d3-9fce-f0b58b5fba34', 'has_context', NOW(), NOW())
            ON CONFLICT DO NOTHING;
        END IF;

    END IF;
END $$;

COMMIT;
