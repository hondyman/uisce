import React, { useState, useEffect } from 'react';
import {
  Autocomplete,
  TextField,
  Box,
  Typography,
  Chip,
  CircularProgress,
  Paper,
  Stack,
  Avatar,
} from '@mui/material';
import { Group as GroupIcon, Business as BusinessIcon } from '@mui/icons-material';

export interface LdapGroup {
  dn: string;
  displayName: string;
  description?: string;
  memberCount?: number;
  type?: 'team' | 'department' | 'role';
}

interface GroupSelectorProps {
  value: string | null;
  onChange: (groupDn: string | null) => void;
  label?: string;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  helperText?: string;
  error?: boolean;
  multiple?: boolean;
}

export const GroupSelector: React.FC<GroupSelectorProps> = ({
  value,
  onChange,
  label = 'Team / User Group',
  placeholder = 'Search for a team...',
  required = false,
  disabled = false,
  helperText,
  error = false,
  multiple = false,
}) => {
  const [groups, setGroups] = useState<LdapGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const [inputValue, setInputValue] = useState('');

  // Fetch LDAP groups from API
  useEffect(() => {
    const fetchGroups = async () => {
      setLoading(true);
      try {
        // TODO: Replace with actual LDAP API call
        const response = await fetch('/api/ldap/groups');
        if (response.ok) {
          const data = await response.json();
          setGroups(data);
        } else {
          // Fallback to mock data
          setGroups([
            {
              dn: 'cn=Finance Team,ou=groups,dc=company,dc=com',
              displayName: 'Finance Team',
              description: 'Financial operations and accounting',
              memberCount: 45,
              type: 'team',
            },
            {
              dn: 'cn=Sales Team,ou=groups,dc=company,dc=com',
              displayName: 'Sales Team',
              description: 'Sales and business development',
              memberCount: 32,
              type: 'team',
            },
            {
              dn: 'cn=Engineering,ou=groups,dc=company,dc=com',
              displayName: 'Engineering',
              description: 'Software engineering and development',
              memberCount: 78,
              type: 'department',
            },
            {
              dn: 'cn=Executives,ou=groups,dc=company,dc=com',
              displayName: 'Executives',
              description: 'Executive leadership team',
              memberCount: 12,
              type: 'role',
            },
            {
              dn: 'cn=Compliance,ou=groups,dc=company,dc=com',
              displayName: 'Compliance Team',
              description: 'Regulatory compliance and risk',
              memberCount: 18,
              type: 'team',
            },
            {
              dn: 'cn=Portfolio Managers,ou=groups,dc=company,dc=com',
              displayName: 'Portfolio Managers',
              description: 'Investment portfolio management',
              memberCount: 25,
              type: 'role',
            },
            {
              dn: 'cn=Client Services,ou=groups,dc=company,dc=com',
              displayName: 'Client Services',
              description: 'Customer support and relations',
              memberCount: 56,
              type: 'team',
            },
          ]);
        }
      } catch (error) {
        console.error('Failed to fetch LDAP groups:', error);
        setGroups([]);
      } finally {
        setLoading(false);
      }
    };

    void fetchGroups();
  }, []);

  const selectedGroup = groups.find((g) => g.dn === value) || null;

  const getTypeColor = (type?: string) => {
    switch (type) {
      case 'team':
        return 'primary';
      case 'department':
        return 'secondary';
      case 'role':
        return 'success';
      default:
        return 'default';
    }
  };

  const getTypeIcon = (type?: string) => {
    switch (type) {
      case 'department':
        return '🏢';
      case 'role':
        return '👤';
      default:
        return '👥';
    }
  };

  return (
    <Autocomplete
      value={selectedGroup}
      onChange={(_, newValue) => onChange(newValue?.dn || null)}
      inputValue={inputValue}
      onInputChange={(_, newInputValue) => setInputValue(newInputValue)}
      options={groups}
      groupBy={(option) => option.type?.toUpperCase() || 'OTHER'}
      getOptionLabel={(option) => option.displayName}
      loading={loading}
      disabled={disabled}
      renderInput={(params) => (
        <TextField
          {...params}
          label={label}
          placeholder={placeholder}
          required={required}
          error={error}
          helperText={helperText}
          InputProps={{
            ...params.InputProps,
            startAdornment: (
              <>
                <GroupIcon sx={{ mr: 1, color: 'action.active' }} />
                {params.InputProps.startAdornment}
              </>
            ),
            endAdornment: (
              <>
                {loading ? <CircularProgress color="inherit" size={20} /> : null}
                {params.InputProps.endAdornment}
              </>
            ),
          }}
        />
      )}
      renderOption={(props, option) => (
        <li {...props}>
          <Stack direction="row" spacing={2} alignItems="center" sx={{ width: '100%' }}>
            <Avatar sx={{ bgcolor: 'primary.light', width: 32, height: 32 }}>
              {getTypeIcon(option.type)}
            </Avatar>
            <Stack sx={{ flex: 1 }}>
              <Stack direction="row" alignItems="center" spacing={1}>
                <Typography variant="body2" sx={{ fontWeight: 500 }}>
                  {option.displayName}
                </Typography>
                {option.type && (
                  <Chip
                    label={option.type}
                    size="small"
                    color={getTypeColor(option.type) as any}
                    variant="outlined"
                  />
                )}
              </Stack>
              {option.description && (
                <Typography variant="caption" color="text.secondary">
                  {option.description}
                </Typography>
              )}
              {option.memberCount !== undefined && (
                <Typography variant="caption" color="text.secondary">
                  {option.memberCount} members
                </Typography>
              )}
            </Stack>
          </Stack>
        </li>
      )}
      PaperComponent={({ children }) => (
        <Paper elevation={8}>{children}</Paper>
      )}
    />
  );
};

export default GroupSelector;
