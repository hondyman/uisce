import React from 'react';
import {
  Box,
  Select,
  MenuItem,
  ToggleButtonGroup,
  ToggleButton,
  IconButton,
  Tooltip,
  Divider,
  Paper
} from '@mui/material';
import FormatBoldIcon from '@mui/icons-material/FormatBold';
import FormatItalicIcon from '@mui/icons-material/FormatItalic';
import FormatUnderlinedIcon from '@mui/icons-material/FormatUnderlined';
import StrikethroughSIcon from '@mui/icons-material/StrikethroughS';
import FormatAlignLeftIcon from '@mui/icons-material/FormatAlignLeft';
import FormatAlignCenterIcon from '@mui/icons-material/FormatAlignCenter';
import FormatAlignRightIcon from '@mui/icons-material/FormatAlignRight';
import FormatAlignJustifyIcon from '@mui/icons-material/FormatAlignJustify';
import FormatColorTextIcon from '@mui/icons-material/FormatColorText';
import FormatColorFillIcon from '@mui/icons-material/FormatColorFill';

export const FONT_FAMILIES = [
  { label: 'Calibri', value: 'Calibri, sans-serif' },
  { label: 'Arial', value: 'Arial, sans-serif' },
  { label: 'Segoe UI', value: '"Segoe UI", sans-serif' },
  { label: 'Roboto', value: 'Roboto, sans-serif' },
  { label: 'Times New Roman', value: '"Times New Roman", serif' },
  { label: 'Georgia', value: 'Georgia, serif' },
  { label: 'Courier New', value: '"Courier New", monospace' },
  { label: 'Verdana', value: 'Verdana, sans-serif' },
  { label: 'Trebuchet MS', value: '"Trebuchet MS", sans-serif' },
  { label: 'System Default', value: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif' }
];

export const FONT_SIZES = [8, 9, 10, 11, 12, 14, 16, 18, 20, 24, 28, 32, 36, 48, 72];

interface FormatToolbarProps {
  properties: Record<string, any>;
  onUpdate: (property: string, value: any) => void;
  compact?: boolean;
}

export const FormatToolbar: React.FC<FormatToolbarProps> = ({
  properties,
  onUpdate,
  compact = false,
}) => {
  const formats = [
    properties.bold ? 'bold' : null,
    properties.italic ? 'italic' : null,
    properties.underline ? 'underline' : null,
    properties.strikethrough ? 'strikethrough' : null,
  ].filter(Boolean) as string[];

  const handleFormatChange = (_: any, newFormats: string[]) => {
    onUpdate('bold', newFormats.includes('bold'));
    onUpdate('italic', newFormats.includes('italic'));
    onUpdate('underline', newFormats.includes('underline'));
    onUpdate('strikethrough', newFormats.includes('strikethrough'));
  };

  return (
    <Paper
      elevation={0}
      sx={{
        display: 'flex',
        alignItems: 'center',
        flexWrap: 'wrap',
        gap: 0.75,
        p: 0.5,
        bgcolor: 'rgba(255, 255, 255, 0.04)',
        border: '1px solid rgba(255, 255, 255, 0.08)',
        borderRadius: 1.5,
      }}
    >
      {/* Font Family */}
      <Select
        size="small"
        value={properties.fontFamily || FONT_FAMILIES[0].value}
        onChange={(e) => onUpdate('fontFamily', e.target.value)}
        sx={{
          height: 28,
          fontSize: '0.75rem',
          minWidth: compact ? 100 : 130,
          '& .MuiSelect-select': { py: 0.5 }
        }}
      >
        {FONT_FAMILIES.map((font) => (
          <MenuItem key={font.value} value={font.value} sx={{ fontSize: '0.8rem', fontFamily: font.value }}>
            {font.label}
          </MenuItem>
        ))}
      </Select>

      {/* Font Size */}
      <Select
        size="small"
        value={properties.fontSize || 12}
        onChange={(e) => onUpdate('fontSize', Number(e.target.value))}
        sx={{
          height: 28,
          fontSize: '0.75rem',
          width: 65,
          '& .MuiSelect-select': { py: 0.5 }
        }}
      >
        {FONT_SIZES.map((size) => (
          <MenuItem key={size} value={size} sx={{ fontSize: '0.8rem' }}>
            {size}pt
          </MenuItem>
        ))}
      </Select>

      <Divider orientation="vertical" flexItem sx={{ my: 0.5 }} />

      {/* Bold, Italic, Underline, Strikethrough */}
      <ToggleButtonGroup
        size="small"
        value={formats}
        onChange={handleFormatChange}
        aria-label="text formatting"
        sx={{ height: 28 }}
      >
        <ToggleButton value="bold" aria-label="bold" sx={{ px: 0.75 }}>
          <Tooltip title="Bold (Ctrl+B)">
            <FormatBoldIcon sx={{ fontSize: 16 }} />
          </Tooltip>
        </ToggleButton>
        <ToggleButton value="italic" aria-label="italic" sx={{ px: 0.75 }}>
          <Tooltip title="Italic (Ctrl+I)">
            <FormatItalicIcon sx={{ fontSize: 16 }} />
          </Tooltip>
        </ToggleButton>
        <ToggleButton value="underline" aria-label="underline" sx={{ px: 0.75 }}>
          <Tooltip title="Underline (Ctrl+U)">
            <FormatUnderlinedIcon sx={{ fontSize: 16 }} />
          </Tooltip>
        </ToggleButton>
        <ToggleButton value="strikethrough" aria-label="strikethrough" sx={{ px: 0.75 }}>
          <Tooltip title="Strikethrough">
            <StrikethroughSIcon sx={{ fontSize: 16 }} />
          </Tooltip>
        </ToggleButton>
      </ToggleButtonGroup>

      <Divider orientation="vertical" flexItem sx={{ my: 0.5 }} />

      {/* Text Align */}
      <ToggleButtonGroup
        size="small"
        exclusive
        value={properties.textAlign || 'left'}
        onChange={(_, val) => val && onUpdate('textAlign', val)}
        aria-label="text alignment"
        sx={{ height: 28 }}
      >
        <ToggleButton value="left" aria-label="left aligned" sx={{ px: 0.75 }}>
          <FormatAlignLeftIcon sx={{ fontSize: 16 }} />
        </ToggleButton>
        <ToggleButton value="center" aria-label="centered" sx={{ px: 0.75 }}>
          <FormatAlignCenterIcon sx={{ fontSize: 16 }} />
        </ToggleButton>
        <ToggleButton value="right" aria-label="right aligned" sx={{ px: 0.75 }}>
          <FormatAlignRightIcon sx={{ fontSize: 16 }} />
        </ToggleButton>
        <ToggleButton value="justify" aria-label="justified" sx={{ px: 0.75 }}>
          <FormatAlignJustifyIcon sx={{ fontSize: 16 }} />
        </ToggleButton>
      </ToggleButtonGroup>

      <Divider orientation="vertical" flexItem sx={{ my: 0.5 }} />

      {/* Text Color Picker */}
      <Tooltip title="Text Color">
        <Box sx={{ position: 'relative', display: 'inline-flex', alignItems: 'center' }}>
          <IconButton size="small" component="label" sx={{ p: 0.5 }}>
            <FormatColorTextIcon sx={{ fontSize: 18, color: properties.textColor || 'text.primary' }} />
            <input
              type="color"
              hidden
              value={properties.textColor || '#000000'}
              onChange={(e) => onUpdate('textColor', e.target.value)}
            />
          </IconButton>
        </Box>
      </Tooltip>

      {/* Background/Fill Color Picker */}
      <Tooltip title="Background / Fill Color">
        <Box sx={{ position: 'relative', display: 'inline-flex', alignItems: 'center' }}>
          <IconButton size="small" component="label" sx={{ p: 0.5 }}>
            <FormatColorFillIcon sx={{ fontSize: 18, color: properties.backgroundColor || properties.fillColor || 'text.secondary' }} />
            <input
              type="color"
              hidden
              value={properties.backgroundColor || properties.fillColor || '#ffffff'}
              onChange={(e) => {
                onUpdate('backgroundColor', e.target.value);
                onUpdate('fillColor', e.target.value);
              }}
            />
          </IconButton>
        </Box>
      </Tooltip>
    </Paper>
  );
};

export default FormatToolbar;
