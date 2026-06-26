import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

type JsonRecord = Record<string, unknown>;

const API_PREFIX = '/api/v1';

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

const resolvePath = (path: string): string => {
  if (path.startsWith('http')) {
    return path;
  }
  // Read Vite env in a type-safe local narrow
  const __env = (import.meta as unknown as { env?: Record<string, unknown> })?.env;
  const base = (__env?.VITE_API_BASE_URL as string | undefined) || undefined;
  const cleanBase = base ? base.replace(/\/?$/, '') : '';
  const cleanPath = path.startsWith('/') ? path : `/${path}`;
  return `${cleanBase}${cleanPath}` || cleanPath;
};

const parseJson = async (response: Response) => {
  if (response.status === 204) {
    return undefined;
  }

  const text = await response.text();
  if (!text) {
    return undefined;
  }

  try {
    return JSON.parse(text);
  } catch (error) {
    throw new Error(`Failed to parse server response: ${(error as Error).message}`);
  }
};

async function request<T>(path: string, { method = 'GET', body, headers, ...rest }: RequestInit & { method?: HttpMethod } = {}): Promise<T> {
  const resolvedPath = resolvePath(path);
  const finalHeaders = new Headers(headers ?? undefined);

  if (body != null && !finalHeaders.has('Content-Type')) {
    finalHeaders.set('Content-Type', 'application/json');
  }
  if (!finalHeaders.has('Accept')) {
    finalHeaders.set('Accept', 'application/json');
  }

  const response = await fetch(resolvedPath, {
    method,
    credentials: 'include',
    body,
    headers: finalHeaders,
    ...rest,
  });

  if (!response.ok) {
    const payload = await response.text().catch(() => '');
    const message = payload || response.statusText || 'Unknown error';
    throw new Error(`[${response.status}] ${message}`);
  }

  return parseJson(response) as Promise<T>;
}

// -------------------------------------------------------------------------------------
// Data sources
// -------------------------------------------------------------------------------------

export interface ReportDataSource {
  id: string;
  name: string;
  type?: string;
  connectionString?: string;
  url?: string;
  description?: string;
  createdAt?: string;
  updatedAt?: string;
  [key: string]: unknown;
}

export interface CreateDataSourceInput {
  name: string;
  type?: string;
  connectionString?: string;
  url?: string;
  credentials?: JsonRecord;
  description?: string;
}

export interface UpdateDataSourceInput extends CreateDataSourceInput {
  id: string;
}

const toReportDataSource = (raw: JsonRecord | null | undefined): ReportDataSource | null => {
  if (!raw) {
    return null;
  }

  const id = (raw.id as string) ?? (raw.tenant_instance_id as string) ?? (raw.uuid as string);
  const name = (raw.name as string) ?? (raw.datasource_name as string) ?? (raw.title as string);

  if (!id || !name) {
    return null;
  }

  return {
    id,
    name,
    type: (raw.type as string) ?? (raw.datasource_type as string) ?? (raw.kind as string),
    connectionString: (raw.connectionString as string) ?? (raw.connection_string as string),
    url: (raw.url as string) ?? (raw.endpoint as string),
    description: (raw.description as string) ?? undefined,
    createdAt: (raw.created_at as string) ?? undefined,
    updatedAt: (raw.updated_at as string) ?? undefined,
    ...raw,
  };
};

const normaliseCollection = (raw: unknown): JsonRecord[] => {
  if (!raw) {
    return [];
  }
  if (Array.isArray(raw)) {
    return raw as JsonRecord[];
  }
  const record = raw as JsonRecord;
  for (const key of ['items', 'results', 'data', 'dataSources', 'data_sources']) {
    const value = record[key];
    if (Array.isArray(value)) {
      return value as JsonRecord[];
    }
  }
  return [];
};

const fetchDataSources = async (): Promise<ReportDataSource[]> => {
  const raw = await request<unknown>(`${API_PREFIX}/data-sources`);
  return normaliseCollection(raw)
    .map(toReportDataSource)
    .filter((item): item is ReportDataSource => item !== null);
};

export const useDataSources = () =>
  useQuery({
    queryKey: ['reporting', 'data-sources'],
    queryFn: fetchDataSources,
    staleTime: 60_000,
  });

export const useCreateDataSource = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateDataSourceInput) =>
      request<JsonRecord>(`${API_PREFIX}/data-sources`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'data-sources'] });
    },
  });
};

