import React, { useState, useMemo, useRef, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  TextField,
  FormControlLabel,
  Checkbox,
  Chip,
  Typography,
  Collapse,
  IconButton,
  Stack,
  Paper,
  Divider,
  Button,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  ErrorOutline as ErrorIcon,
  WarningAmber as WarningIcon,
  Info as InfoIcon,
  Add as AddIcon,
  Edit as EditIcon,
} from '@mui/icons-material';
import type { Entity } from '../../types/entity-schema';
import { categorizeValidationRules, type AnnotatedValidationRule } from '../../utils/validationRules';

// Lazy loading component wrapper
function LazyLoadWrapper({ children }: { children: React.ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
          observer.unobserve(entry.target);
        }
      },
      {
        rootMargin: '50px',
      }
    );

    if (ref.current) {
      observer.observe(ref.current);
    }

    return () => observer.disconnect();
  }, []);

  return (
    <div ref={ref}>
      {isVisible ? children : <Box sx={{ minHeight: 100 }} />}
    </div>
  );
}

interface ValidationsTabProps {
  entity: Entity;
  rules?: AnnotatedValidationRule[];
  onRulesUpdate?: (rules: AnnotatedValidationRule[]) => void;
  onCrossEntitySave?: (condition: any) => void;
  onAddRule?: () => void;
  onEditRule?: (rule: AnnotatedValidationRule) => void;
}

interface Severity {
  label: string;
  color: 'error' | 'warning' | 'info' | 'success';
  icon: React.ReactNode;
  bgColor: string;
}

const severityMap: Record<string, Severity> = {
  error: {
    label: 'Error',
    color: 'error',
    icon: <ErrorIcon sx={{ fontSize: 18 }} />,
    bgColor: '#ffebee',
  },
  warning: {
    label: 'Warning',
    color: 'warning',
    icon: <WarningIcon sx={{ fontSize: 18 }} />,
    bgColor: '#fff3e0',
  },
  info: {
    label: 'Info',
    color: 'info',
    icon: <InfoIcon sx={{ fontSize: 18 }} />,
    bgColor: '#e3f2fd',
  },
};

// Small runtime narrowers to avoid wide `as any` casts
function asRecord<T extends Record<string, unknown> = Record<string, unknown>>(v: unknown): T {
  return (v && typeof v === 'object') ? (v as T) : ({} as T);
}

interface RuleCardProps {
  rule: AnnotatedValidationRule;
  expanded: boolean;
  onToggle: () => void;
  onEdit?: (rule: AnnotatedValidationRule) => void;
}

