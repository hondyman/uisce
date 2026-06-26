import { gql } from '@apollo/client';

// CORRECTED: Tenant mutations now include all editable fields
export const CREATE_TENANT = gql`
  mutation CreateTenant($display_name: String!, $description: String, $is_active: Boolean!) {
    insert_tenants_one(object: {display_name: $display_name, description: $description, is_active: $is_active}) {
      id
    }
  }
`;

export const UPDATE_TENANT = gql`
  mutation UpdateTenant($id: uuid!, $display_name: String!, $description: String, $is_active: Boolean!) {
    update_tenants_by_pk(pk_columns: { id: $id }, _set: { display_name: $display_name, description: $description, is_active: $is_active }) {
      id
    }
  }
`;

export const DELETE_TENANT = gql`
  mutation DeleteTenant($id: uuid!) {
    delete_tenants_by_pk(id: $id) {
      id
    }
  }
`;

// CORRECTED: Tenant Instance mutations now include all editable fields
export const CREATE_TENANT_INSTANCE = gql`
  mutation CreateTenantInstance($tenant_id: uuid!, $instance_name: String!, $display_name: String!, $description: String, $url: String, $is_active: Boolean!, $config: jsonb!) {
    insert_tenant_instance_one(object: {tenant_id: $tenant_id, instance_name: $instance_name, display_name: $display_name, description: $description, url: $url, is_active: $is_active, config: $config}) {
      id
    }
  }
`;

export const UPDATE_TENANT_INSTANCE = gql`
  mutation UpdateTenantInstance($id: uuid!, $instance_name: String, $display_name: String!, $description: String, $url: String, $is_active: Boolean!, $config: jsonb!) {
    update_tenant_instance_by_pk(pk_columns: { id: $id }, _set: { instance_name: $instance_name, display_name: $display_name, description: $description, url: $url, is_active: $is_active, config: $config }) {
      id
    }
  }
`;

export const DELETE_TENANT_INSTANCE = gql`
  mutation DeleteTenantInstance($id: uuid!) {
    delete_tenant_instance_by_pk(id: $id) {
      id
    }
  }
`;

export const ADD_TENANT_PRODUCT = gql`
  mutation AddTenantProduct($tenant_id: uuid!, $product_id: uuid, $alpha_product_id: uuid, $version: Float!, $is_active: Boolean!) {
    insert_tenant_product_one(object: { tenant_id: $tenant_id, product_id: $product_id, alpha_product_id: $alpha_product_id, version: $version, is_active: $is_active }) {
      id
    }
  }
`;

export const DELETE_TENANT_PRODUCT = gql`
  mutation DeleteTenantProduct($id: uuid!) {
    delete_tenant_product_by_pk(id: $id) {
      id
    }
  }
`;

export const UPDATE_TENANT_PRODUCT = gql`
  mutation UpdateTenantProduct($id: uuid!, $version: Float!, $tenant_id: uuid, $product_id: uuid, $is_active: Boolean) {
    update_tenant_product_by_pk(pk_columns: { id: $id }, _set: { version: $version, tenant_id: $tenant_id, product_id: $product_id, is_active: $is_active }) {
      id
      version
      tenant_id
      product_id
      is_active
    }
  }
`;



export const CREATE_DATASOURCE = gql`
  mutation CreateDatasource($datasource_name: String!, $datasource_code: String!, $datasource_type: String, $is_active: Boolean!, $config: jsonb) {
    insert_alpha_datasource_one(object: { 
      datasource_name: $datasource_name, 
      datasource_code: $datasource_code, 
      datasource_type: $datasource_type, 
      is_active: $is_active,
      config: $config
    }) {
      id
      datasource_name
      datasource_code
    }
  }
`;

export const UPDATE_DATASOURCE = gql`
  mutation UpdateDatasource($id: uuid!, $datasource_name: String!, $datasource_code: String!, $datasource_type: String, $is_active: Boolean!, $config: jsonb) {
    update_alpha_datasource_by_pk(
      pk_columns: { id: $id }, 
      _set: { 
        datasource_name: $datasource_name, 
        datasource_code: $datasource_code, 
        datasource_type: $datasource_type, 
        is_active: $is_active,
        config: $config
      }
    ) {
      id
      datasource_name
      datasource_code
    }
  }
`;

