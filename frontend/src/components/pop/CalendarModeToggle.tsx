import React, { useState, useEffect } from 'react';
import { devError } from '../../utils/devLogger';
import { Select, MenuItem, Card, CardContent, Stack, Typography, Tooltip } from '@mui/material';
import { Calendar, Clock, Settings } from 'lucide-react';
import styles from './CalendarModeToggle.module.css';

interface CalendarModeToggleProps {
  onModeChange: (mode: string, granularity?: string) => void;
  selectedMode?: string;
  availableGranularities?: string[];
}

const CalendarModeToggle: React.FC<CalendarModeToggleProps> = ({
  onModeChange,
  selectedMode = 'gregorian',
  availableGranularities = ['day', 'week', 'month', 'quarter', 'year']
}) => {
  const [calendarMode, setCalendarMode] = useState(selectedMode);
  const [selectedGranularity, setSelectedGranularity] = useState('month');
  const [fiscalGranularities, setFiscalGranularities] = useState<string[]>([]);

  useEffect(() => {
    fetchFiscalGranularities();
  }, []);

  const fetchFiscalGranularities = async () => {
    try {
      const response = await fetch('/api/granularities');
      const data = await response.json();

      const fiscalGrans = data.granularities
        .filter((g: any) => g.schema_def.calendar_type === 'fiscal')
        .map((g: any) => g.name);

      setFiscalGranularities(fiscalGrans);
    } catch (error) {
      devError('Failed to fetch fiscal granularities:', error);
    }
  };

  const handleModeChange = (mode: string) => {
    setCalendarMode(mode);
    let defaultGranularity = 'month';

    switch (mode) {
      case 'fiscal':
        defaultGranularity = fiscalGranularities.length > 0 ? fiscalGranularities[0] : 'fiscal_year';
        break;
      case 'iso_week':
        defaultGranularity = 'iso_week';
        break;
      case 'custom':
        defaultGranularity = 'custom_week';
        break;
      default:
        defaultGranularity = 'month';
    }

    setSelectedGranularity(defaultGranularity);
    onModeChange(mode, defaultGranularity);
  };

  const handleGranularityChange = (granularity: string) => {
    setSelectedGranularity(granularity);
    onModeChange(calendarMode, granularity);
  };

  const getModeIcon = (mode: string) => {
    switch (mode) {
      case 'gregorian':
        return <Calendar size={18} />;
      case 'fiscal':
        return <Clock size={18} />;
      case 'iso_week':
        return <Settings size={18} />;
      case 'custom':
        return <Settings size={18} />;
      default:
        return <Calendar size={18} />;
    }
  };

  const getModeDescription = (mode: string) => {
    switch (mode) {
      case 'gregorian':
        return 'Standard Gregorian calendar';
      case 'fiscal':
        return 'Fiscal year calendar with custom offsets';
      case 'iso_week':
        return 'ISO 8601 week numbering';
      case 'custom':
        return 'Custom calendar with flexible week starts';
      default:
        return 'Standard calendar';
    }
  };

  const getAvailableGranularities = () => {
    switch (calendarMode) {
      case 'fiscal':
        return fiscalGranularities.length > 0 ? fiscalGranularities : ['fiscal_year', 'fiscal_quarter', 'fiscal_month'];
      case 'iso_week':
        return ['iso_year', 'iso_week'];
      case 'custom':
        return ['custom_week', 'custom_month', 'custom_quarter'];
      default:
        return availableGranularities;
    }
  };

  return (
    <Card className={styles.calendarModeCard}>
      <CardContent className="p-0">
        <Stack spacing={2} sx={{ width: '100%' }}>
          <div>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Calendar Mode:
            </Typography>
            <Select
              value={calendarMode}
              onChange={(e) => handleModeChange(e.target.value)}
              sx={{ width: '100%', marginTop: 1 }}
              placeholder="Select calendar mode"
            >
              <MenuItem value="gregorian">
                <Stack direction="row" spacing={1} alignItems="center">
                  <Calendar size={18} />
                  <span>Gregorian</span>
                </Stack>
              </MenuItem>
              <MenuItem value="fiscal">
                <Stack direction="row" spacing={1} alignItems="center">
                  <Clock size={18} />
                  <span>Fiscal</span>
                </Stack>
              </MenuItem>
              <MenuItem value="iso_week">
                <Stack direction="row" spacing={1} alignItems="center">
                  <Settings size={18} />
                  <span>ISO Week</span>
                </Stack>
              </MenuItem>
              <MenuItem value="custom">
                <Stack direction="row" spacing={1} alignItems="center">
                  <Settings size={18} />
                  <span>Custom</span>
                </Stack>
              </MenuItem>
            </Select>
          </div>

          <div>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Time Granularity:
            </Typography>
            <Select
              value={selectedGranularity}
              onChange={(e) => handleGranularityChange(e.target.value)}
              sx={{ width: '100%', marginTop: 1 }}
              placeholder="Select time granularity"
            >
              {getAvailableGranularities().map(granularity => (
                <MenuItem key={granularity} value={granularity}>
                  {granularity.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                </MenuItem>
              ))}
            </Select>
          </div>

          <div className={styles.modeDescription}>
            <Tooltip title={getModeDescription(calendarMode)}>
              <Stack direction="row" spacing={1} alignItems="flex-start">
                {getModeIcon(calendarMode)}
                <Typography variant="caption" sx={{ fontSize: '12px', color: 'text.secondary' }}>
                  {getModeDescription(calendarMode)}
                </Typography>
              </Stack>
            </Tooltip>
          </div>

          {calendarMode === 'fiscal' && fiscalGranularities.length === 0 && (
            <div className={styles.warningMessage}>
              <Typography variant="caption" sx={{ fontSize: '12px', color: 'warning.main' }}>
                No fiscal granularities configured. Create them in the steward cockpit.
              </Typography>
            </div>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
};

export default CalendarModeToggle;