function RuleCard({ rule, expanded, onToggle, onEdit }: RuleCardProps) {
  const severity = severityMap[rule.severity || 'info'] || severityMap.info;
  const ruleExtras = asRecord(rule);
  const altStatus = typeof ruleExtras.status === 'string' ? ruleExtras.status : undefined;

  return (
    <Card
      sx={{
        mb: 1.5,
        border: '1px solid',
        borderColor: 'divider',
        transition: 'all 0.2s',
        bgcolor: (theme) => theme.palette.mode === 'dark' ? '#1c2636' : '#ffffff',
        '&:hover': {
          boxShadow: 2,
          backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#232f42' : '#f8fafc',
        },
      }}
    >
      <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1.5,
            cursor: 'pointer',
            userSelect: 'none',
          }}
          onClick={onToggle}
        >
          <IconButton
            size="small"
            sx={{
              transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
              transition: 'transform 0.2s',
            }}
          >
            <ExpandMoreIcon />
          </IconButton>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ color: severity.color }}>{severity.icon}</Box>
            <Chip
              label={severity.label}
              size="small"
              color={severity.color}
              variant="outlined"
            />
          </Box>

          <Box sx={{ flex: 1 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              {rule.rule_name}
            </Typography>
            <Typography variant="caption" sx={{ color: 'text.secondary' }}>
              {rule.description}
            </Typography>
          </Box>

          {/* Status Badge */}
          {rule.is_active !== undefined && (
            <Chip
              label={rule.is_active ? 'Active' : 'Inactive'}
              size="small"
              color={rule.is_active ? 'success' : 'default'}
              variant="filled"
              sx={{ ml: 1 }}
            />
          )}
          {/* Alternative status display if is_active doesn't exist but status field does */}
          {rule.is_active === undefined && altStatus && (
            <Chip
              label={altStatus === 'active' ? 'Active' : 'Inactive'}
              size="small"
              color={altStatus === 'active' ? 'success' : 'default'}
              variant="filled"
              sx={{ ml: 1 }}
            />
          )}

          {onEdit && (
            <IconButton
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                onEdit(rule);
              }}
              sx={{ ml: 1, color: 'primary.main' }}
              title="Edit Rule"
            >
              <EditIcon fontSize="small" />
            </IconButton>
          )}
        </Box>

        <Collapse in={expanded} timeout="auto" unmountOnExit>
          <Divider sx={{ my: 1.5 }} />
          <Stack spacing={1.5} sx={{ mt: 1.5 }}>
            <Box>
              <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary' }}>
                RULE ID
              </Typography>
              <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                {rule.id}
              </Typography>
            </Box>

            {rule.condition_json && (
              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary' }}>
                  CONDITION
                </Typography>
                <Paper
                  sx={{
                    p: 2,
                    backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#111827' : '#f1f5f9',
                    fontFamily: 'monospace',
                    fontSize: '0.75rem',
                    overflow: 'auto',
                    borderRadius: 2,
                    border: '1px solid',
                    borderColor: 'divider',
                  }}
                >
                  {typeof rule.condition_json === 'string'
                    ? rule.condition_json
                    : JSON.stringify(rule.condition_json, null, 2)}
                </Paper>
              </Box>
            )}

            {rule.remediation && (
              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary' }}>
                  REMEDIATION
                </Typography>
                <Typography variant="body2">{rule.remediation}</Typography>
              </Box>
            )}
          </Stack>
        </Collapse>
      </CardContent>
    </Card>
  );
}

interface RuleCategoryProps {
  title: string;
  description: string;
  rules: AnnotatedValidationRule[];
  expandedRuleId: string | null;
  onExpandChange: (id: string | null) => void;
  onEdit?: (rule: AnnotatedValidationRule) => void;
}

function RuleCategory({
  title,
  description,
  rules,
  expandedRuleId,
  onExpandChange,
  onEdit,
}: RuleCategoryProps) {
  if (rules.length === 0) return null;

  const severityCount = {
    error: rules.filter((r) => r.severity === 'error').length,
    warning: rules.filter((r) => r.severity === 'warning').length,
    info: rules.filter((r) => r.severity === 'info').length,
  };

  const hasErrors = severityCount.error > 0;
  const hasWarnings = severityCount.warning > 0;

  return (
    <div className="bg-white dark:bg-[#151b23] border border-[#dbe0e6] dark:border-[#232f3e] rounded-2xl shadow-sm mb-6 overflow-hidden">
      <div className={`px-5 py-3 border-b border-[#f0f2f5] dark:border-[#232f3e] flex items-center justify-between ${
        hasErrors
          ? 'bg-red-50/50 dark:bg-red-900/10'
          : hasWarnings
            ? 'bg-amber-50/50 dark:bg-amber-900/10'
            : 'bg-[#f8fafc] dark:bg-[#1c2636]'
      }`}>
        <div className="flex items-center gap-3">
          <span className={`material-symbols-outlined ${
            hasErrors ? 'text-red-500' : hasWarnings ? 'text-amber-500' : 'text-[#60758a] dark:text-[#94a3b8]'
          }`}>
            {hasErrors ? 'error' : hasWarnings ? 'warning' : 'folder_open'}
          </span>
          <div>
            <h4 className="text-sm font-bold text-[#111418] dark:text-white leading-tight">{title}</h4>
            <p className="text-[11px] text-[#60758a] dark:text-[#94a3b8]">{description}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {severityCount.error > 0 && (
            <span className="px-2 py-0.5 rounded-full bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 text-[10px] font-bold">
              {severityCount.error} Errors
            </span>
          )}
          {severityCount.warning > 0 && (
            <span className="px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 text-[10px] font-bold">
              {severityCount.warning} Warnings
            </span>
          )}
        </div>
      </div>

      <div className="p-4 space-y-3">
        {rules.map((rule) => (
          <LazyLoadWrapper key={rule.id}>
            <RuleCard
              rule={rule}
              expanded={expandedRuleId === rule.id}
              onToggle={() => onExpandChange(expandedRuleId === rule.id ? null : rule.id)}
              onEdit={onEdit}
            />
          </LazyLoadWrapper>
        ))}
      </div>
    </div>
  );
}

