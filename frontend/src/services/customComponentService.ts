import { CustomComponent } from '../components/CustomComponentManager/CustomComponentManager';
import resolveApiUrl from '../utils/resolveApiUrl';

interface CustomComponentResponse extends CustomComponent {
  id: string;
}

export const customComponentService = {
  /**
   * List all custom components for a tenant and datasource
   */
  async listComponents(tenantId: string, datasourceId: string): Promise<CustomComponent[]> {
    const response = await fetch(
      resolveApiUrl(`/api/custom-components?tenant_id=${tenantId}&datasource_id=${datasourceId}`),
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to list components: ${response.statusText}`);
    }

    const data = await response.json();
    return data || [];
  },

  /**
   * Get a single custom component
   */
  async getComponent(tenantId: string, datasourceId: string, componentId: string): Promise<CustomComponent> {
    const response = await fetch(
      resolveApiUrl(`/api/custom-components/${componentId}?tenant_id=${tenantId}&datasource_id=${datasourceId}`),
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to get component: ${response.statusText}`);
    }

    return await response.json();
  },

  /**
   * Create a new custom component
   */
  async createComponent(tenantId: string, datasourceId: string, component: CustomComponent): Promise<CustomComponentResponse> {
    const response = await fetch(
      resolveApiUrl(`/api/custom-components?tenant_id=${tenantId}&datasource_id=${datasourceId}`),
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify({
          ...component,
          tenant_id: tenantId,
          datasource_id: datasourceId,
        }),
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to create component: ${response.statusText}`);
    }

    return await response.json();
  },

  /**
   * Update an existing custom component
   */
  async updateComponent(tenantId: string, datasourceId: string, component: CustomComponent): Promise<CustomComponentResponse> {
    const response = await fetch(
      resolveApiUrl(`/api/custom-components/${component.id}?tenant_id=${tenantId}&datasource_id=${datasourceId}`),
      {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify({
          ...component,
          tenant_id: tenantId,
          datasource_id: datasourceId,
        }),
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to update component: ${response.statusText}`);
    }

    return await response.json();
  },

  /**
   * Delete a custom component
   */
  async deleteComponent(tenantId: string, datasourceId: string, componentId: string): Promise<void> {
    const response = await fetch(
      resolveApiUrl(`/api/custom-components/${componentId}?tenant_id=${tenantId}&datasource_id=${datasourceId}`),
      {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to delete component: ${response.statusText}`);
    }
  },

  /**
   * Test a component's API endpoint
   */
  async testComponentAPI(
    tenantId: string,
    datasourceId: string,
    apiEndpoint: string
  ): Promise<any> {
    const response = await fetch(
      resolveApiUrl(`/api/custom-components/test-api?tenant_id=${tenantId}&datasource_id=${datasourceId}`),
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify({ endpoint: apiEndpoint }),
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to test API: ${response.statusText}`);
    }

    return await response.json();
  },

  /**
   * Export components configuration
   */
  async exportComponents(tenantId: string, datasourceId: string): Promise<Blob> {
    const response = await fetch(
      resolveApiUrl(`/api/custom-components/export?tenant_id=${tenantId}&datasource_id=${datasourceId}`),
      {
        method: 'GET',
        headers: {
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to export components: ${response.statusText}`);
    }

    return await response.blob();
  },

  /**
   * Import components configuration
   */
  async importComponents(tenantId: string, datasourceId: string, file: File): Promise<CustomComponent[]> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(
      resolveApiUrl(`/api/custom-components/import?tenant_id=${tenantId}&datasource_id=${datasourceId}`),
      {
        method: 'POST',
        headers: {
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: formData,
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to import components: ${response.statusText}`);
    }

    return await response.json();
  },
};