export const useUpdateDataSource = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...payload }: UpdateDataSourceInput) =>
      request<JsonRecord>(`${API_PREFIX}/data-sources/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'data-sources'] });
    },
  });
};

export const useDeleteDataSource = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) =>
      request<void>(`${API_PREFIX}/data-sources/${id}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'data-sources'] });
    },
  });
};

// -------------------------------------------------------------------------------------
// Datasets
// -------------------------------------------------------------------------------------

export interface ReportingDatasetField {
  name: string;
  type?: string;
  description?: string;
  [key: string]: unknown;
}

export interface ReportingDataset {
  id: string;
  name: string;
  dataSourceId?: string;
  fields: ReportingDatasetField[];
  description?: string;
  [key: string]: unknown;
}

const toReportingDataset = (raw: JsonRecord | null | undefined): ReportingDataset | null => {
  if (!raw) {
    return null;
  }
  const id = (raw.id as string) ?? (raw.dataset_id as string) ?? (raw.datasetId as string);
  const name = (raw.name as string) ?? (raw.dataset_name as string) ?? (raw.title as string);
  if (!id || !name) {
    return null;
  }
  const fieldsSource = Array.isArray(raw.fields)
    ? (raw.fields as JsonRecord[])
    : Array.isArray(raw.columns)
    ? (raw.columns as JsonRecord[])
    : [];
  const fields: ReportingDatasetField[] = fieldsSource.map((field) => ({
    name: (field.name as string) ?? (field.column_name as string) ?? 'Field',
    type: (field.type as string) ?? (field.data_type as string),
    description: (field.description as string) ?? undefined,
    ...field,
  }));

  return {
    id,
    name,
    dataSourceId: (raw.data_source_id as string) ?? (raw.dataSourceId as string),
    fields,
    description: (raw.description as string) ?? undefined,
    ...raw,
  };
};

const fetchDatasets = async (): Promise<ReportingDataset[]> => {
  const raw = await request<unknown>(`${API_PREFIX}/datasets`);
  return normaliseCollection(raw)
    .map(toReportingDataset)
    .filter((item): item is ReportingDataset => item !== null);
};

export const useDatasets = () =>
  useQuery({
    queryKey: ['reporting', 'datasets'],
    queryFn: fetchDatasets,
    staleTime: 60_000,
  });

// -------------------------------------------------------------------------------------
// Report templates
// -------------------------------------------------------------------------------------

export interface ReportTemplate {
  id: string;
  name: string;
  description?: string;
  definition?: JsonRecord | null;
  metadata?: JsonRecord | null;
  createdAt?: string;
  updatedAt?: string;
  [key: string]: unknown;
}

export interface SaveReportTemplateInput {
  name: string;
  description?: string;
  definition: JsonRecord;
  metadata?: JsonRecord;
}

export interface UpdateReportTemplateInput {
  id: string;
  payload: SaveReportTemplateInput;
}

const tryParseDefinition = (value: unknown): JsonRecord | null => {
  if (!value) {
    return null;
  }
  if (typeof value === 'string') {
    try {
      return JSON.parse(value) as JsonRecord;
    } catch {
      return null;
    }
  }
  if (typeof value === 'object') {
    return value as JsonRecord;
  }
  return null;
};

const toReportTemplate = (raw: JsonRecord | null | undefined): ReportTemplate | null => {
  if (!raw) {
    return null;
  }
  const id = (raw.id as string) ?? (raw.report_id as string) ?? (raw.uuid as string);
  const name = (raw.name as string) ?? (raw.title as string) ?? 'Untitled Report';
  if (!id) {
    return null;
  }

  const definition = tryParseDefinition(raw.definition ?? raw.template ?? raw.payload ?? raw.report_definition);
  const metadata = tryParseDefinition(raw.metadata ?? raw.meta);

  return {
    id,
    name,
    description: (raw.description as string) ?? undefined,
    definition,
    metadata,
    createdAt: (raw.created_at as string) ?? undefined,
    updatedAt: (raw.updated_at as string) ?? undefined,
    ...raw,
  };
};

const fetchReportTemplates = async (): Promise<ReportTemplate[]> => {
  const raw = await request<unknown>(`${API_PREFIX}/reports`);
  return normaliseCollection(raw)
    .map(toReportTemplate)
    .filter((item): item is ReportTemplate => item !== null);
};

