import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import type { ColumnMeta, PageInfo } from './types';

export default function ResultsGrid({ rows, columns, page, onPageChange }: { rows: Record<string, unknown>[], columns: ColumnMeta[], page: PageInfo, onPageChange: (p: PageInfo) => void }) {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  return (
    <Box sx={{ width: '100%' }}>
      <TableContainer component={Paper} variant="outlined">
        <Table>
          <TableHead>
            <TableRow>
              {columns.map(c => (
                <TableCell key={c.name} sx={{ fontWeight: 600 }}>
                  {c.name}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((r, i) => (
              <TableRow key={i}>
                {columns.map(c => (
                  <TableCell key={c.name}>{String(r[c.name])}</TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, mt: 2 }}>
        <Button
          variant="outlined"
          size="small"
          disabled={!page.offset || page.offset === 0}
          onClick={() => onPageChange({ ...page, offset: (page.offset || 0) - (page.limit || 50) })}
        >
          Prev
        </Button>
        <Button
          variant="outlined"
          size="small"
          disabled={!page.hasNext}
          onClick={() => onPageChange({ ...page, offset: (page.offset || 0) + (page.limit || 50) })}
        >
          Next
        </Button>
      </Box>
    </Box>
  );
}
