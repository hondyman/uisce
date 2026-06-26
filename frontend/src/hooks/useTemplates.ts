import { useState, useCallback, useEffect } from 'react';

// ============================================================================
// Types
// ============================================================================

export interface TemplateParamDef {
  name: string;
  type: 'string' | 'number' | 'bool';
  required: boolean;
  default?: any;
  help?: string;
}

export interface SemanticQuery {
  datasource: string;
  version?: string;
  select: string[];
  filters: any[];
  order_by?: any[];
  limit?: number;
}

export interface SemanticQueryTemplate {
  id: string;
  tenant_id?: string;
  name: string;
  description?: string;
  datasource: string;
  version: string;
  semantic_query: SemanticQuery;
  parameters: TemplateParamDef[];
  created_by: string;
  created_at: string;
  updated_at: string;
  visibility: 'private' | 'team' | 'public';
  tags: string[];
  deprecated: boolean;
}

export interface TemplateRunRequest {
  params: Record<string, any>;
}

export interface TemplateRunResponse {
  datasource: string;
  version: string;
  sql: string;
  rows: any[];
  count: number;
  executed_at: string;
  duration_ms: number;
}

export interface TemplateListResponse {
  templates: SemanticQueryTemplate[];
  total: number;
  page: number;
  per_page: number;
}

export interface TemplateVersion {
  id: string;
  template_id: string;
  version_number: number;
  name: string;
  description: string;
  created_by: string;
  created_at: string;
  is_promoted: boolean;
  promoted_at?: string;
  promoted_by?: string;
  change_message?: string;
}

export interface TemplateVersionsResponse {
  versions: TemplateVersion[];
  total: number;
}

export interface TemplateDiff {
  name_changed: boolean;
  description_changed: boolean;
  query_changed: boolean;
  parameters_changed: boolean;
  changes: Record<string, any>;
}

export interface ApiError {
  error: string;
  message?: string;
}

// ============================================================================
// useTemplates Hook - List and Manage Templates
// ============================================================================

export interface UseTemplatesOptions {
  datasource?: string;
  version?: string;
  search?: string;
  page?: number;
  perPage?: number;
}

