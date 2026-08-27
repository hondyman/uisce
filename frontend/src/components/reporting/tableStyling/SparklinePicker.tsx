import React from 'react';
import { Box, Typography, FormControl, InputLabel, Select, MenuItem, TextField, Button, Divider } from '@mui/material';
import { SparklineConfig } from '../tableColumnModel';

interface SparklinePickerProps {
  sparkline?: SparklineConfig;
  onChange: (sp: SparklineConfig | undefined) => void;
}

export const SparklinePicker: React.FC<SparklinePickerProps> = ({ sparkline, onChange }) => {
  const update = (partial: Partial<SparklineConfig>) =>
    onChange(sparkline ? { ...sparkline, ...partial } : { type: 'line', color: '#00D4FF', ...partial });

  const types: SparklineConfig['type'][] = ['line', 'bar', 'win-loss'];

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <FormControl size="small" fullWidth>
        <InputLabel sx={{ fontSize: '0.7rem' }}>Type</InputLabel>
        <Select value={sparkline?.type || 'line'} label="Type"
          onChange={e => update({ type: e.target.value as SparklineConfig['type'] })}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}>
          {types.map(t => <MenuItem key={t} value={t} sx={{ fontSize: '0.72rem', textTransform: 'capitalize' }}>{t}</MenuItem>)}
        </Select>
      </FormControl>

      <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

      <Box sx={{ display: 'flex', gap: 1 }}>
        <TextField size="small" label="Color" type="color" value={sparkline?.color || '#00D4FF'}
          onChange={e => update({ color: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
        <TextField size="small" label="High" type="color" value={sparkline?.highColor || '#10B981'}
          onChange={e => update({ highColor: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
      </Box>
      <Box sx={{ display: 'flex', gap: 1 }}>
        <TextField size="small" label="Low" type="color" value={sparkline?.lowColor || '#EF4444'}
          onChange={e => update({ lowColor: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
        <TextField size="small" label="Neg" type="color" value={sparkline?.negativeColor || '#F59E0B'}
          onChange={e => update({ negativeColor: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
      </Box>

      {/* Mini preview */}
      <Box sx={{ height: 32, display: 'flex', alignItems: 'center', gap: 0.5, px: 1 }}>
        <SparklineInline type={sparkline?.type || 'line'} color={sparkline?.color || '#00D4FF'} />
      </Box>
    </Box>
  );
};

interface SparklineInlineProps {
  type: SparklineConfig['type'];
  color: string;
  data?: number[];
  highColor?: string;
  lowColor?: string;
  negativeColor?: string;
}

export const SparklineInline: React.FC<SparklineInlineProps> = ({
  type,
  color,
  data = [3, 5, 2, 8, 4, 6, 5, 7, 3, 9, 5, 8, 6],
  highColor,
  lowColor,
  negativeColor,
}) => {
  if (data.length < 3) {
    return <Typography variant="caption" color="text.disabled" sx={{ fontSize: '0.6rem' }}>—</Typography>;
  }

  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const W = 80;
  const H = 20;
  const pad = 2;

  const toX = (i: number) => pad + (i / (data.length - 1)) * (W - pad * 2);
  const toY = (v: number) => pad + (1 - (v - min) / range) * (H - pad * 2);

  if (type === 'line') {
    const points = data.map((v, i) => `${toX(i)},${toY(v)}`).join(' ');
    const fillPoints = `${toX(0)},${H} ${points} ${toX(data.length - 1)},${H}`;
    return (
      <svg width={W} height={H} viewBox={`0 0 ${W} ${H}`} style={{ overflow: 'visible' }}>
        <defs>
          <linearGradient id={`sg_${color.replace('#','')}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.3} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <polyline points={fillPoints} fill={`url(#sg_${color.replace('#','')})`} />
        <polyline points={points} fill="none" stroke={color} strokeWidth={1.5} strokeLinejoin="round" strokeLinecap="round" />
        {highColor && data[data.length - 1] === max && (
          <circle cx={toX(data.length - 1)} cy={toY(max)} r={2.5} fill={highColor} />
        )}
        {lowColor && data[data.length - 1] === min && (
          <circle cx={toX(data.length - 1)} cy={toY(min)} r={2.5} fill={lowColor} />
        )}
      </svg>
    );
  }

  if (type === 'bar') {
    const barW = Math.max(1, (W - pad * 2) / data.length - 1);
    return (
      <svg width={W} height={H} viewBox={`0 0 ${W} ${H}`}>
        {data.map((v, i) => {
          const barH = Math.abs((v - min) / range) * (H - pad * 2);
          const y = v >= min ? toY(max) : toY(min);
          const last = i === data.length - 1;
          return (
            <rect
              key={i}
              x={toX(i) - barW / 2}
              y={y}
              width={barW}
              height={Math.max(1, barH)}
              fill={last ? color : `${color}80`}
            />
          );
        })}
      </svg>
    );
  }

  if (type === 'win-loss') {
    const mid = toY(min + range / 2);
    const barW = Math.max(1, (W - pad * 2) / data.length - 1);
    return (
      <svg width={W} height={H} viewBox={`0 0 ${W} ${H}`}>
        {data.map((v, i) => {
          const isPos = v >= (min + range / 2);
          const barH = Math.max(1, Math.abs((v - (min + range / 2)) / range) * (H - pad * 2));
          const c = isPos ? (color) : (negativeColor || '#EF4444');
          return (
            <rect
              key={i}
              x={toX(i) - barW / 2}
              y={isPos ? mid - barH : mid}
              width={barW}
              height={barH}
              fill={c}
            />
          );
        })}
        <line x1={pad} y1={mid} x2={W - pad} y2={mid} stroke="rgba(255,255,255,0.15)" strokeWidth={0.5} />
      </svg>
    );
  }

  return null;
};

export default SparklinePicker;
