import React, { useEffect, useState } from 'react';
import { devWarn } from '../../utils/devLogger';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Alert,
  CircularProgress,
  Tooltip,
  TextField,
  Autocomplete,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  Add as AddIcon,
  Remove as RemoveIcon,
  Link as LinkIcon,
  Storage as TableIcon,
  AccountTree as AccountTreeIcon,
  AutoAwesome as AutoAwesomeIcon,
} from '@mui/icons-material';
import { 
  JoinSuggestion, 
  JoinExtractionService,
  TableJoinDefinitions,
  GeneratedCube,
} from '../../services/joinExtractionService';

interface DatabaseJoinExplorerProps {
  datasourceId: string;
  selectedTable?: string;
  onJoinSelect?: (join: JoinSuggestion) => void;
  onCubeGenerate?: (cube: GeneratedCube) => void;
  className?: string;
}

interface JoinPathVisualizationProps {
  path: string[];
  joinStatements: string[];
  onPathUpdate?: (path: string[]) => void;
}

const JoinPathVisualization: React.FC<JoinPathVisualizationProps> = ({ path, joinStatements }) => {
  return (
    <Box sx={{ mt: 2 }}>
      <Typography variant="h6" gutterBottom>
        <AccountTreeIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
        Join Path
      </Typography>

      <Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 1 }}>
        {path.map((table, index) => (
          <React.Fragment key={table + index}>
            <Chip label={table} variant="outlined" color="primary" icon={<TableIcon />} />
            {index < path.length - 1 && (
              <Tooltip title={joinStatements[index] || 'Join statement'}>
                <LinkIcon color="action" />
              </Tooltip>
            )}
          </React.Fragment>
        ))}
      </Box>

      {joinStatements.length > 0 && (
        <Box sx={{ mt: 2 }}>
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Join SQL:
          </Typography>
          <Box
            component="pre"
            sx={{
              backgroundColor: 'grey.100',
              p: 1,
              borderRadius: 1,
              fontSize: '0.875rem',
              overflow: 'auto',
            }}
          >
            {joinStatements.join('\n')}
          </Box>
        </Box>
      )}
    </Box>
  );
};