export const useTemplates = (options: UseTemplatesOptions = {}) => {
  const [templates, setTemplates] = useState<SemanticQueryTemplate[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTemplates = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const params = new URLSearchParams();
      if (options.datasource) params.append('datasource', options.datasource);
      if (options.version) params.append('version', options.version);
      if (options.search) params.append('search', options.search);
      if (options.page) params.append('page', String(options.page));
      if (options.perPage) params.append('per_page', String(options.perPage));

      const response = await fetch(
        `/api/semantic/templates?${params.toString()}`,
        {
          headers: { 'X-Tenant-ID': getTenantId() },
        }
      );

      if (!response.ok) {
        const errorData = (await response.json()) as ApiError;
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      const data = (await response.json()) as TemplateListResponse;
      setTemplates(data.templates || []);
      setTotal(data.total || 0);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [options]);

  // Auto-fetch on mount or options change
  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  return {
    templates,
    total,
    loading,
    error,
    refetch: fetchTemplates,
  };
};

// ============================================================================
// useTemplate Hook - Get Single Template
// ============================================================================

export const useTemplate = (templateId?: string) => {
  const [template, setTemplate] = useState<SemanticQueryTemplate | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTemplate = useCallback(async () => {
    if (!templateId) {
      setTemplate(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `/api/semantic/templates/${templateId}`,
        {
          headers: { 'X-Tenant-ID': getTenantId() },
        }
      );

      if (!response.ok) {
        const errorData = (await response.json()) as ApiError;
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      const data = (await response.json()) as SemanticQueryTemplate;
      setTemplate(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [templateId]);

  useEffect(() => {
    fetchTemplate();
  }, [fetchTemplate]);

  return {
    template,
    loading,
    error,
    refetch: fetchTemplate,
  };
};

// ============================================================================
// useTemplateCreate Hook - Create New Template
// ============================================================================

export const useTemplateCreate = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const create = useCallback(
    async (template: Partial<SemanticQueryTemplate>) => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch('/api/semantic/templates', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': getTenantId(),
          },
          body: JSON.stringify(template),
        });

        if (!response.ok) {
          const errorData = (await response.json()) as ApiError;
          throw new Error(errorData.error || `HTTP ${response.status}`);
        }

        return await response.json() as SemanticQueryTemplate;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  return {
    create,
    loading,
    error,
  };
};

// ============================================================================
// useTemplateUpdate Hook - Update Existing Template
// ============================================================================

export const useTemplateUpdate = (templateId?: string) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const update = useCallback(
    async (
      changes: Partial<SemanticQueryTemplate>,
      changeMessage?: string
    ) => {
      if (!templateId) {
        throw new Error('Template ID is required for update');
      }

      setLoading(true);
      setError(null);

      try {
        const response = await fetch(
          `/api/semantic/templates/${templateId}`,
          {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json',
              'X-Tenant-ID': getTenantId(),
            },
            body: JSON.stringify({
              ...changes,
              change_message: changeMessage,
            }),
          }
        );

        if (!response.ok) {
          const errorData = (await response.json()) as ApiError;
          throw new Error(errorData.error || `HTTP ${response.status}`);
        }

        return await response.json() as SemanticQueryTemplate;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [templateId]
  );

  return {
    update,
    loading,
    error,
  };
};

// ============================================================================
// useTemplateDelete Hook - Delete Template
// ============================================================================

export const useTemplateDelete = (templateId?: string) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const delete_ = useCallback(async () => {
    if (!templateId) {
      throw new Error('Template ID is required for delete');
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `/api/semantic/templates/${templateId}`,
        {
          method: 'DELETE',
          headers: { 'X-Tenant-ID': getTenantId() },
        }
      );

      if (!response.ok) {
        const errorData = (await response.json()) as ApiError;
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [templateId]);

  return {
    delete: delete_,
    loading,
    error,
  };
};

// ============================================================================
// useTemplateRun Hook - Execute Template with Parameters
// ============================================================================

export const useTemplateRun = (templateId?: string) => {
  const [result, setResult] = useState<TemplateRunResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = useCallback(
    async (params: Record<string, any>) => {
      if (!templateId) {
        throw new Error('Template ID is required for execution');
      }

      setLoading(true);
      setError(null);

      try {
        const response = await fetch(
          `/api/semantic/templates/${templateId}/run`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-Tenant-ID': getTenantId(),
            },
            body: JSON.stringify({ params }),
          }
        );

        if (!response.ok) {
          const errorData = (await response.json()) as ApiError;
          throw new Error(errorData.error || `HTTP ${response.status}`);
        }

        const response_data = await response.json() as TemplateRunResponse;
        setResult(response_data);
        return response_data;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [templateId]
  );

  return {
    result,
    run,
    loading,
    error,
    clear: () => setResult(null),
  };
};

// ============================================================================
// useTemplateVersions Hook - Get Template Version History
// ============================================================================

export const useTemplateVersions = (templateId?: string) => {
  const [versions, setVersions] = useState<TemplateVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchVersions = useCallback(async () => {
    if (!templateId) {
      setVersions([]);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `/api/semantic/templates/${templateId}/versions`,
        {
          headers: { 'X-Tenant-ID': getTenantId() },
        }
      );

      if (!response.ok) {
        const errorData = (await response.json()) as ApiError;
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      const data = (await response.json()) as TemplateVersionsResponse;
      setVersions(data.versions || []);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [templateId]);

  useEffect(() => {
    fetchVersions();
  }, [fetchVersions]);

  return {
    versions,
    loading,
    error,
    refetch: fetchVersions,
  };
};

// ============================================================================
// useTemplateDiff Hook - Compare Two Versions
// ============================================================================

export const useTemplateDiff = (templateId?: string) => {
  const [diff, setDiff] = useState<TemplateDiff | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const compare = useCallback(
    async (fromVersion: number, toVersion: number) => {
      if (!templateId) {
        throw new Error('Template ID is required');
      }

      setLoading(true);
      setError(null);

      try {
        const response = await fetch(
          `/api/semantic/templates/${templateId}/diff`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-Tenant-ID': getTenantId(),
            },
            body: JSON.stringify({
              from_version: fromVersion,
              to_version: toVersion,
            }),
          }
        );

        if (!response.ok) {
          const errorData = (await response.json()) as ApiError;
          throw new Error(errorData.error || `HTTP ${response.status}`);
        }

        const response_data = await response.json() as TemplateDiff;
        setDiff(response_data);
        return response_data;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [templateId]
  );

  return {
    diff,
    compare,
    loading,
    error,
  };
};

// ============================================================================
// useTemplatePromote Hook - Promote Version to Production
// ============================================================================

export const useTemplatePromote = (templateId?: string) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const promote = useCallback(
    async (versionNumber: number) => {
      if (!templateId) {
        throw new Error('Template ID is required');
      }

      setLoading(true);
      setError(null);

      try {
        const response = await fetch(
          `/api/semantic/templates/${templateId}/promote`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-Tenant-ID': getTenantId(),
            },
            body: JSON.stringify({ version_number: versionNumber }),
          }
        );

        if (!response.ok) {
          const errorData = (await response.json()) as ApiError;
          throw new Error(errorData.error || `HTTP ${response.status}`);
        }

        return await response.json() as TemplateVersion;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [templateId]
  );

  return {
    promote,
    loading,
    error,
  };
};

// ============================================================================
// Utility Functions
// ============================================================================

// getTenantId - Extract tenant ID from localStorage or context
function getTenantId(): string {
  // In production, get this from your auth context or localStorage
  const tenantId = localStorage.getItem('tenant_id') || 'tenant-1';
  return tenantId;
}

// formatDuration - Format milliseconds to readable string
export const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
};

// copyToClipboard - Copy text to clipboard
export const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (err) {
    console.error('Failed to copy to clipboard:', err);
    return false;
  }
};

// downloadSQL - Download SQL as text file
export const downloadSQL = (sql: string, templateName: string) => {
  const element = document.createElement('a');
  element.setAttribute(
    'href',
    `data:text/plain;charset=utf-8,${encodeURIComponent(sql)}`
  );
  element.setAttribute('download', `${templateName}.sql`);
  element.style.display = 'none';
  document.body.appendChild(element);
  element.click();
  document.body.removeChild(element);
};

// validateTemplateParameters - Client-side validation
export const validateTemplateParameters = (
  parameters: TemplateParamDef[],
  values: Record<string, any>
): Record<string, string> => {
  const errors: Record<string, string> = {};

  for (const param of parameters) {
    const value = values[param.name];

    if (param.required && (value === undefined || value === '')) {
      errors[param.name] = `${param.name} is required`;
      continue;
    }

    if (value !== undefined && typeof value !== 'object') {
      switch (param.type) {
        case 'number':
          if (isNaN(Number(value))) {
            errors[param.name] = 'Must be a number';
          }
          break;

        case 'bool':
          if (typeof value !== 'boolean') {
            errors[param.name] = 'Must be a boolean';
          }
          break;

        case 'string':
          if (typeof value !== 'string') {
            errors[param.name] = 'Must be a string';
          }
          break;
      }
    }
  }

  return errors;
};

// ============================================================================
// Rule Templates (Phase 4)
// ============================================================================

export interface RuleTemplateParameter {
  name: string;
  dataType: 'STRING' | 'NUMBER' | 'BOOLEAN' | 'DATE';
  defaultValue?: any;
  description?: string;
  isRequired: boolean;
}

export interface RuleTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  businessObject: string;
  parameters: RuleTemplateParameter[];
  parameterSchema: any;
  isPublic: boolean;
  status: 'draft' | 'approved' | 'deprecated';
  usageCount: number;
  baseConfidence: number;
  createdBy: string;
  createdAt: string;
}

export interface RuleTemplatePreview {
  template: RuleTemplate;
  sampleParameters: Record<string, any>;
  previewSteps: any[];
  estimatedConfidence: number;
}

// ============================================================================
// useRuleTemplates Hook - List and Manage Rule Templates
// ============================================================================

export const useRuleTemplates = (businessObject?: string, category?: string) => {
  const [templates, setTemplates] = useState<RuleTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTemplates = useCallback(async () => {
    if (!businessObject) return;

    setLoading(true);
    setError(null);

    try {
      let url = `/api/v1/templates?businessObject=${businessObject}`;
      if (category) {
        url += `&category=${category}`;
      }

      const response = await fetch(url, {
        headers: { 'X-Tenant-ID': getTenantId() },
      });

      if (!response.ok) {
        throw new Error(`Failed to load templates (${response.status})`);
      }

      const data = await response.json();
      setTemplates(Array.isArray(data) ? data : []);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load templates';
      setError(message);
      console.error('Error loading templates:', err);
    } finally {
      setLoading(false);
    }
  }, [businessObject, category]);

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  return {
    templates,
    loading,
    error,
    refetch: fetchTemplates,
  };
};

// ============================================================================
// useRuleTemplate Hook - Get Single Rule Template
// ============================================================================

export const useRuleTemplate = (templateId?: string) => {
  const [template, setTemplate] = useState<RuleTemplate | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTemplate = useCallback(async () => {
    if (!templateId) {
      setTemplate(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/templates/${templateId}`, {
        headers: { 'X-Tenant-ID': getTenantId() },
      });

      if (!response.ok) {
        throw new Error(`Failed to load template (${response.status})`);
      }

      const data = await response.json();
      setTemplate(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load template';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [templateId]);

  useEffect(() => {
    fetchTemplate();
  }, [fetchTemplate]);

  return {
    template,
    loading,
    error,
    refetch: fetchTemplate,
  };
};

// ============================================================================
// useRuleTemplateCreate Hook - Create New Rule Template
// ============================================================================

export const useRuleTemplateCreate = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const create = useCallback(async (template: Partial<RuleTemplate>) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/v1/templates', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': getTenantId(),
          'X-User-ID': localStorage.getItem('userId') || 'user-001',
        },
        body: JSON.stringify(template),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `HTTP ${response.status}`);
      }

      return await response.json() as RuleTemplate;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create template';
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    create,
    loading,
    error,
  };
};

// ============================================================================
// useRuleTemplateInstantiate Hook - Create Rule from Template
// ============================================================================

export const useRuleTemplateInstantiate = (templateId?: string) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const instantiate = useCallback(
    async (ruleName: string, parameters: Record<string, any>) => {
      if (!templateId) {
        throw new Error('Template ID is required');
      }

      setLoading(true);
      setError(null);

      try {
        const response = await fetch(`/api/v1/templates/${templateId}/create-rule`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': getTenantId(),
            'X-User-ID': localStorage.getItem('userId') || 'user-001',
          },
          body: JSON.stringify({
            ruleName,
            parameters,
          }),
        });

        if (!response.ok) {
          const errorData = await response.json();
          throw new Error(errorData.message || `HTTP ${response.status}`);
        }

        return await response.json();
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to instantiate template';
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [templateId]
  );

  return {
    instantiate,
    loading,
    error,
  };
};

// ============================================================================
// useRuleTemplatePreview Hook - Preview Template Instantiation
// ============================================================================

export const useRuleTemplatePreview = (templateId?: string) => {
  const [preview, setPreview] = useState<RuleTemplatePreview | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const generatePreview = useCallback(
    async (parameters: Record<string, any>) => {
      if (!templateId) {
        throw new Error('Template ID is required');
      }

      setLoading(true);
      setError(null);

      try {
        const response = await fetch(`/api/v1/templates/${templateId}/preview`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': getTenantId(),
          },
          body: JSON.stringify({ parameters }),
        });

        if (!response.ok) {
          throw new Error(`Failed to generate preview (${response.status})`);
        }

        const data = await response.json();
        setPreview(data);
        return data;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to generate preview';
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [templateId]
  );

  return {
    preview,
    generatePreview,
    loading,
    error,
    clear: () => setPreview(null),
  };
};
