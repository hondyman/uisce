import React, { useState } from 'react';
import { Handle, Position } from 'reactflow';
import {
  Box,
  Typography,
  Chip,
  IconButton,
  Collapse,
  Stack,
  Divider,
} from '@mui/material';
import {
  BusinessCenter as BOIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Category as SubtypeIcon,
  FiberManualRecord as TermDotIcon,
  Storage as TableIcon,
} from '@mui/icons-material';

export const BONode: React.FC<{ data: any }> = ({ data }) => {
  const isRelated = data.relationshipType !== undefined;
  const [expanded, setExpanded] = useState<boolean>(data.isExpanded || false);

  // Group terms by subtype
  const terms: any[] = data.terms || [];
  const groupedTerms: Record<string, any[]> = {};

  terms.forEach((t) => {
    const groupName = t.subtypeName || 'Core Semantic Terms';
    if (!groupedTerms[groupName]) {
      groupedTerms[groupName] = [];
    }
    groupedTerms[groupName].push(t);
  });

  const hasTerms = terms.length > 0 || (data.termCount && data.termCount > 0);

  return (
    <Box
      sx={{
        padding: 2,
        border: '2px solid',
        borderColor: isRelated ? 'secondary.main' : 'primary.main',
        borderRadius: 2.5,
        background: isRelated
          ? 'linear-gradient(135deg, #ffffff 0%, #f3e5f5 100%)'
          : 'linear-gradient(135deg, #ffffff 0%, #e3f2fd 100%)',
        minWidth: 260,
        maxWidth: 360,
        boxShadow: 3,
        position: 'relative',
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: '#1976d2' }} />

      <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={1}>
        <Stack direction="row" alignItems="center" spacing={1.2}>
          <BOIcon color={isRelated ? 'secondary' : 'primary'} sx={{ fontSize: 24 }} />
          <Box>
            <Typography variant="subtitle2" fontWeight="bold" color={isRelated ? 'secondary.dark' : 'primary.dark'}>
              {data.name || data.label}
            </Typography>
            {data.driverTableName && (
              <Stack direction="row" alignItems="center" spacing={0.5}>
                <TableIcon sx={{ fontSize: 12, color: 'text.secondary' }} />
                <Typography variant="caption" color="text.secondary">
                  {data.driverTableName}
                </Typography>
              </Stack>
            )}
          </Box>
        </Stack>

        <Stack direction="row" alignItems="center" spacing={0.5}>
          {isRelated && (
            <Chip
              label={data.relationshipType}
              size="small"
              color="secondary"
              variant="outlined"
              sx={{ fontSize: '0.65rem', height: 20 }}
            />
          )}
          {hasTerms && (
            <IconButton
              size="small"
              onClick={() => {
                setExpanded(!expanded);
                data.onToggleExpand?.(!expanded);
              }}
              sx={{ p: 0.5 }}
              title={expanded ? 'Collapse Semantic Terms' : 'Expand Semantic Terms'}
            >
              {expanded ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
            </IconButton>
          )}
        </Stack>
      </Stack>

      {data.joinCondition && (
        <Box sx={{ mt: 1, p: 0.75, bgcolor: 'action.hover', borderRadius: 1, border: '1px dashed', borderColor: 'divider' }}>
          <Typography variant="caption" sx={{ fontFamily: 'monospace', fontSize: '0.7rem', color: 'primary.dark', display: 'block' }}>
            🔗 {data.joinCondition}
          </Typography>
        </Box>
      )}

      {data.description && (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.75 }}>
          {data.description.length > 75
            ? `${data.description.substring(0, 75)}...`
            : data.description}
        </Typography>
      )}

      <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
        <Chip
          label={`${terms.length || data.termCount || 0} terms`}
          size="small"
          variant="outlined"
          sx={{ height: 20, fontSize: '0.65rem' }}
        />
        {data.relatedBOCount > 0 && (
          <Chip
            label={`${data.relatedBOCount} related`}
            size="small"
            variant="outlined"
            sx={{ height: 20, fontSize: '0.65rem' }}
          />
        )}
      </Stack>

      {/* Expanded Semantic Terms Grouped By Subtype */}
      <Collapse in={expanded} timeout="auto" unmountOnExit>
        <Divider sx={{ my: 1.5 }} />
        <Box sx={{ maxHeight: 220, overflowY: 'auto', pr: 0.5 }}>
          {Object.entries(groupedTerms).map(([group, groupTerms]) => (
            <Box key={group} sx={{ mb: 1.5 }}>
              <Stack direction="row" alignItems="center" spacing={0.5} sx={{ mb: 0.5 }}>
                <SubtypeIcon sx={{ fontSize: 13, color: 'text.secondary' }} />
                <Typography variant="caption" fontWeight={700} color="text.primary">
                  {group} ({groupTerms.length})
                </Typography>
              </Stack>
              <Stack spacing={0.5} sx={{ pl: 1 }}>
                {groupTerms.map((term: any) => (
                  <Stack key={term.id || term.nodeName} direction="row" alignItems="center" spacing={0.8}>
                    <TermDotIcon sx={{ fontSize: 7, color: term.isKey ? 'error.main' : 'primary.main' }} />
                    <Typography variant="caption" sx={{ fontSize: '0.72rem', color: 'text.secondary' }}>
                      {term.nodeName || term.name}
                    </Typography>
                    {term.dataType && (
                      <Typography variant="caption" sx={{ fontSize: '0.65rem', color: 'text.disabled', fontFamily: 'monospace' }}>
                        ({term.dataType})
                      </Typography>
                    )}
                  </Stack>
                ))}
              </Stack>
            </Box>
          ))}
        </Box>
      </Collapse>

      <Handle type="source" position={Position.Bottom} style={{ background: '#1976d2' }} />
    </Box>
  );
};