export const DatabaseJoinExplorer: React.FC<DatabaseJoinExplorerProps> = ({
  datasourceId,
  selectedTable,
  onJoinSelect,
  onCubeGenerate,
  className,
}) => {
  const [joinService] = useState(() => new JoinExtractionService());
  const [joinSuggestions, setJoinSuggestions] = useState<JoinSuggestion[]>([]);
  const [, setTableJoins] = useState<TableJoinDefinitions>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedJoins, setSelectedJoins] = useState<JoinSuggestion[]>([]);
  const [joinPath, setJoinPath] = useState<string[]>([]);
  const [joinStatements, setJoinStatements] = useState<string[]>([]);
  const [availableTables, setAvailableTables] = useState<string[]>([]);
  const [targetTable, setTargetTable] = useState<string>('');

  // Load join suggestions on mount
  useEffect(() => {
    if (datasourceId) {
      loadJoinSuggestions();
    }
  }, [datasourceId]);

  // Load table-specific joins when selected table changes
  useEffect(() => {
    if (datasourceId && selectedTable) {
      loadTableJoins(selectedTable);
      setJoinPath([selectedTable]);
    }
  }, [datasourceId, selectedTable]);

  const loadJoinSuggestions = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await joinService.extractJoinSuggestions(datasourceId);
      setJoinSuggestions(response.joins);
      
      // Extract unique table names
      const tables = new Set<string>();
      response.joins.forEach(join => {
        tables.add(join.source_table);
        tables.add(join.target_table);
      });
      setAvailableTables(Array.from(tables).sort());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load join suggestions');
    } finally {
      setLoading(false);
    }
  };

  const loadTableJoins = async (tableName: string) => {
    try {
      const response = await joinService.getTableJoinDefinitions(datasourceId, tableName);
      setTableJoins(response.joins);
    } catch (err) {
      devWarn(`Failed to load table joins for ${tableName}:`, err);
      setTableJoins({});
    }
  };

  const handleJoinToggle = (join: JoinSuggestion) => {
    const isSelected = selectedJoins.some(
      j => j.source_table === join.source_table && j.target_table === join.target_table
    );
    
    let newSelectedJoins;
    if (isSelected) {
      newSelectedJoins = selectedJoins.filter(
        j => !(j.source_table === join.source_table && j.target_table === join.target_table)
      );
    } else {
      newSelectedJoins = [...selectedJoins, join];
    }
    
    setSelectedJoins(newSelectedJoins);
    onJoinSelect?.(join);
  };

  const buildJoinPath = async () => {
    if (!selectedTable || !targetTable) {
      return;
    }
    
    setLoading(true);
    try {
      const path = await joinService.buildJoinPath(datasourceId, selectedTable, targetTable);
      setJoinPath(path);
      
      if (path.length > 1) {
        const statements = await joinService.generateJoinSQL(datasourceId, path);
        setJoinStatements(statements);
      } else {
        setJoinStatements([]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to build join path');
    } finally {
      setLoading(false);
    }
  };

  const generateCubeFromTable = async () => {
    if (!selectedTable) {
      return;
    }
    
    setLoading(true);
    try {
      const response = await joinService.generateCubeFromTable(datasourceId, selectedTable);
      onCubeGenerate?.(response.cube);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate cube');
    } finally {
      setLoading(false);
    }
  };

  const getRelationshipColor = (relationship: string) => {
    switch (relationship) {
      case 'one_to_one': return 'success';
      case 'one_to_many': return 'info';
      case 'many_to_one': return 'warning';
      case 'many_to_many': return 'error';
      default: return 'default';
    }
  };

  if (loading && joinSuggestions.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Card className={className}>
      <CardContent>
        <Typography variant="h5" gutterBottom>
          <LinkIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          Database Join Explorer
        </Typography>
        
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {/* Join Path Builder */}
        {selectedTable && (
          <Accordion defaultExpanded>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="h6">Join Path Builder</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', mb: 2 }}>
                <TextField
                  label="Source Table"
                  value={selectedTable}
                  disabled
                  size="small"
                  sx={{ minWidth: 150 }}
                />
                
                <Autocomplete
                  size="small"
                  sx={{ minWidth: 150 }}
                  options={availableTables.filter(t => t !== selectedTable)}
                  value={targetTable}
                  onChange={(_, newValue) => setTargetTable(newValue || '')}
                  renderInput={(params) => (
                    <TextField {...params} label="Target Table" />
                  )}
                />
                
                <Button
                  variant="contained"
                  onClick={buildJoinPath}
                  disabled={!targetTable || loading}
                  startIcon={loading ? <CircularProgress size={16} /> : <AccountTreeIcon />}
                >
                  Build Path
                </Button>
              </Box>
              
              {joinPath.length > 1 && (
                <JoinPathVisualization
                  path={joinPath}
                  joinStatements={joinStatements}
                  onPathUpdate={setJoinPath}
                />
              )}
            </AccordionDetails>
          </Accordion>
        )}

        {/* Cube Generator */}
        {selectedTable && (
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="h6">Auto-Generate Cube</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Typography variant="body2" color="text.secondary" sx={{ flex: 1 }}>
                  Generate a complete Cube.js definition with dimensions, measures, and joins
                  based on the database schema for table: <strong>{selectedTable}</strong>
                </Typography>
                
                <Button
                  variant="contained"
                  color="secondary"
                  onClick={generateCubeFromTable}
                  disabled={loading}
                  startIcon={loading ? <CircularProgress size={16} /> : <AutoAwesomeIcon />}
                >
                  Generate Cube
                </Button>
              </Box>
            </AccordionDetails>
          </Accordion>
        )}

        {/* Available Joins */}
        <Accordion defaultExpanded>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="h6">
              Available Joins ({joinSuggestions.length})
            </Typography>
          </AccordionSummary>
          <AccordionDetails>
            {joinSuggestions.length === 0 ? (
              <Typography color="text.secondary">
                No join relationships found in the database metadata.
              </Typography>
            ) : (
              <List dense>
                {joinSuggestions.map((join, index) => {
                  const isSelected = selectedJoins.some(
                    j => j.source_table === join.source_table && j.target_table === join.target_table
                  );
                  
                  return (
                    <ListItem
                      key={index}
                      sx={{
                        border: 1,
                        borderColor: 'divider',
                        borderRadius: 1,
                        mb: 1,
                        backgroundColor: isSelected ? 'action.selected' : 'background.paper',
                      }}
                    >
                      <ListItemText
                        primary={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Typography variant="subtitle2" component="span">
                              {join.source_table}.{join.source_column}
                            </Typography>
                            <LinkIcon fontSize="small" />
                            <Typography variant="subtitle2" component="span">
                              {join.target_table}.{join.target_column}
                            </Typography>
                            <Chip
                              size="small"
                              label={join.relationship.replace(/_/g, '-')}
                              color={getRelationshipColor(join.relationship) as any}
                            />
                          </Box>
                        }
                        secondary={
                          <Box sx={{ mt: 1 }}>
                            <Typography variant="body2" color="text.secondary" component="div">
                              {join.description}
                            </Typography>
                            <Typography
                              variant="caption"
                              component="span"
                              sx={{
                                fontFamily: 'monospace',
                                backgroundColor: 'grey.100',
                                p: 0.5,
                                borderRadius: 0.5,
                                display: 'inline-block',
                                mt: 0.5,
                              }}
                            >
                              {join.join_sql}
                            </Typography>
                          </Box>
                        }
                        primaryTypographyProps={{ component: 'div' }}
                        secondaryTypographyProps={{ component: 'div' }}
                      />
                      <ListItemSecondaryAction>
                        <IconButton
                          edge="end"
                          onClick={() => handleJoinToggle(join)}
                          color={isSelected ? 'primary' : 'default'}
                        >
                          {isSelected ? <RemoveIcon /> : <AddIcon />}
                        </IconButton>
                      </ListItemSecondaryAction>
                    </ListItem>
                  );
                })}
              </List>
            )}
          </AccordionDetails>
        </Accordion>

        {/* Selected Joins Summary */}
        {selectedJoins.length > 0 && (
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="h6">
                Selected Joins ({selectedJoins.length})
              </Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {selectedJoins.map((join, index) => (
                  <Chip
                    key={index}
                    label={`${join.source_table} → ${join.target_table}`}
                    onDelete={() => handleJoinToggle(join)}
                    color="primary"
                    variant="outlined"
                  />
                ))}
              </Box>
            </AccordionDetails>
          </Accordion>
        )}
      </CardContent>
    </Card>
  );
};

export default DatabaseJoinExplorer;
