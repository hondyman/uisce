import type { FC } from 'react';
import { Paper, Grid, Typography, FormControl, InputLabel, Select, MenuItem } from '@mui/material';

type Props = {
  pageSize: string;
  orientation: string;
  onChangePageSize: (v: string) => void;
  onChangeOrientation: (v: string) => void;
};

const PageSettings: FC<Props> = ({ pageSize, orientation, onChangePageSize, onChangeOrientation }) => {
  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Grid container spacing={2} alignItems="center">
        <Grid item>
          <Typography variant="subtitle2">Page Setup:</Typography>
        </Grid>
        <Grid item>
          <FormControl size="small" sx={{ minWidth: 100 }}>
            <InputLabel>Size</InputLabel>
            <Select value={pageSize} onChange={(e) => onChangePageSize(e.target.value)}>
              <MenuItem value="A4">A4</MenuItem>
              <MenuItem value="Letter">Letter</MenuItem>
              <MenuItem value="Legal">Legal</MenuItem>
            </Select>
          </FormControl>
        </Grid>
        <Grid item>
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Orientation</InputLabel>
            <Select value={orientation} onChange={(e) => onChangeOrientation(e.target.value)}>
              <MenuItem value="Portrait">Portrait</MenuItem>
              <MenuItem value="Landscape">Landscape</MenuItem>
            </Select>
          </FormControl>
        </Grid>
      </Grid>
    </Paper>
  );
};

export default PageSettings;
