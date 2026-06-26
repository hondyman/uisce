// React default import removed — using automatic JSX runtime
import { Box, Typography, Grid, Paper, Button, TextField, IconButton, Chip, FormControl, InputLabel, Select, MenuItem } from '@mui/material';
import { Add as AddIcon, Delete as DeleteIcon } from '@mui/icons-material';

interface HierarchyLevel {
  name: string;
  title: string;
  dimension: string;
}

interface Hierarchy {
  name: string;
  title: string;
  levels: HierarchyLevel[];
}

interface Props {
  hierarchies: Hierarchy[];
  drillMembers: string[];
  availableDimensions: string[];
  onHierarchiesChange: (h: Hierarchy[]) => void;
  onDrillMembersChange: (d: string[]) => void;
}

const HierarchyDrillMembersEditor: React.FC<Props> = ({ hierarchies, drillMembers, availableDimensions, onHierarchiesChange, onDrillMembersChange }) => {
  const addHierarchy = () => {
    const newH: Hierarchy = { name: `hier_${Date.now()}`, title: 'New Hierarchy', levels: [] };
    onHierarchiesChange([...hierarchies, newH]);
  };

  const addLevel = (index: number) => {
    const h = [...hierarchies];
    h[index].levels.push({ name: `lvl_${Date.now()}`, title: 'New Level', dimension: '' });
    onHierarchiesChange(h);
  };

  const updateLevel = (hIndex: number, lIndex: number, updates: Partial<HierarchyLevel>) => {
    const h = [...hierarchies];
    h[hIndex].levels[lIndex] = { ...h[hIndex].levels[lIndex], ...updates };
    onHierarchiesChange(h);
  };

  const removeLevel = (hIndex: number, lIndex: number) => {
    const h = [...hierarchies];
    h[hIndex].levels = h[hIndex].levels.filter((_, i) => i !== lIndex);
    onHierarchiesChange(h);
  };

  const toggleDrillMember = (dimension: string) => {
    if (drillMembers.includes(dimension)) {
      onDrillMembersChange(drillMembers.filter(d => d !== dimension));
    } else {
      onDrillMembersChange([...drillMembers, dimension]);
    }
  };

  return (
    <Box sx={{ p: 2 }}>
      <Grid container spacing={2}>
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
              <Typography variant="h6">Hierarchies</Typography>
              <Button size="small" startIcon={<AddIcon />} onClick={addHierarchy}>Add</Button>
            </Box>
            {hierarchies.map((h, hi) => (
              <Paper key={h.name} sx={{ p: 1, mb: 1 }}>
                <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
                  <TextField label="Name" size="small" value={h.name} onChange={(e) => {
                    const copy = [...hierarchies]; copy[hi] = { ...copy[hi], name: e.target.value }; onHierarchiesChange(copy);
                  }} />
                  <TextField label="Title" size="small" value={h.title} onChange={(e) => {
                    const copy = [...hierarchies]; copy[hi] = { ...copy[hi], title: e.target.value }; onHierarchiesChange(copy);
                  }} />
                </Box>
                {h.levels.map((lvl, li) => (
                  <Box key={lvl.name} sx={{ display: 'flex', gap: 1, alignItems: 'center', mb: 1 }}>
                    <TextField size="small" label="Level Name" value={lvl.name} onChange={(e) => updateLevel(hi, li, { name: e.target.value })} />
                    <TextField size="small" label="Level Title" value={lvl.title} onChange={(e) => updateLevel(hi, li, { title: e.target.value })} />
                    <FormControl size="small" sx={{ minWidth: 160 }}>
                      <InputLabel>Dimension</InputLabel>
                      <Select value={lvl.dimension} label="Dimension" onChange={(e) => updateLevel(hi, li, { dimension: e.target.value })}>
                        <MenuItem value=""><em>None</em></MenuItem>
                        {availableDimensions.map(dim => <MenuItem key={dim} value={dim}>{dim}</MenuItem>)}
                      </Select>
                    </FormControl>
                    <IconButton size="small" onClick={() => removeLevel(hi, li)}><DeleteIcon /></IconButton>
                  </Box>
                ))}
                <Button size="small" onClick={() => addLevel(hi)} startIcon={<AddIcon />}>Add Level</Button>
              </Paper>
            ))}
          </Paper>
        </Grid>

        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" sx={{ mb: 1 }}>Drill Members</Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
              {availableDimensions.map(dim => (
                <Chip key={dim} label={dim} color={drillMembers.includes(dim) ? 'primary' : 'default'} onClick={() => toggleDrillMember(dim)} />
              ))}
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default HierarchyDrillMembersEditor;
