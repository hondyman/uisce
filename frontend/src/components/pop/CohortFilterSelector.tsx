import React, { useState, useEffect } from 'react';
import {
  Select,
  MenuItem,
  Card,
  CardContent,
  Chip,
  Stack,
  Typography,
} from '@mui/material';
import { Filter, Users, DollarSign, Clock } from 'lucide-react';
import styles from './CohortFilterSelector.module.css';
import { useNotification } from '../../hooks/useNotification';

interface Cohort {
  name: string;
  description: string;
  type: 'behavioral' | 'domain' | 'temporal';
  sql: string;
  source: string;
  estimated_size: string;
}

interface CohortValue {
  value: string;
  metadata: any;
  label: string;
}

interface CohortFilterSelectorProps {
  onCohortSelect: (cohort: Cohort, values: string[]) => void;
  selectedCohort?: Cohort;
  selectedValues?: string[];
}

const CohortFilterSelector: React.FC<CohortFilterSelectorProps> = ({
  onCohortSelect,
  selectedCohort,
  selectedValues = []
}) => {
  const notification = useNotification();
  const [cohorts, setCohorts] = useState<Cohort[]>([]);
  const [availableValues, setAvailableValues] = useState<CohortValue[]>([]);
  const [loading, setLoading] = useState(false);
  const [valuesLoading, setValuesLoading] = useState(false);

  useEffect(() => {
    fetchCohorts();
  }, []);

  const fetchCohorts = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/cohorts');
      const data = await response.json();
      setCohorts(data.cohorts);
    } catch (error) {
      notification.error('Failed to fetch cohorts');
    } finally {
      setLoading(false);
    }
  };

  const fetchCohortValues = async (cohortName: string) => {
    setValuesLoading(true);
    try {
      const response = await fetch(`/api/cohorts/${cohortName}/values`);
      const data = await response.json();
      setAvailableValues(data.values);
    } catch (error) {
      notification.error('Failed to fetch cohort values');
    } finally {
      setValuesLoading(false);
    }
  };

  const handleCohortChange = (event: any) => {
    const cohortName = event.target.value;
    const cohort = cohorts.find(c => c.name === cohortName);
    if (cohort) {
      setAvailableValues([]);
      fetchCohortValues(cohortName);
      onCohortSelect(cohort, []);
    }
  };

  const handleValuesChange = (event: any) => {
    const values = event.target.value;
    if (selectedCohort) {
      onCohortSelect(selectedCohort, values);
    }
  };

  const getCohortIcon = (type: string) => {
    switch (type) {
      case 'behavioral':
        return <Users size={18} />;
      case 'domain':
        return <DollarSign size={18} />;
      case 'temporal':
        return <Clock size={18} />;
      default:
        return <Filter size={18} />;
    }
  };

  const getCohortColor = (type: string) => {
    switch (type) {
      case 'behavioral':
        return '#2196F3';
      case 'domain':
        return '#4CAF50';
      case 'temporal':
        return '#FF9800';
      default:
        return '#9E9E9E';
    }
  };

  return (
    <Card className={styles.cohortFilterSelector}>
      <CardContent>
        <Stack spacing={2} sx={{ width: '100%' }}>
          <div>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, marginBottom: 1 }}>
              Select Cohort:
            </Typography>
            <Select
              className={styles.cohortSelect}
              placeholder="Choose a cohort filter"
              value={selectedCohort?.name || ''}
              onChange={handleCohortChange}
              disabled={loading}
              fullWidth
            >
              {cohorts.map(cohort => (
                <MenuItem key={cohort.name} value={cohort.name}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    {getCohortIcon(cohort.type)}
                    <span>{cohort.name.replace(/_/g, ' ')}</span>
                    <Chip
                      label={cohort.type}
                      size="small"
                      sx={{ backgroundColor: getCohortColor(cohort.type), color: 'white' }}
                    />
                  </Stack>
                </MenuItem>
              ))}
            </Select>
          </div>

          {selectedCohort && (
            <div>
              <Typography variant="body2" className={styles.cohortDescription}>
                {selectedCohort.description}
              </Typography>
              <Typography variant="caption" className={styles.cohortMetadata}>
                Source: {selectedCohort.source} | Size: {selectedCohort.estimated_size}
              </Typography>
            </div>
          )}

          {selectedCohort && availableValues.length > 0 && (
            <div>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, marginBottom: 1 }}>
                Select Values:
              </Typography>
              <Select
                multiple
                className={styles.valuesSelect}
                value={selectedValues}
                onChange={handleValuesChange}
                disabled={valuesLoading}
                fullWidth
              >
                {availableValues.map(value => (
                  <MenuItem key={value.value} value={value.value}>
                    {value.label}
                  </MenuItem>
                ))}
              </Select>
            </div>
          )}

          {selectedValues.length > 0 && (
            <div>
              <Typography variant="body2" className={styles.selectedValuesInfo}>
                Selected {selectedValues.length} value(s)
              </Typography>
            </div>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
};

export default CohortFilterSelector;
