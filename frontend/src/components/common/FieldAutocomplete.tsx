import React, { useState, useEffect, useRef, useMemo } from 'react';
import { devError } from '../../utils/devLogger';
import {
  TextField,
  Box,
  // CircularProgress unused — removed to silence lint
  Paper,
  Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';

/**
 * Field interface representing metadata about a data field
 */
export interface Field {
  name: string;
  type: string;
  description?: string;
  nullable?: boolean;
  relatedEntity?: string;
}

/**
 * Entity schemas mapping entity names to their fields
 * This should be imported or provided from your application's schema definitions
 */
export const ENTITY_SCHEMAS: Record<string, Field[]> = {
  // Example schema - replace with actual entity schemas
  Employee: [
    {
      name: 'employee_id',
      type: 'uuid',
      description: 'Unique employee identifier',
      nullable: false,
    },
    {
      name: 'first_name',
      type: 'text',
      description: 'Employee first name',
      nullable: false,
    },
    {
      name: 'last_name',
      type: 'text',
      description: 'Employee last name',
      nullable: false,
    },
    {
      name: 'email',
      type: 'text',
      description: 'Employee email address',
      nullable: true,
    },
    {
      name: 'department_id',
      type: 'uuid',
      description: 'References Department entity',
      nullable: false,
      relatedEntity: 'Department',
    },
  ],
  Department: [
    {
      name: 'department_id',
      type: 'uuid',
      description: 'Unique department identifier',
      nullable: false,
    },
    {
      name: 'name',
      type: 'text',
      description: 'Department name',
      nullable: false,
    },
    {
      name: 'manager_id',
      type: 'uuid',
      description: 'References Employee entity',
      nullable: true,
      relatedEntity: 'Employee',
    },
  ],
};

interface FieldAutocompleteProps {
  value: string;
  onChange: (value: string) => void;
  entityName: string;
  placeholder?: string;
  error?: string;
  label?: string;
  required?: boolean;
  showRecentFields?: boolean;
  disabled?: boolean;
}

/**
 * FieldAutocomplete Component
 *
 * A context-aware autocomplete field selector with:
 * - Real-time search across field names and descriptions
 * - Recently used field memory
 * - Full keyboard navigation support (Arrow keys, Enter, Escape)
 * - Rich field information display (type, nullability, relationships)
 * - Mouse and keyboard highlight synchronization
 */
const FieldAutocomplete: React.FC<FieldAutocompleteProps> = ({
  value,
  onChange,
  entityName,
  placeholder = 'Search for a field...',
  error,
  label,
  required = false,
  showRecentFields = true,
  disabled = false,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState(value);
  const [recentFields, setRecentFields] = useState<string[]>([]);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);

  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const fields = ENTITY_SCHEMAS[entityName] || [];

  // Initialize search value when value prop changes
  useEffect(() => {
    setSearch(value);
  }, [value]);

  // Load recently used fields from sessionStorage
  useEffect(() => {
    const stored = sessionStorage.getItem(`recent_fields_${entityName}`);
    if (stored) {
      try {
        setRecentFields(JSON.parse(stored));
      } catch (e) {
        devError('Failed to parse recent fields:', e);
      }
    }
  }, [entityName]);

  // Handle click outside dropdown
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Filter recent fields - only show those that exist in the current entity schema
  const recentFilteredFields = useMemo(() => {
    return showRecentFields
      ? fields.filter((f) => recentFields.includes(f.name))
      : [];
  }, [fields, recentFields, showRecentFields]);

  // Filter all fields based on search query
  const filteredFields = useMemo(() => {
    return fields.filter(
      (field) =>
        (field.name.toLowerCase().includes(search.toLowerCase()) ||
          (field.description &&
            field.description.toLowerCase().includes(search.toLowerCase()))) &&
        !recentFilteredFields.some((rf) => rf.name === field.name) // Avoid duplication
    );
  }, [fields, search, recentFilteredFields]);

  // Combine recent and filtered fields for keyboard navigation
  const combinedFields = useMemo(
    () => [...recentFilteredFields, ...filteredFields],
    [recentFilteredFields, filteredFields]
  );

  // Auto-scroll highlighted item into view
  useEffect(() => {
    if (highlightedIndex >= 0 && dropdownRef.current) {
      const highlightedElement = dropdownRef.current.children[
        highlightedIndex
      ] as HTMLElement;
      highlightedElement?.scrollIntoView({ block: 'nearest' });
    }
  }, [highlightedIndex]);

  // Handle field selection
  const handleSelect = (fieldName: string) => {
    onChange(fieldName);
    setSearch(fieldName);
    setIsOpen(false);
    setHighlightedIndex(-1);

    // Update recently used fields
    const updated = [
      fieldName,
      ...recentFields.filter((f) => f !== fieldName),
    ].slice(0, 5);
    setRecentFields(updated);
    sessionStorage.setItem(`recent_fields_${entityName}`, JSON.stringify(updated));
  };

  // Handle keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!isOpen && e.key !== 'ArrowDown') return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setIsOpen(true);
        setHighlightedIndex((prev) =>
          prev < combinedFields.length - 1 ? prev + 1 : 0
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setHighlightedIndex((prev) =>
          prev > 0 ? prev - 1 : combinedFields.length - 1
        );
        break;
      case 'Enter':
        e.preventDefault();
        if (highlightedIndex >= 0 && combinedFields[highlightedIndex]) {
          handleSelect(combinedFields[highlightedIndex].name);
        }
        break;
      case 'Escape':
        e.preventDefault();
        setIsOpen(false);
        setHighlightedIndex(-1);
        break;
      default:
        break;
    }
  };

  // Get icon for field type
  const getFieldTypeIcon = (type: string): string => {
    const typeMap: Record<string, string> = {
      uuid: '🔑',
      text: '📝',
      varchar: '📝',
      integer: '#️⃣',
      int: '#️⃣',
      bigint: '📊',
      decimal: '💰',
      numeric: '💰',
      boolean: '✓',
      bool: '✓',
      timestamp: '⏰',
      date: '📅',
      time: '🕐',
      json: '{}',
      jsonb: '{}',
      array: '[]',
    };
    return typeMap[type.toLowerCase()] || '📦';
  };

  // Get badge styling for field type
  const getFieldTypeBadge = (type: string): string => {
    const typeMap: Record<string, string> = {
      uuid: 'bg-purple-100 text-purple-800',
      text: 'bg-blue-100 text-blue-800',
      varchar: 'bg-blue-100 text-blue-800',
      integer: 'bg-green-100 text-green-800',
      int: 'bg-green-100 text-green-800',
      bigint: 'bg-green-100 text-green-800',
      decimal: 'bg-amber-100 text-amber-800',
      numeric: 'bg-amber-100 text-amber-800',
      boolean: 'bg-red-100 text-red-800',
      bool: 'bg-red-100 text-red-800',
      timestamp: 'bg-indigo-100 text-indigo-800',
      date: 'bg-indigo-100 text-indigo-800',
      time: 'bg-indigo-100 text-indigo-800',
      json: 'bg-slate-100 text-slate-800',
      jsonb: 'bg-slate-100 text-slate-800',
      array: 'bg-slate-100 text-slate-800',
    };
    return typeMap[type.toLowerCase()] || 'bg-gray-100 text-gray-800';
  };

  // Render individual field item
  const renderFieldItem = (field: Field, index: number) => (
    <button
      key={field.name}
      onClick={() => handleSelect(field.name)}
      onMouseMove={() => setHighlightedIndex(index)}
      className={`w-full px-4 py-3 text-left flex items-start gap-3 border-b border-gray-100 last:border-b-0 transition-colors ${
        highlightedIndex === index ? 'bg-blue-50' : 'hover:bg-gray-50'
      }`}
      type="button"
    >
      <span className="text-lg flex-shrink-0 mt-0.5">
        {getFieldTypeIcon(field.type)}
      </span>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1 flex-wrap">
          <span className="font-medium text-gray-900">{field.name}</span>
          <span
            className={`px-2 py-0.5 text-xs rounded font-medium ${getFieldTypeBadge(
              field.type
            )}`}
          >
            {field.type}
          </span>
          {field.nullable && (
            <span className="px-2 py-0.5 text-xs rounded font-medium bg-gray-100 text-gray-600">
              nullable
            </span>
          )}
        </div>
        {field.description && (
          <div className="text-xs text-gray-500">{field.description}</div>
        )}
        {field.relatedEntity && (
          <div className="text-xs text-orange-600 mt-1">
            → References {field.relatedEntity}
          </div>
        )}
      </div>
    </button>
  );

  return (
    <div className="relative">
      {label && (
        <label className="block text-sm font-semibold text-gray-700 mb-2">
          {label} {required && <span className="text-red-500">*</span>}
        </label>
      )}
      <div className="relative">
        <TextField
          ref={inputRef}
          type="text"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setIsOpen(true);
            setHighlightedIndex(-1);
          }}
          onFocus={() => setIsOpen(true)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          placeholder={placeholder}
          error={!!error}
          fullWidth
          size="small"
          InputProps={{
            endAdornment: (
              <SearchIcon
                sx={{ color: 'action.active', mr: 1 }}
                fontSize="small"
              />
            ),
          }}
        />
      </div>

      {/* Dropdown menu */}
      {isOpen && !disabled && (
        <Paper
          ref={dropdownRef}
          sx={{
            position: 'absolute',
            top: '100%',
            left: 0,
            right: 0,
            zIndex: 1300,
            mt: 1,
            maxHeight: 320,
            overflowY: 'auto',
            boxShadow: 1,
          }}
          onMouseLeave={() => setHighlightedIndex(-1)}
        >
          {/* Recently used section */}
          {recentFilteredFields.length > 0 && search === value && (
            <>
              <Box sx={{ px: 2, py: 1.5, bgcolor: '#f5f5f5' }}>
                <Typography
                  variant="caption"
                  sx={{ fontWeight: 600, color: '#666' }}
                >
                  RECENTLY USED
                </Typography>
              </Box>
              {recentFilteredFields.map((field, index) =>
                renderFieldItem(field, index)
              )}
            </>
          )}

          {/* All fields section */}
          {filteredFields.length > 0 && (
            <>
              <Box sx={{ px: 2, py: 1.5, bgcolor: '#f5f5f5' }}>
                <Typography
                  variant="caption"
                  sx={{ fontWeight: 600, color: '#666' }}
                >
                  ALL FIELDS ({filteredFields.length})
                </Typography>
              </Box>
              {filteredFields.map((field, index) =>
                renderFieldItem(field, recentFilteredFields.length + index)
              )}
            </>
          )}

          {/* Empty state */}
          {combinedFields.length === 0 && (
            <Box sx={{ px: 4, py: 8, textAlign: 'center', color: '#999' }}>
              <SearchIcon sx={{ fontSize: 32, mb: 2, opacity: 0.5 }} />
              <Typography variant="body2">
                No fields found matching &quot;{search}&quot;
              </Typography>
              <Typography variant="caption" sx={{ mt: 1, display: 'block' }}>
                Try a different search term
              </Typography>
            </Box>
          )}
        </Paper>
      )}

      {/* Error message */}
      {error && (
        <Typography color="error" variant="caption" sx={{ mt: 0.5, display: 'block' }}>
          {error}
        </Typography>
      )}
    </div>
  );
};

export default FieldAutocomplete;
