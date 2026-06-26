// React default import removed — using automatic JSX runtime
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'; // unused imports
import { Box, Typography, Chip as _Chip } from '@mui/material';
import { TableChart as ExcelIcon } from '@mui/icons-material';

interface ExcelAwareChartProps {
  data: any[];
  xKey: string;
  yKey: string;
  title: string;
  excelFormula?: string;
  excelArguments?: Record<string, any>;
}

interface ExcelTooltipProps {
  active?: boolean;
  payload?: any[];
  label?: string;
  excelFormula?: string;
  excelArguments?: Record<string, any>;
}

const ExcelTooltip: React.FC<ExcelTooltipProps> = ({
  active,
  payload,
  label,
  excelFormula,
  excelArguments
}) => {
  if (!active || !payload || !payload.length) return null;

  const value = payload[0]?.value;

  return (
    <Box sx={{
      background: 'white',
      padding: 2,
      border: '1px solid #ccc',
      borderRadius: 1,
      boxShadow: 1,
      maxWidth: 300
    }}>
      <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
        {label}
      </Typography>
      <Typography variant="body2">
        Value: {value?.toFixed(4)}
      </Typography>

      {excelFormula && (
        <Box sx={{ mt: 1, pt: 1, borderTop: '1px solid #eee' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <ExcelIcon sx={{ mr: 1, color: 'green', fontSize: 16 }} />
            <Typography variant="caption" sx={{ fontWeight: 'bold', color: 'green' }}>
              Excel Formula
            </Typography>
          </Box>
          <Typography variant="caption" sx={{ fontFamily: 'monospace', display: 'block', mb: 1 }}>
            {excelFormula}
          </Typography>
          {excelArguments && (
            <Box>
              <Typography variant="caption" sx={{ fontWeight: 'bold' }}>
                Arguments:
              </Typography>
              {Object.entries(excelArguments).map(([key, value]) => (
                <Typography key={key} variant="caption" sx={{ display: 'block', ml: 1 }}>
                  {key}: {Array.isArray(value) ? `[${value.join(', ')}]` : value}
                </Typography>
              ))}
            </Box>
          )}
        </Box>
      )}
    </Box>
  );
};

const ExcelAwareChart: React.FC<ExcelAwareChartProps> = ({
  data,
  xKey,
  yKey,
  title,
  excelFormula,
  excelArguments
}) => {
  return (
    <Box sx={{ width: '100%', height: 400, p: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6" sx={{ mr: 2 }}>
          {title}
        </Typography>
        {excelFormula && (
          <_Chip
            icon={<ExcelIcon />}
            label="Excel Formula"
            size="small"
            sx={{
              backgroundColor: 'rgba(76, 175, 80, 0.1)',
              color: 'green',
              '& .MuiChip-icon': { color: 'green' }
            }}
          />
        )}
      </Box>

      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis
            dataKey={xKey}
            tick={{ fontSize: 12 }}
          />
          <YAxis
            tick={{ fontSize: 12 }}
          />
          <Tooltip
            content={
              <ExcelTooltip
                excelFormula={excelFormula}
                excelArguments={excelArguments}
              />
            }
          />
          <Legend />
          <Line
            type="monotone"
            dataKey={yKey}
            stroke="#4E79A7"
            strokeWidth={2}
            dot={{ r: 4 }}
            activeDot={{ r: 6 }}
          />
        </LineChart>
      </ResponsiveContainer>
    </Box>
  );
};

export default ExcelAwareChart;
