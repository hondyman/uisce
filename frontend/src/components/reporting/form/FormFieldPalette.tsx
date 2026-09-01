import React from 'react';
import {
  Box,
  Typography,
  Stack,
  Paper,
  Tooltip,
} from '@mui/material';
import {
  ShortText as LabelIcon,
  Draw as SignatureIcon,
  HorizontalRule as DividerIcon,
} from '@mui/icons-material';

export const FormFieldPalette: React.FC = () => {
  return (
    <Paper
      elevation={0}
      sx={{
        width: 280,
        borderRight: '1px solid #1E293B',
        bgcolor: '#071526',
        p: 2,
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      <Typography variant="caption" sx={{ fontWeight: 700, color: '#00D4FF', textTransform: 'uppercase' }}>
        Static Building Blocks
      </Typography>

      <Stack spacing={1}>
        <Tooltip title="Add a static text label or section header" placement="right">
          <Box
            draggable
            onDragStart={(e) => {
              e.dataTransfer.setData(
                'application/json',
                JSON.stringify({
                  type: 'PALETTE_STATIC',
                  elementType: 'STATIC_LABEL',
                  label: 'Section Label / Description',
                  defaultExpression: 'Static explanatory text or title',
                })
              );
            }}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1.5,
              p: 1,
              bgcolor: '#0B1E36',
              borderRadius: 1,
              border: '1px solid #1E293B',
              cursor: 'grab',
              '&:hover': { borderColor: '#00D4FF' },
            }}
          >
            <LabelIcon sx={{ fontSize: 16, color: '#38BDF8' }} />
            <Typography variant="body2" sx={{ fontSize: 12, color: '#E2E8F0' }}>
              Static Label / Text
            </Typography>
          </Box>
        </Tooltip>

        <Tooltip title="Add a formal signoff box with signature line and date field" placement="right">
          <Box
            draggable
            onDragStart={(e) => {
              e.dataTransfer.setData(
                'application/json',
                JSON.stringify({
                  type: 'PALETTE_STATIC',
                  elementType: 'SIGNATURE_BLOCK',
                  label: 'Authorized Signatory',
                  defaultExpression: '',
                })
              );
            }}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1.5,
              p: 1,
              bgcolor: '#0B1E36',
              borderRadius: 1,
              border: '1px solid #1E293B',
              cursor: 'grab',
              '&:hover': { borderColor: '#00D4FF' },
            }}
          >
            <SignatureIcon sx={{ fontSize: 16, color: '#A855F7' }} />
            <Typography variant="body2" sx={{ fontSize: 12, color: '#E2E8F0' }}>
              Signature Block
            </Typography>
          </Box>
        </Tooltip>

        <Tooltip title="Add a horizontal divider line to separate form sections" placement="right">
          <Box
            draggable
            onDragStart={(e) => {
              e.dataTransfer.setData(
                'application/json',
                JSON.stringify({
                  type: 'PALETTE_STATIC',
                  elementType: 'DIVIDER',
                  label: 'Section Divider',
                  defaultExpression: '',
                })
              );
            }}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1.5,
              p: 1,
              bgcolor: '#0B1E36',
              borderRadius: 1,
              border: '1px solid #1E293B',
              cursor: 'grab',
              '&:hover': { borderColor: '#00D4FF' },
            }}
          >
            <DividerIcon sx={{ fontSize: 16, color: '#64748B' }} />
            <Typography variant="body2" sx={{ fontSize: 12, color: '#E2E8F0' }}>
              Divider Line
            </Typography>
          </Box>
        </Tooltip>
      </Stack>
    </Paper>
  );
};
