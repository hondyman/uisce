import { gql, useQuery } from '@apollo/client';
import { GET_ERD_CHART } from './datasourceQueries';

// Query for tenants list with tenant-level products
export const GET_TENANTS = gql`
  query GetTenants {
    tenants {
      id
      display_name
      description
      is_active
      gold_copy
      tenant_instances {
        id
        instance_name
        display_name

        is_active
        url
        config
        tenant_id
        __typename
      }
      tenant_products {
        id
        version
        alpha_product_id
        is_active
        __typename
        alpha_product {
          id
          is_active

          product_name
          __typename
        }
      }
      __typename
    }
  }
`;

export const GET_SCOPED_TENANT = gql`
  query GetScopedTenant($tenantId: uuid!) {
    tenants(where: { id: { _eq: $tenantId } }) {
      id
      display_name
      description
      is_active
      gold_copy
      tenant_instances {
        id
        instance_name
        display_name

        is_active
        url
        config
        tenant_id
        __typename
      }
      tenant_products {
        id
        version
        alpha_product_id
        is_active
        __typename
        alpha_product {
          id
          is_active

          product_name
          __typename
        }
        tenant_product_datasources {
          id
          tenant_product_id
          tenant_instance_id
          alpha_tenant_instance_id
          connection_id
          source_name
          is_active
          config
          __typename
        }
      }
      __typename
    }
  }
`;

// Get products from the gold copy tenant (Uisce)
export const GET_GOLD_COPY_PRODUCTS = gql`
  query GetGoldCopyProducts {
    tenants(where: { gold_copy: { _eq: true } }) {
      id
      display_name
      tenant_products {
        id
        alpha_product {
          id
          product_name
          description
          __typename
        }
        __typename
      }
    }
  }
`;

// Get registered products for a specific tenant
export const GET_TENANT_REGISTERED_PRODUCTS = gql`
  query GetTenantRegisteredProducts($tenantId: uuid!) {
    tenant_product(where: { tenant_id: { _eq: $tenantId } }) {
      id
      alpha_product_id
      version
      is_active
      alpha_product {
        id
        product_name

        __typename
      }
      __typename
    }
  }
`;

export const GET_ALPHA_DATASOURCES = gql`
  query GetAlphaDatasources {
    alpha_datasource(where: { is_active: { _eq: true } }) {
      id
      display_name
      datasource_code

      __typename
    }
  }
`;




// Optional: Enhanced query with additional metadata for the catalog