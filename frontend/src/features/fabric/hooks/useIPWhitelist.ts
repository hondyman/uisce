import { useState, useCallback, useMemo } from 'react';
import { IPWhitelistEntry, Tenant, IPWhitelistFilters, ALL_TENANTS_ID } from '../types/ipWhitelist';

export interface ConflictInfo {
  ipAddress: string;
  currentOwner: string;
  requestedAssignments: string[];
}

export const useIPWhitelistAPI = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [conflictDetected, setConflictDetected] = useState<ConflictInfo | null>(null);

  const handleRequest = useCallback(async <T>(
    request: () => Promise<Response>,
    errorMessage: string = 'An error occurred'
  ): Promise<T | null> => {
    setLoading(true);
    setError(null);
    setConflictDetected(null);
    
    try {
      const response = await request();
      
      if (!response.ok) {
        const errorText = await response.text();
        let errorData;
        try {
          errorData = JSON.parse(errorText);
        } catch {
          throw new Error(`${response.status}: ${errorText || response.statusText}`);
        }
        
        if (errorData.conflict) {
          const conflictError = new Error(errorData.message || 'Conflict detected');
          (conflictError as any).conflict = errorData.conflict;
          setConflictDetected(errorData.conflict);
          throw conflictError;
        }
        
        throw new Error(errorData.message || errorText || response.statusText);
      }
      
      const data = await response.json();
      return data;
    } catch (err: any) {
      setError(err.message || errorMessage);
      if (err.conflict) {
        throw err; // Re-throw conflict errors for special handling
      }
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchTenants = useCallback(async (): Promise<Tenant[]> => {
  const backendBase = (import.meta.env.VITE_API_BASE_URL as string) || 'http://localhost:29080';
  const candidates = ['/api/tenants', `${backendBase}/api/tenants`, 'http://localhost:3000/api/tenants'];
    
    for (const url of candidates) {
      try {
        const response = await fetch(url);
        if (response.ok) {
          const data = await response.json();
          const raw = data.tenants || [];
          return raw.map((t: any) => ({
            id: t.id,
            displayName: t.display_name || t.displayName || t.name || t.id,
            name: t.name,
            tenant_code: t.tenant_code
          }));
        }
      } catch {
        continue;
      }
    }
    
    setError('Failed to fetch tenants from any endpoint');
    return [];
  }, []);

  const fetchAllIPWhitelist = useCallback(async (): Promise<IPWhitelistEntry[]> => {
    const data = await handleRequest<{ whitelist: any[] }>(
      () => fetch('/api/ip-whitelist'),
      'Failed to fetch IP whitelist'
    );
    
    if (data) {
      return data.whitelist.map((entry: any) => ({
        ...entry,
        isActive: true,
        tenantIds: Array.isArray(entry.tenantIds) ? entry.tenantIds : [],
      }));
    }
    
    return [];
  }, [handleRequest]);

  const fetchTenantIPWhitelist = useCallback(async (tenantId: string): Promise<IPWhitelistEntry[]> => {
    const data = await handleRequest<{ whitelist: IPWhitelistEntry[] }>(
      () => fetch(`/api/tenants/${tenantId}/ip-whitelist`),
      `Failed to fetch IP whitelist for tenant ${tenantId}`
    );
    
    return data?.whitelist || [];
  }, [handleRequest]);

  const addIPWhitelist = useCallback(async (
    tenantId: string,
    ipAddress: string,
    label?: string,
    description?: string,
    additionalTenantIds?: string[],
    opts?: { allTenants?: boolean }
  ): Promise<boolean> => {
    // If ALL_TENANTS, send allTenants: true and avoid assignments
  const isAll = opts?.allTenants === true || tenantId === ALL_TENANTS_ID || (additionalTenantIds || []).includes(ALL_TENANTS_ID);
    let primary = tenantId;
    let extras = additionalTenantIds || [];
    if (isAll) {
      // choose any non-ALL tenant as primary path segment; backend ignores assignments when allTenants is true
      const nonAll = [tenantId, ...extras].find(id => id && id !== ALL_TENANTS_ID);
      primary = nonAll || 'default';
      extras = [];
    } else if (primary === ALL_TENANTS_ID) {
      // fallback safety: if somehow primary is ALL, pick first extra
      primary = extras[0] || '';
      extras = extras.filter(id => id !== primary);
    }

  const payload: any = {
      ipAddress,
      label: label || null,
      description: description || null,
      tenantIds: (extras || []).filter(id => id && id !== ALL_TENANTS_ID)
    };
  if (isAll) payload.allTenants = true;

    const success = await handleRequest(
      () => fetch(`/api/tenants/${primary}/ip-whitelist`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      }),
      'Failed to add IP to whitelist'
    );

    return success !== null;
  }, [handleRequest]);

  const removeIPWhitelist = useCallback(async (
    tenantId: string,
    ipAddress: string,
    tenantIdsToRemove?: string[]
  ): Promise<boolean> => {
    const payload = {
      ipAddress,
      tenantIds: tenantIdsToRemove || []
    };

    const success = await handleRequest(
      () => fetch(`/api/tenants/${tenantId}/ip-whitelist`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      }),
      'Failed to remove IP from whitelist'
    );

    return success !== null;
  }, [handleRequest]);

  const updateIPAssignments = useCallback(async (
    ipAddress: string,
    currentTenantId: string,
    newTenantIds: string[],
    options?: { allTenants?: boolean; prevTenantIds?: string[] }
  ): Promise<boolean> => {
    try {
      // Remove from current or all previous tenants when converting assignments
      if (options?.prevTenantIds && options.prevTenantIds.length > 0) {
        // Use the first as path param, but send all for removal
        const primary = options.prevTenantIds[0];
        await removeIPWhitelist(primary, ipAddress, options.prevTenantIds);
      } else if (currentTenantId) {
        await removeIPWhitelist(currentTenantId, ipAddress);
      }
      
      // Add to new tenants or mark as All Tenants
  if (options?.allTenants || newTenantIds.includes(ALL_TENANTS_ID)) {
        await addIPWhitelist(ALL_TENANTS_ID, ipAddress, undefined, undefined, [], { allTenants: true });
      } else {
        const cleaned = newTenantIds.filter(id => id && id !== ALL_TENANTS_ID);
        if (cleaned.length > 0) {
          const primaryTenant = cleaned[0];
          const additionalTenants = cleaned.slice(1);
          await addIPWhitelist(primaryTenant, ipAddress, undefined, undefined, additionalTenants);
        }
      }
      
      return true;
    } catch (err) {
      return false;
    }
  }, [addIPWhitelist, removeIPWhitelist]);

  return {
    loading,
    error,
    setError,
    conflictDetected,
    setConflictDetected,
    fetchTenants,
    fetchAllIPWhitelist,
    fetchTenantIPWhitelist,
    addIPWhitelist,
    removeIPWhitelist,
    updateIPAssignments
  };
};

