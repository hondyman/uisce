import React, { useMemo } from 'react';
import {
  Box,
  Button,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Typography,
  Divider,
  Stack,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { CoreIcon, CustomIcon } from '../../../components/common/CoreCustomIcons';
import type { AttributeDef } from '../types';

interface FieldListProps {
  fields: AttributeDef[];
  selectedId?: string | null;
  onSelect: (field: AttributeDef) => void;
  onAdd: () => void;
}

function isCoreOrigin(field: AttributeDef): boolean {
  return (field.origin || '').toUpperCase() === 'CORE';
}

export function FieldList({ fields, selectedId, onSelect, onAdd }: FieldListProps) {
  const grouped = useMemo(() => {
    const map = new Map<string, AttributeDef[]>();
    for (const f of fields) {
      const section = f.section || 'Custom';
      if (!map.has(section)) map.set(section, []);
      map.get(section)!.push(f);
    }
    return Array.from(map.entries());
  }, [fields]);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Box sx={{ px: 2, py: 1.5 }}>
        <Typography variant="subtitle2" color="text.secondary">
          CUSTOM FIELDS
        </Typography>
        <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
          <Typography variant="caption" color="text.secondary">
            {fields.length} definition{fields.length === 1 ? '' : 's'}
          </Typography>
          <CustomIcon fontSize="small" />
          <Typography variant="caption" color="text.secondary">
            = tenant custom
          </Typography>
          <CoreIcon fontSize="small" />
          <Typography variant="caption" color="text.secondary">
            = core
          </Typography>
        </Stack>
      </Box>
      <Divider />
      <Box sx={{ flex: 1, overflow: 'auto' }}>
        {grouped.map(([section, items]) => (
          <Box key={section} sx={{ mb: 1 }}>
            <Typography
              variant="caption"
              sx={{ px: 2, pt: 1.5, display: 'block', fontWeight: 600, color: 'text.secondary' }}
            >
              {section}
            </Typography>
            <List dense disablePadding>
              {items.map((field) => {
                const core = isCoreOrigin(field);
                return (
                  <ListItemButton
                    key={field.id}
                    selected={selectedId === field.id}
                    onClick={() => onSelect(field)}
                    sx={{ px: 2 }}
                  >
                    <ListItemIcon sx={{ minWidth: 32 }}>
                      {core ? <CoreIcon fontSize="small" /> : <CustomIcon fontSize="small" />}
                    </ListItemIcon>
                    <ListItemText
                      primary={field.name}
                      secondary={
                        field.semantic_term_id
                          ? `${field.field_cd} · linked to semantic term`
                          : field.field_cd
                      }
                      primaryTypographyProps={{ variant: 'body2' }}
                      secondaryTypographyProps={{ variant: 'caption' }}
                    />
                  </ListItemButton>
                );
              })}
            </List>
          </Box>
        ))}
        {fields.length === 0 && (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            No custom fields yet. Add a field to get started.
          </Typography>
        )}
      </Box>
      <Divider />
      <Box sx={{ p: 1.5 }}>
        <Button fullWidth startIcon={<AddIcon />} variant="contained" onClick={onAdd}>
          Add Field
        </Button>
      </Box>
    </Box>
  );
}