export function ValidationsTab({
  rules = [],
  onRulesUpdate,
  onCrossEntitySave,
  onAddRule,
  onEditRule,
}: ValidationsTabProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [filtersOpen, setFiltersOpen] = useState(true);
  const [selectedSeverities, setSelectedSeverities] = useState<Set<string>>(new Set());
  const [selectedEntitySubtypes, setSelectedEntitySubtypes] = useState<Set<string>>(new Set());
  const [selectedStatuses, setSelectedStatuses] = useState<Set<string>>(new Set());
  const [selectedRuleTypes, setSelectedRuleTypes] = useState<Set<string>>(new Set());
  const [expandedRuleId, setExpandedRuleId] = useState<string | null>(null);

  const categorized = categorizeValidationRules(rules || []);

  const filterRulesBySearch = (rulesToFilter: AnnotatedValidationRule[]) => {
    return rulesToFilter.filter((rule) => {
      const term = searchTerm.toLowerCase();
      return (
        rule.rule_name.toLowerCase().includes(term) ||
        (rule.description && rule.description.toLowerCase().includes(term)) ||
        JSON.stringify(rule.condition_json).toLowerCase().includes(term)
      );
    });
  };

  const applyAllFilters = (rulesToFilter: AnnotatedValidationRule[]) => {
    return rulesToFilter.filter((rule) => {
      if (
        selectedSeverities.size === 0 &&
        selectedStatuses.size === 0 &&
        selectedRuleTypes.size === 0 &&
        selectedEntitySubtypes.size === 0
      ) {
        return filterRulesBySearch([rule]).length > 0;
      }

      const passesSearch = filterRulesBySearch([rule]).length > 0;
      if (!passesSearch) return false;

      if (selectedSeverities.size > 0 && !selectedSeverities.has(rule.severity || 'info')) {
        return false;
      }

      if (selectedStatuses.size > 0) {
        const ruleStatus = rule.is_active ? 'active' : 'inactive';
        if (!selectedStatuses.has(ruleStatus)) {
          return false;
        }
      }

      if (selectedRuleTypes.size > 0 && !selectedRuleTypes.has(rule.rule_type || '')) {
        return false;
      }

      if (selectedEntitySubtypes.size > 0) {
        const rrec = asRecord(rule);
        const subtype = typeof rrec.entity_subtype === 'string'
          ? rrec.entity_subtype
          : (typeof rrec.sub_entity_type === 'string' ? rrec.sub_entity_type : 'customer');
        if (!selectedEntitySubtypes.has(subtype)) {
          return false;
        }
      }

      return true;
    });
  };

  const filteredDirect = useMemo(
    () => applyAllFilters(categorized.direct),
    [
      categorized.direct,
      searchTerm,
      selectedSeverities,
      selectedStatuses,
      selectedRuleTypes,
      selectedEntitySubtypes,
    ]
  );

  const filteredGlobal = useMemo(
    () => applyAllFilters(categorized.global),
    [
      categorized.global,
      searchTerm,
      selectedSeverities,
      selectedStatuses,
      selectedRuleTypes,
      selectedEntitySubtypes,
    ]
  );

  if (!rules || rules.length === 0) {
    return (
      <Box sx={{ width: '100%' }}>
        {/* Premium Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 4 }}>
          <Box>
            <h2 className="text-2xl font-bold text-[#111418] dark:text-white flex items-center gap-2">
              <span className="material-symbols-outlined text-primary text-[28px]">verified_user</span>
              Validation Analytics & Rules
            </h2>
            <p className="text-[#60758a] dark:text-[#94a3b8] mt-1 text-sm max-w-xl">
              Monitor data quality and enforce business logic constraints. Rules are executed during ingestion and data processing.
            </p>
          </Box>
        </Box>

        <div className="py-20 px-6 text-center bg-[#f8fafc] dark:bg-[#1c2636] rounded-2xl border-2 border-dashed border-[#dbe0e6] dark:border-[#232f3e] flex flex-col items-center gap-4">
          <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
            <span className="material-symbols-outlined text-primary text-[32px]">rule</span>
          </div>
          <div>
            <h4 className="text-xl font-bold text-[#111418] dark:text-white">No validation rules defined</h4>
            <p className="text-[#60758a] dark:text-[#94a3b8] text-sm mt-2 max-w-sm mx-auto">
              You haven't added any validation rules for this entity yet. Create your first rule to start monitoring data quality.
            </p>
          </div>
          {onAddRule && (
            <button
              onClick={onAddRule}
              className="mt-4 flex items-center gap-2 px-8 py-3 bg-primary hover:bg-primary/90 text-white font-bold rounded-xl transition-all shadow-lg shadow-primary/20 hover:shadow-primary/40 active:scale-[0.98] group"
            >
              <span className="material-symbols-outlined text-[20px] group-hover:rotate-90 transition-transform duration-300">add</span>
              Create Your First Rule
            </button>
          )}
        </div>
      </Box>
    );
  }

  const toggleSeverity = (severity: string) => {
    const newSet = new Set(selectedSeverities);
    if (newSet.has(severity)) {
      newSet.delete(severity);
    } else {
      newSet.add(severity);
    }
    setSelectedSeverities(newSet);
  };

  const toggleStatus = (status: string) => {
    const newSet = new Set(selectedStatuses);
    if (newSet.has(status)) {
      newSet.delete(status);
    } else {
      newSet.add(status);
    }
    setSelectedStatuses(newSet);
  };

  const toggleRuleType = (type: string) => {
    const newSet = new Set(selectedRuleTypes);
    if (newSet.has(type)) {
      newSet.delete(type);
    } else {
      newSet.add(type);
    }
    setSelectedRuleTypes(newSet);
  };

  const toggleEntitySubtype = (subtype: string) => {
    const newSet = new Set(selectedEntitySubtypes);
    if (newSet.has(subtype)) {
      newSet.delete(subtype);
    } else {
      newSet.add(subtype);
    }
    setSelectedEntitySubtypes(newSet);
  };

  // Count rules by filters for display
  const severityCount = {
    error: rules.filter((r) => r.severity === 'error').length,
    warning: rules.filter((r) => r.severity === 'warning').length,
    info: rules.filter((r) => r.severity === 'info').length,
  };

  const statusCount = {
    active: rules.filter((r) => r.is_active === true).length,
    inactive: rules.filter((r) => r.is_active === false).length,
  };

  const ruleTypeCount = {
    field_format: rules.filter((r) => r.rule_type === 'field_format').length,
    business_logic: rules.filter((r) => r.rule_type === 'business_logic').length,
  };

  // Entity Subtype counts - accurately calculated from actual rules
  const entitySubtypeCount = {
    customer: rules.length,
    retail_customer: rules.filter((r) => {
      const rr = asRecord(r);
      return rr.entity_subtype === 'retail_customer' || rr.sub_entity_type === 'retail_customer';
    }).length,
    industry_customer: rules.filter((r) => {
      const rr = asRecord(r);
      return rr.entity_subtype === 'industry_customer' || rr.sub_entity_type === 'industry_customer';
    }).length,
    government_customer: rules.filter((r) => {
      const rr = asRecord(r);
      return rr.entity_subtype === 'government_customer' || rr.sub_entity_type === 'government_customer';
    }).length,
  };

  return (
    <Box 
      sx={{ 
        display: 'flex', 
        gap: 4, 
        p: 4, 
        minHeight: '100%',
        bgcolor: (theme) => theme.palette.mode === 'dark' ? '#0b1118' : '#f8fafc',
      }}
    >
      {/* Filter Sidebar */}
      {filtersOpen && (
        <Box
          sx={{
            width: { xs: '100%', sm: 280, lg: 320 },
            flexShrink: 0,
            display: { xs: filtersOpen ? 'block' : 'none', sm: 'block' },
          }}
        >
          <div className="bg-white dark:bg-[#151b23] border border-[#dbe0e6] dark:border-[#232f3e] rounded-2xl shadow-sm sticky top-4 overflow-hidden h-fit">
            <div className="px-5 py-4 border-b border-[#f0f2f5] dark:border-[#232f3e] flex justify-between items-center bg-[#f8fafc] dark:bg-[#1c2636]">
              <h3 className="text-sm font-bold text-[#111418] dark:text-white flex items-center gap-2">
                <span className="material-symbols-outlined text-primary text-[20px]">filter_alt</span>
                Refine Rules
              </h3>
              <button
                onClick={() => {
                  setSearchTerm('');
                  setSelectedEntitySubtypes(new Set());
                  setSelectedSeverities(new Set());
                  setSelectedStatuses(new Set());
                  setSelectedRuleTypes(new Set());
                  setExpandedRuleId(null);
                }}
                className="text-xs font-semibold text-primary hover:text-primary/80 transition-colors"
              >
                Clear All
              </button>
            </div>

            <div className="p-5 space-y-6 max-h-[calc(100vh-200px)] overflow-y-auto">
              {/* Entity Subtypes - Hierarchical */}
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1.5 }}>
                  Entity Subtypes
                </Typography>
                <Stack spacing={1}>
                  {/* Parent: Customer - NOT indented */}
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={selectedEntitySubtypes.has('customer')}
                        onChange={() => toggleEntitySubtype('customer')}
                        indeterminate={
                          ['retail_customer', 'industry_customer', 'government_customer'].some(
                            (st) => selectedEntitySubtypes.has(st)
                          ) &&
                          !['retail_customer', 'industry_customer', 'government_customer'].every(
                            (st) => selectedEntitySubtypes.has(st)
                          )
                        }
                        size="small"
                      />
                    }
                    label={
                      <Typography variant="body2" sx={{ fontWeight: 600 }}>
                        Customer <Typography component="span" sx={{ color: 'text.secondary' }}>({entitySubtypeCount.customer})</Typography>
                      </Typography>
                    }
                  />
                  
                  {/* Children: Subtypes - INDENTED */}
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, pl: 3 }}>
                    <FormControlLabel
                      control={
                        <Checkbox
                          checked={selectedEntitySubtypes.has('retail_customer')}
                          onChange={() => toggleEntitySubtype('retail_customer')}
                          size="small"
                        />
                      }
                      label={
                        <Typography variant="body2" sx={{ fontSize: '0.875rem', color: 'text.secondary' }}>
                          Retail Customer <Typography component="span" sx={{ color: 'text.secondary' }}>({entitySubtypeCount.retail_customer})</Typography>
                        </Typography>
                      }
                    />
                    <FormControlLabel
                      control={
                        <Checkbox
                          checked={selectedEntitySubtypes.has('industry_customer')}
                          onChange={() => toggleEntitySubtype('industry_customer')}
                          size="small"
                        />
                      }
                      label={
                        <Typography variant="body2" sx={{ fontSize: '0.875rem', color: 'text.secondary' }}>
                          Industry Customer <Typography component="span" sx={{ color: 'text.secondary' }}>({entitySubtypeCount.industry_customer})</Typography>
                        </Typography>
                      }
                    />
                    <FormControlLabel
                      control={
                        <Checkbox
                          checked={selectedEntitySubtypes.has('government_customer')}
                          onChange={() => toggleEntitySubtype('government_customer')}
                          size="small"
                        />
                      }
                      label={
                        <Typography variant="body2" sx={{ fontSize: '0.875rem', color: 'text.secondary' }}>
                          Government Customer <Typography component="span" sx={{ color: 'text.secondary' }}>({entitySubtypeCount.government_customer})</Typography>
                        </Typography>
                      }
                    />
                  </Box>
                </Stack>
              </Box>

              <Divider />

              {/* Severity */}
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1.5 }}>
                  Severity
                </Typography>
                <Stack spacing={1}>
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={selectedSeverities.has('error')}
                        onChange={() => toggleSeverity('error')}
                        size="small"
                      />
                    }
                    label={
                      <Typography variant="body2">
                        Error <Typography component="span" sx={{ color: 'text.secondary' }}>({severityCount.error})</Typography>
                      </Typography>
                    }
                  />
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={selectedSeverities.has('warning')}
                        onChange={() => toggleSeverity('warning')}
                        size="small"
                      />
                    }
                    label={
                      <Typography variant="body2">
                        Warning <Typography component="span" sx={{ color: 'text.secondary' }}>({severityCount.warning})</Typography>
                      </Typography>
                    }
                  />
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={selectedSeverities.has('info')}
                        onChange={() => toggleSeverity('info')}
                        size="small"
                      />
                    }
                    label={
                      <Typography variant="body2">
                        Info <Typography component="span" sx={{ color: 'text.secondary' }}>({severityCount.info})</Typography>
                      </Typography>
                    }
                  />
                </Stack>
              </Box>

              <Divider />

              {/* Status */}
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1.5 }}>
                  Status
                </Typography>
                <Stack spacing={1}>
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={selectedStatuses.has('active')}
                        onChange={() => toggleStatus('active')}
                        size="small"
                      />
                    }
                    label={
                      <Typography variant="body2">
                        Active <Typography component="span" sx={{ color: 'text.secondary' }}>({statusCount.active})</Typography>
                      </Typography>
                    }
                  />
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={selectedStatuses.has('inactive')}
                        onChange={() => toggleStatus('inactive')}
                        size="small"
                      />
                    }
                    label={
                      <Typography variant="body2">
                        Inactive <Typography component="span" sx={{ color: 'text.secondary' }}>({statusCount.inactive})</Typography>
                      </Typography>
                    }
                  />
                </Stack>
              </Box>

              <Divider />

              {/* Rule Type */}
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1.5 }}>
                  Rule Type
                </Typography>
                <Stack spacing={1}>
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={selectedRuleTypes.has('field_format')}
                        onChange={() => toggleRuleType('field_format')}
                        size="small"
                      />
                    }
                    label={
                      <Typography variant="body2">
                        Field Format <Typography component="span" sx={{ color: 'text.secondary' }}>({ruleTypeCount.field_format})</Typography>
                      </Typography>
                    }
                  />
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={selectedRuleTypes.has('business_logic')}
                        onChange={() => toggleRuleType('business_logic')}
                        size="small"
                      />
                    }
                    label={
                      <Typography variant="body2">
                        Business Logic <Typography component="span" sx={{ color: 'text.secondary' }}>({ruleTypeCount.business_logic})</Typography>
                      </Typography>
                    }
                  />
                </Stack>
              </Box>
            </div>
          </div>
        </Box>
      )}

      {/* Main Content */}
      <Box sx={{ flex: 1, minWidth: 0 }}>
        {/* Premium Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 4 }}>
          <Box>
            <h2 className="text-2xl font-bold text-[#111418] dark:text-white flex items-center gap-2">
              <span className="material-symbols-outlined text-primary text-[28px]">verified_user</span>
              Validation Analytics & Rules
            </h2>
            <p className="text-[#60758a] dark:text-[#94a3b8] mt-1 text-sm max-w-xl">
              Monitor data quality and enforce business logic constraints. Rules are executed during ingestion and data processing.
            </p>
          </Box>
          {onAddRule && (
            <button
              onClick={onAddRule}
              className="flex items-center gap-2 px-5 py-2.5 bg-primary hover:bg-primary/90 text-white font-bold rounded-xl transition-all shadow-lg shadow-primary/20 hover:shadow-primary/40 active:scale-[0.98] group"
            >
              <span className="material-symbols-outlined text-[20px] group-hover:rotate-90 transition-transform duration-300">add</span>
              Create New Rule
            </button>
          )}
        </Box>

        {/* Search and Filter Row */}
        <Box sx={{ display: 'flex', gap: 2, mb: 3, alignItems: 'center' }}>
          <div className="flex-1 relative flex items-center">
            <span className="material-symbols-outlined absolute left-3 text-[#60758a] dark:text-[#94a3b8] text-[20px] pointer-events-none">search</span>
            <input
              type="text"
              placeholder="Search rules by name, description, or condition..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-white dark:bg-[#1c2636] border border-[#dbe0e6] dark:border-[#232f3e] rounded-xl py-2 pl-10 pr-4 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all text-[#111418] dark:text-white placeholder-[#94a3b8]"
            />
          </div>

          <button
            onClick={() => setFiltersOpen(!filtersOpen)}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl border transition-all text-sm font-semibold ${
              filtersOpen 
                ? 'bg-primary/10 border-primary/30 text-primary shadow-inner' 
                : 'bg-white dark:bg-[#1c2636] border-[#dbe0e6] dark:border-[#232f3e] text-[#60758a] dark:text-[#94a3b8] hover:border-primary/50'
            }`}
          >
            <span className="material-symbols-outlined text-[20px]">{filtersOpen ? 'filter_list_off' : 'filter_list'}</span>
            Filters
          </button>
        </Box>

        {/* Direct Rules */}
        <Stack spacing={2}>
          {filteredDirect.length > 0 && (
            <RuleCategory
              title="Direct Assignment"
              description="Rules applied directly to this entity's fields"
              rules={filteredDirect}
              expandedRuleId={expandedRuleId}
              onExpandChange={setExpandedRuleId}
              onEdit={onEditRule}
            />
          )}

          {/* Global Rules */}
          {filteredGlobal.length > 0 && (
            <RuleCategory
              title="Global Rules"
              description="Rules that apply to all entities across the system"
              rules={filteredGlobal}
              expandedRuleId={expandedRuleId}
              onExpandChange={setExpandedRuleId}
              onEdit={onEditRule}
            />
          )}

          {/* No Results */}
          {filteredDirect.length === 0 && filteredGlobal.length === 0 && (
            <div className="py-12 px-6 text-center bg-[#f8fafc] dark:bg-[#1c2636] rounded-2xl border-2 border-dashed border-[#dbe0e6] dark:border-[#232f3e] flex flex-col items-center gap-3">
              <span className="material-symbols-outlined text-[#60758a] dark:text-[#94a3b8] text-[48px]">search_off</span>
              <div>
                <h4 className="text-lg font-bold text-[#111418] dark:text-white">No rules found</h4>
                <p className="text-[#60758a] dark:text-[#94a3b8] text-sm mt-1 max-w-xs mx-auto">
                  Try adjusting your filters or search terms to find what you're looking for.
                </p>
              </div>
              <button
                onClick={() => {
                  setSearchTerm('');
                  setSelectedEntitySubtypes(new Set());
                  setSelectedSeverities(new Set());
                  setSelectedStatuses(new Set());
                  setSelectedRuleTypes(new Set());
                }}
                className="mt-2 text-sm font-bold text-primary hover:text-primary/80 transition-colors"
              >
                Clear all filters
              </button>
            </div>
          )}
        </Stack>
      </Box>
    </Box>
  );
}

export default ValidationsTab;