export const useIPAssignmentModal = () => {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedIP, setSelectedIP] = useState<IPWhitelistEntry | null>(null);
  const [assignedTenants, setAssignedTenants] = useState<string[]>([]);
  const [isGlobal, setIsGlobal] = useState(false);
  const [overrideConflict, setOverrideConflict] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(false);

  const openModal = useCallback((entry: IPWhitelistEntry) => {
    setSelectedIP(entry);
    setAssignedTenants(entry.tenantIds || []);
    setIsGlobal((entry as any).allTenants === true);
    setOverrideConflict(false);
    setSearchQuery('');
    setIsOpen(true);
  }, []);

  const closeModal = useCallback(() => {
    setIsOpen(false);
    setSelectedIP(null);
    setAssignedTenants([]);
    setIsGlobal(false);
    setOverrideConflict(false);
    setSearchQuery('');
  }, []);

  const addTenant = useCallback((tenantId: string) => {
    setAssignedTenants(prev => {
      if (prev.includes(tenantId)) return prev;
      return [...prev, tenantId];
    });
  }, []);

  const removeTenant = useCallback((tenantId: string) => {
    setAssignedTenants(prev => prev.filter(id => id !== tenantId));
  }, []);

  const toggleGlobalAssignment = useCallback((global: boolean) => {
    setIsGlobal(global);
    if (global) {
      setAssignedTenants([]);
    }
  }, []);

  return {
    // State
    isOpen,
    selectedIP,
    assignedTenants,
    isGlobal,
    overrideConflict,
    searchQuery,
    loading,

    // Actions
    openModal,
    closeModal,
    addTenant,
    removeTenant,
    toggleGlobalAssignment,
    setSearchQuery,
    setOverrideConflict,
    setLoading,
  };
};