export const DELETE_DATASOURCE = gql`
  mutation DeleteDatasource($id: uuid!) {
    delete_alpha_datasource_by_pk(id: $id) {
      id
    }
  }
`;


export const CREATE_CONNECTION = gql`
  mutation CreateConnection($object: connections_insert_input!) {
    insert_connections_one(object: $object) {
      id
    }
  }
`;

export const UPDATE_CONNECTION = gql`
  mutation UpdateConnection($id: uuid!, $object: connections_set_input!) {
    update_connections_by_pk(pk_columns: { id: $id }, _set: $object) {
      id
    }
  }
`;

export const DELETE_CONNECTION = gql`
  mutation DeleteConnection($id: uuid!) {
    delete_connections_by_pk(id: $id) {
      id
    }
  }
`;

export const ADD_TENANT_PRODUCT_DATASOURCE = gql`
  mutation AddTenantProductDatasource(
    $tenant_product_id: uuid!,
    $tenant_instance_id: uuid,
    $alpha_tenant_instance_id: uuid,
    $config: jsonb!,
    $is_active: Boolean!,
    $source_name: String!,
    $connection_id: uuid
  ) {
    insert_tenant_product_datasource_one(
      object: {
        tenant_product_id: $tenant_product_id,
        tenant_instance_id: $tenant_instance_id,
        alpha_tenant_instance_id: $alpha_tenant_instance_id,
        config: $config,
        is_active: $is_active,
        source_name: $source_name,
        connection_id: $connection_id
      }
    ) {
      id
      source_name
      tenant_instance_id
      __typename
    }
  }
`;

export const UPDATE_TENANT_PRODUCT_DATASOURCE = gql`
  mutation UpdateTenantProductDatasource(
    $id: uuid!,
    $tenant_product_id: uuid,
    $tenant_instance_id: uuid,
    $alpha_tenant_instance_id: uuid,
    $config: jsonb,
    $is_active: Boolean,
    $source_name: String,
    $connection_id: uuid
  ) {
    update_tenant_product_datasource_by_pk(
      pk_columns: { id: $id },
      _set: {
        tenant_product_id: $tenant_product_id,
        tenant_instance_id: $tenant_instance_id,
        alpha_tenant_instance_id: $alpha_tenant_instance_id,
        config: $config,
        is_active: $is_active,
        source_name: $source_name,
        connection_id: $connection_id
      }
    ) {
      id
      __typename
    }
  }
`;








export const TEST_DATASOURCE_CONNECTION = gql`
  mutation TestDatasourceConnection($connection_details: String!) {
    test_datasource_connection(connection_details: $connection_details) {
      success
      message
    }
  }
`;

export const SCAN_DATASOURCE = gql`
  mutation ScanDatasource($tenant_instance_id: uuid!) {
    scan_datasource(tenant_instance_id: $tenant_instance_id) {
      status
      message
      results {
        tenant_instance_id
        name
        success
        error
        added
        updated
        removed
      }
    }
  }
`;

export const UPDATE_TENANT_PRODUCT_DATASOURCE_LINKING = gql`
  mutation UpdateTenantProductDatasourceLinking(
    $id: uuid!,
    $tenant_instance_id: uuid,
    $alpha_tenant_instance_id: uuid,
    $config: jsonb,
    $is_active: Boolean,
    $source_name: String,
    $connection_id: uuid
  ) {
    update_tenant_product_datasource_by_pk(
      pk_columns: { id: $id },
      _set: {
        tenant_instance_id: $tenant_instance_id,
        alpha_tenant_instance_id: $alpha_tenant_instance_id,
        config: $config,
        is_active: $is_active,
        source_name: $source_name,
        connection_id: $connection_id
      }
    ) {
      id
      __typename
    }
  }
`;

// Ultra-minimal mutation to just unlink/link a connection. Only updates connection_id.
export const UPDATE_TPD_CONNECTION_ONLY = gql`
  mutation UpdateTpdConnectionOnly($id: uuid!, $connection_id: uuid) {
    update_tenant_product_datasource_by_pk(
      pk_columns: { id: $id },
      _set: {
        connection_id: $connection_id
      }
    ) {
      id
      __typename
    }
  }
`;