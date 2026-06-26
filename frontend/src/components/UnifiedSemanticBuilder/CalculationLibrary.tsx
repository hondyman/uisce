import React, { useState, useMemo, useEffect } from 'react';
import { devError } from '../../utils/devLogger';
import { Box, Typography, Paper, Divider, CircularProgress, Alert } from '@mui/material';
import IconCalculator from '@tabler/icons-react/dist/esm/icons/IconCalculator.js';
import { IconDatabase, IconCube } from '@tabler/icons-react';
import type { CoreOption } from './financialCalculations';
import { libraryOptions } from './financialCalculations';
import MonacoCodeEditor from './MonacoCodeEditor.lazy';
import './CalculationLibrary.css';
import getErrorMessage from '../../utils/errors';

// Mock lineage data structure
interface LineageInfo {
  datasource_name: string;
  models: Array<{
    model_name: string;
    model_key: string;
    last_updated: string;
  }>;
}

const CalculationLibrary: React.FC = () => {
  const [selectedCalc, setSelectedCalc] = useState<CoreOption | null>(null);
  const [lineage, setLineage] = useState<LineageInfo[] | null>(null);
  const [lineageLoading, setLineageLoading] = useState(false);
  const [lineageError, setLineageError] = useState<string | null>(null);

  const groupedCalculations = useMemo(() => {
    const groups: { [key: string]: CoreOption[] } = {};
    libraryOptions.forEach(calc => {
      const category = calc.category || 'Uncategorized';
      if (!groups[category]) {
        groups[category] = [];
      }
      groups[category].push(calc);
    });
    return groups;
  }, []);

  useEffect(() => {
    if (selectedCalc) {
      const fetchLineage = async () => {
        setLineageLoading(true);
        setLineageError(null);
        setLineage(null);
        try {
          // This is a hypothetical API endpoint. The backend would need to implement
          // the logic to find all models using this calculation.
          const response = await fetch(`/api/fabric/calculations/lineage?name=${selectedCalc.name}`);
          if (!response.ok) {
            // For demo purposes, we'll handle a 404 as "not found" instead of an error.
            if (response.status === 404) {
              setLineage([]);
              return;
            }
            throw new Error(`Failed to fetch lineage: ${response.statusText}`);
          }
          const data = await response.json();
          setLineage(data.lineage || []);
        } catch (e) {
          try { devError("Failed to fetch calculation lineage", getErrorMessage(e)); } catch {}
          setLineageError("Could not load usage information for this calculation.");
        } finally {
          setLineageLoading(false);
        }
      };

      fetchLineage();
    }
  }, [selectedCalc]);

  const renderDetailView = () => {
    if (!selectedCalc) {
      return (
        <div className="calc-library-empty-state">
          <IconCalculator size={48} strokeWidth={1.5} />
          <Typography variant="h6" color="text.secondary">Select a Calculation</Typography>
          <Typography variant="body1" color="text.secondary">
            Choose a calculation from the library on the left to view its details, template, and usage.
          </Typography>
        </div>
      );
    }

    return (
      <div className="calc-detail-view">
        <Typography variant="h4" gutterBottom className="detail-title">
          {selectedCalc.title}
        </Typography>
        <Typography variant="body1" color="text.secondary" paragraph>
          {selectedCalc.description}
        </Typography>
        
        <Divider sx={{ my: 3 }} />

        <Typography variant="h6" gutterBottom>Template Configuration</Typography>
        <Paper variant="outlined" className="code-paper">
          <Typography variant="subtitle2" className="code-label">SQL / Gonja Template</Typography>
          <div className="editor-wrapper-full editor-h-400">
            <MonacoCodeEditor value={selectedCalc.sql} language="json" readOnly />
          </div>
        </Paper>

        {selectedCalc.preAggregationTemplate && (
          <Paper variant="outlined" className="code-paper">
            <Typography variant="subtitle2" className="code-label">Required Pre-Aggregation Template</Typography>
            <div className="editor-wrapper-full editor-h-400">
              <MonacoCodeEditor value={JSON.stringify(selectedCalc.preAggregationTemplate, null, 2)} language="yaml" readOnly />
            </div>
          </Paper>
        )}

        <Divider sx={{ my: 3 }} />

        <Typography variant="h6" gutterBottom>Where Used / Lineage</Typography>
        {lineageLoading && <CircularProgress />}
        {lineageError && <Alert severity="error">{lineageError}</Alert>}
        {lineage && !lineageLoading && !lineageError && (
          <div className="lineage-results">
            {lineage.length === 0 ? (
              <Typography color="text.secondary">This calculation is not currently used in any models.</Typography>
            ) : (
              lineage.map(ds => (
                <div key={ds.datasource_name} className="lineage-datasource-group">
                  <div className="datasource-name">
                    <IconDatabase size={18} /> <Typography variant="subtitle1">{ds.datasource_name}</Typography>
                  </div>
                  <ul className="lineage-model-list">
                    {ds.models.map(model => (
                      <li key={model.model_key}>
                        <IconCube size={16} />
                        <span className="model-name">{model.model_name}</span>
                        <span className="model-key">({model.model_key})</span>
                      </li>
                    ))}
                  </ul>
                </div>
              ))
            )}
          </div>
        )}
      </div>
    );
  };

  return (
    <Box className="calc-library-container">
      <Paper className="calc-library-sidebar">
        <div className="sidebar-header">
          <Typography variant="h6">Calculations</Typography>
        </div>
        <div className="sidebar-content">
          {Object.entries(groupedCalculations).map(([category, calcs]) => (
            <div key={category} className="category-group">
              <Typography variant="overline" className="category-title">{category}</Typography>
              <ul className="calculation-list">
                {calcs.map(calc => (
                  <li 
                    key={calc.name} 
                    className={`calculation-item ${selectedCalc?.name === calc.name ? 'selected' : ''}`}
                    onClick={() => setSelectedCalc(calc)}
                  >
                    {calc.title}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </Paper>
      <Box className="calc-library-main">
        {renderDetailView()}
      </Box>
    </Box>
  );
};

export default CalculationLibrary;