export const useReportTemplates = () =>
  useQuery({
    queryKey: ['reporting', 'reports'],
    queryFn: fetchReportTemplates,
    staleTime: 30_000,
  });

export const useCreateReportTemplate = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: SaveReportTemplateInput) =>
      request<JsonRecord>(`${API_PREFIX}/reports`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'reports'] });
    },
  });
};

export const useUpdateReportTemplate = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, payload }: UpdateReportTemplateInput) =>
      request<JsonRecord>(`${API_PREFIX}/reports/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'reports'] });
      queryClient.invalidateQueries({ queryKey: ['reporting', 'reports', variables.id] });
    },
  });
};

export const useDeleteReportTemplate = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) =>
      request<void>(`${API_PREFIX}/reports/${id}`, {
        method: 'DELETE',
      }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'reports'] });
      queryClient.removeQueries({ queryKey: ['reporting', 'reports', id] });
    },
  });
};

// -------------------------------------------------------------------------------------
// Report preview & parameters
// -------------------------------------------------------------------------------------

export interface ReportPreviewParams {
  reportId?: string | null;
  datasetId?: string | null;
  dataSourceId?: string | null;
}

export interface ReportPreviewData {
  detailRows: JsonRecord[];
  matrixRows: JsonRecord[];
}

const buildQueryString = (params: ReportPreviewParams | undefined): string => {
  if (!params) {
    return '';
  }
  const search = new URLSearchParams();
  if (params.reportId) {
    search.set('report_id', params.reportId);
  }
  if (params.datasetId) {
    search.set('dataset_id', params.datasetId);
  }
  if (params.dataSourceId) {
    search.set('data_source_id', params.dataSourceId);
  }
  const qs = search.toString();
  return qs ? `?${qs}` : '';
};

const toPreviewData = (raw: unknown): ReportPreviewData => {
  const record = (raw ?? {}) as JsonRecord;
  const detailSource = Array.isArray(record.detailRows)
    ? (record.detailRows as JsonRecord[])
    : Array.isArray(record.detail_rows)
    ? (record.detail_rows as JsonRecord[])
    : Array.isArray(record.rows)
    ? (record.rows as JsonRecord[])
    : [];
  const matrixSource = Array.isArray(record.matrixRows)
    ? (record.matrixRows as JsonRecord[])
    : Array.isArray(record.matrix_rows)
    ? (record.matrix_rows as JsonRecord[])
    : Array.isArray(record.matrix)
    ? (record.matrix as JsonRecord[])
    : [];
  return {
    detailRows: detailSource,
    matrixRows: matrixSource,
  };
};

const fetchReportPreview = async (params?: ReportPreviewParams): Promise<ReportPreviewData> => {
  const raw = await request<unknown>(`${API_PREFIX}/reports/preview${buildQueryString(params)}`);
  return toPreviewData(raw);
};

export const useReportPreview = (params?: ReportPreviewParams) =>
  useQuery({
    queryKey: ['reporting', 'preview', params?.reportId ?? null, params?.datasetId ?? null, params?.dataSourceId ?? null],
    queryFn: () => fetchReportPreview(params),
  });

export interface ReportParameter {
  id?: string;
  name: string;
  type?: string;
  prompt?: string;
  defaultValue?: unknown;
  multiValue?: boolean;
  description?: string;
  [key: string]: unknown;
}

const toReportParameter = (raw: JsonRecord | null | undefined): ReportParameter | null => {
  if (!raw) {
    return null;
  }
  const name = (raw.name as string) ?? (raw.parameter_name as string);
  if (!name) {
    return null;
  }
  return {
    id: (raw.id as string) ?? (raw.parameter_id as string),
    name,
    type: (raw.type as string) ?? (raw.data_type as string),
    prompt: (raw.prompt as string) ?? (raw.label as string),
    defaultValue: raw.defaultValue ?? raw.default_value,
    multiValue: Boolean(raw.multiValue ?? raw.multi_value),
    description: (raw.description as string) ?? undefined,
    ...raw,
  };
};

const fetchReportParameters = async (): Promise<ReportParameter[]> => {
  const raw = await request<unknown>(`${API_PREFIX}/report-parameters`);
  return normaliseCollection(raw)
    .map(toReportParameter)
    .filter((item): item is ReportParameter => item !== null);
};

export const useReportParameters = () =>
  useQuery({
    queryKey: ['reporting', 'report-parameters'],
    queryFn: fetchReportParameters,
    staleTime: 30_000,
  });
