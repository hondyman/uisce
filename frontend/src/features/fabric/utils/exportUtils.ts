import { IPWhitelistEntry, Tenant, TenantIPSummary } from '../types/ipWhitelist';

export interface ExportData {
  exportDate: string;
  totalEntries: number;
  totalTenants: number;
  data: any[];
}

export const exportToCSV = (data: any[], filename: string) => {
  if (data.length === 0) return;

  const headers = Object.keys(data[0]);
  const csvContent = [
    headers.join(','),
    ...data.map(row => 
      headers.map(header => {
        const value = row[header];
        // Handle arrays and objects
        if (Array.isArray(value)) {
          return `"${value.join(', ')}"`;
        }
        if (typeof value === 'object' && value !== null) {
          return `"${JSON.stringify(value)}"`;
        }
        // Escape quotes and wrap in quotes if contains comma
        const stringValue = String(value || '');
        if (stringValue.includes(',') || stringValue.includes('"')) {
          return `"${stringValue.replace(/"/g, '""')}"`;
        }
        return stringValue;
      }).join(',')
    )
  ].join('\n');

  downloadFile(csvContent, `${filename}.csv`, 'text/csv');
};

export const exportToJSON = (data: ExportData, filename: string) => {
  const jsonContent = JSON.stringify(data, null, 2);
  downloadFile(jsonContent, `${filename}.json`, 'application/json');
};

const downloadFile = (content: string, filename: string, contentType: string) => {
  const blob = new Blob([content], { type: contentType });
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
};

export const prepareIPExportData = (
  entries: IPWhitelistEntry[], 
  tenants: Tenant[]
): ExportData => {
  const exportData = entries.map(entry => ({
    ipAddress: entry.ipAddress,
    label: entry.label || '',
    description: entry.description || '',
    assignedTenants: entry.tenantIds.map(id => 
      tenants.find(t => t.id === id)?.displayName || id
    ).join(', '),
    tenantCount: entry.tenantIds.length,
    status: entry.isActive !== false ? 'Active' : 'Inactive',
    createdAt: entry.createdAt || '',
    updatedAt: entry.updatedAt || ''
  }));

  return {
    exportDate: new Date().toISOString(),
    totalEntries: entries.length,
    totalTenants: tenants.length,
    data: exportData
  };
};

export const prepareTenantExportData = (
  tenantSummaries: TenantIPSummary[]
): ExportData => {
  const exportData = tenantSummaries.map(summary => ({
    tenantId: summary.tenant.id,
    tenantName: summary.tenant.displayName,
    tenantCode: summary.tenant.tenant_code || '',
    totalIPs: summary.totalIPs,
    activeIPs: summary.activeIPs,
    ipAddresses: summary.ipAddresses.map(ip => ip.ipAddress).join(', '),
    ipLabels: summary.ipAddresses.map(ip => ip.label || '').filter(Boolean).join(', ')
  }));

  return {
    exportDate: new Date().toISOString(),
    totalEntries: tenantSummaries.length,
    totalTenants: tenantSummaries.length,
    data: exportData
  };
};

export const exportIPWhitelistReport = (
  entries: IPWhitelistEntry[],
  tenants: Tenant[],
  format: 'csv' | 'json' = 'csv'
) => {
  const exportData = prepareIPExportData(entries, tenants);
  const timestamp = new Date().toISOString().split('T')[0];
  const filename = `ip-whitelist-report-${timestamp}`;

  if (format === 'csv') {
    exportToCSV(exportData.data, filename);
  } else {
    exportToJSON(exportData, filename);
  }
};

export const exportTenantReport = (
  tenantSummaries: TenantIPSummary[],
  format: 'csv' | 'json' = 'csv'
) => {
  const exportData = prepareTenantExportData(tenantSummaries);
  const timestamp = new Date().toISOString().split('T')[0];
  const filename = `tenant-ip-report-${timestamp}`;

  if (format === 'csv') {
    exportToCSV(exportData.data, filename);
  } else {
    exportToJSON(exportData, filename);
  }
};
