import * as React from 'react';
import { Stack, TextField, Autocomplete, CircularProgress } from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import { useEffect, useState } from 'react';

interface SearchResult {
  id: string;
  label: string;
  category: string;
  href: string;
}

export function GlobalSearch() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!query) {
      setResults([]);
      return;
    }

    setLoading(true);
    // This would call /api/search in a real implementation
    const timer = setTimeout(() => {
      // Mock results
      setResults([
        {
          id: '1',
          label: `Rule: ${query}`,
          category: 'Rules',
          href: `/console/compliance/rules/${query}`,
        },
        {
          id: '2',
          label: `Scenario: ${query}`,
          category: 'Scenarios',
          href: `/console/risk/scenarios/${query}`,
        },
      ]);
      setLoading(false);
    }, 200);

    return () => clearTimeout(timer);
  }, [query]);

  const grouped = results.reduce(
    (acc, result) => {
      const group = acc[result.category] || [];
      group.push(result);
      acc[result.category] = group;
      return acc;
    },
    {} as Record<string, SearchResult[]>
  );

  return React.createElement(
    Autocomplete as any,
    {
      open,
      onOpen: () => setOpen(true),
      onClose: () => setOpen(false),
      inputValue: query,
      onInputChange: (_: any, value: string) => setQuery(value),
      options: results,
      groupBy: (option: any) => option.category,
      getOptionLabel: (option: any) => option.label,
      renderInput: (params: any) => React.createElement(
        TextField,
        {
          ...params,
          placeholder: "Search rules, scenarios, portfolios...",
          size: "small",
          sx: { width: 300 },
          InputProps: {
            ...params.InputProps,
            startAdornment: React.createElement(React.Fragment, null,
              React.createElement(SearchIcon, { sx: { mr: 1, color: 'textSecondary' } })
            ),
            endAdornment: loading
              ? React.createElement(CircularProgress, { color: 'inherit', size: 20 })
              : params.InputProps.endAdornment,
          },
        }
      ),
      onOptionSelected: (_: any, option: any) => {
        window.location.href = option.href;
      },
      noOptionsText: query ? 'No results found' : 'Type to search',
    }
  );
}