export const useIPWhitelistTable = (entries: IPWhitelistEntry[], tenants: Tenant[]) => {
  const [loadedCount, setLoadedCount] = useState(10);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [tenantFilter, setTenantFilter] = useState('');
  const [sortBy, setSortBy] = useState<'ipAddress' | 'dateAdded' | 'status'>('dateAdded');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  const filteredAndSortedEntries = useMemo(() => {
    let filtered = [...entries];

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(entry =>
        entry.ipAddress.toLowerCase().includes(query) ||
        (entry.label && entry.label.toLowerCase().includes(query)) ||
        entry.tenantIds.some(id => {
          const tenant = tenants.find(t => t.id === id);
          return tenant?.displayName.toLowerCase().includes(query);
        })
      );
    }

    // Tenant filter
    if (tenantFilter && tenantFilter !== 'all') {
      const isAssigned = (e: IPWhitelistEntry) => (e as any).allTenants === true || e.tenantIds.length > 0;
      if (tenantFilter === 'assigned') {
        filtered = filtered.filter(isAssigned);
      } else if (tenantFilter === 'unassigned') {
        filtered = filtered.filter(e => !isAssigned(e));
      } else {
        filtered = filtered.filter(e =>
          (e as any).allTenants === true || e.tenantIds.includes(tenantFilter)
        );
      }
    }

    // Sort
    filtered.sort((a, b) => {
      let aVal: any, bVal: any;

      switch (sortBy) {
        case 'ipAddress':
          aVal = a.ipAddress;
          bVal = b.ipAddress;
          break;
        case 'dateAdded':
          aVal = new Date(a.createdAt || 0).getTime();
          bVal = new Date(b.createdAt || 0).getTime();
          break;
        case 'status':
          aVal = (a as any).allTenants ? 0 : a.tenantIds.length > 0 ? 1 : 2;
          bVal = (b as any).allTenants ? 0 : b.tenantIds.length > 0 ? 1 : 2;
          break;
        default:
          aVal = a.ipAddress;
          bVal = b.ipAddress;
      }

      if (typeof aVal === 'string') {
        aVal = aVal.toLowerCase();
        bVal = bVal.toLowerCase();
      }

      const comparison = aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
      return sortOrder === 'asc' ? comparison : -comparison;
    });

    return filtered;
  }, [entries, tenants, searchQuery, tenantFilter, sortBy, sortOrder]);

  const visibleEntries = useMemo(() => {
    return filteredAndSortedEntries.slice(0, loadedCount);
  }, [filteredAndSortedEntries, loadedCount]);

  const hasMore = useMemo(() => {
    return loadedCount < filteredAndSortedEntries.length;
  }, [loadedCount, filteredAndSortedEntries.length]);

  const loadMore = useCallback(() => {
    setIsLoadingMore(true);
    // Simulate loading delay for smoother UX
    setTimeout(() => {
      setLoadedCount(prev => prev + 10);
      setIsLoadingMore(false);
    }, 300);
  }, []);

  const resetLazyLoad = useCallback(() => {
    setLoadedCount(10);
  }, []);

  return {
    // State
    searchQuery,
    tenantFilter,
    sortBy,
    sortOrder,
    isLoadingMore,

    // Data
    visibleEntries,
    filteredAndSortedEntries,
    totalCount: filteredAndSortedEntries.length,
    hasMore,

    // Actions
    setSearchQuery,
    setTenantFilter,
    setSortBy,
    setSortOrder,
    loadMore,
    resetLazyLoad,
  };
};

export const useIPWhitelistFilters = (entries: IPWhitelistEntry[], tenants: Tenant[]) => {
  const [filters, setFilters] = useState<IPWhitelistFilters>({
    search: '',
    tenantFilter: '',
    statusFilter: 'all',
    sortBy: 'ipAddress',
    sortOrder: 'asc'
  });

  const filteredEntries = useMemo(() => {
    let filtered = [...entries];

    // Search filter
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      filtered = filtered.filter(entry => 
        entry.ipAddress.toLowerCase().includes(searchLower) ||
        (entry.label && entry.label.toLowerCase().includes(searchLower)) ||
        (entry.description && entry.description.toLowerCase().includes(searchLower)) ||
        entry.tenantIds.some(id => {
          const tenant = tenants.find(t => t.id === id);
          return tenant?.displayName.toLowerCase().includes(searchLower);
        })
      );
    }

    // Tenant filter: support special values: 'all' | 'assigned' | 'unassigned' | tenantId
    if (filters.tenantFilter && filters.tenantFilter !== 'all') {
      const isAssigned = (e: any) => (e.allTenants === true) || (e.tenantIds && e.tenantIds.length > 0);
      if (filters.tenantFilter === 'assigned') {
        filtered = filtered.filter(e => isAssigned(e));
      } else if (filters.tenantFilter === 'unassigned') {
        filtered = filtered.filter(e => !isAssigned(e));
      } else {
        // specific tenant id: include allTenants entries too
        filtered = filtered.filter(e => (e as any).allTenants === true || e.tenantIds.includes(filters.tenantFilter));
      }
    }

    // Status filter
    if (filters.statusFilter !== 'all') {
      // Backend currently doesn't track active/inactive; default all to active
      filtered = filters.statusFilter === 'active' ? filtered : [];
    }

    // Sort
    filtered.sort((a, b) => {
      let aValue: any, bValue: any;
      
      switch (filters.sortBy) {
        case 'ipAddress':
          aValue = a.ipAddress;
          bValue = b.ipAddress;
          break;
        case 'label':
          aValue = a.label || '';
          bValue = b.label || '';
          break;
        case 'tenantCount':
          aValue = a.tenantIds.length;
          bValue = b.tenantIds.length;
          break;
        case 'createdAt':
          aValue = a.createdAt || '';
          bValue = b.createdAt || '';
          break;
        default:
          aValue = a.ipAddress;
          bValue = b.ipAddress;
      }

      if (typeof aValue === 'string') {
        aValue = aValue.toLowerCase();
        bValue = bValue.toLowerCase();
      }

      const comparison = aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
      return filters.sortOrder === 'asc' ? comparison : -comparison;
    });

    return filtered;
  }, [entries, tenants, filters]);

  return {
    filters,
    setFilters,
    filteredEntries
  };
};

