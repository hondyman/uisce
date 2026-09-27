import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  FormControl, FormControlLabel, InputLabel, MenuItem, Select, Stack, Switch, TextField,
} from '@mui/material';

/** Report-specific target.params (bursting / export / notify) for ScheduleEditor.paramsSlot. */
export default function ReportDeliveryParams({
  params,
  setParams,
}: {
  params: Record<string, unknown>;
  setParams: (next: Record<string, unknown>) => void;
}) {
  const { t } = useTranslation();
  const set = (key: string, value: unknown) => setParams({ ...params, [key]: value });

  const burst = String(params.burst_dimension ?? 'client_id');
  const format = String(params.export_format ?? 'PDF') as 'PDF' | 'EXCEL' | 'BOTH';
  const notifyInApp = params.notify_in_app !== false;
  const notifyEmail = params.notify_email === true;

  return (
    <Stack spacing={2}>
      <TextField
        label={t('schedules.reportParams.burstDimension')}
        value={burst}
        onChange={(e) => set('burst_dimension', e.target.value)}
        helperText={t('schedules.reportParams.burstHelp')}
      />
      <FormControl sx={{ minWidth: 200 }}>
        <InputLabel>{t('schedules.reportParams.exportFormat')}</InputLabel>
        <Select
          label={t('schedules.reportParams.exportFormat')}
          value={format}
          onChange={(e) => set('export_format', e.target.value)}
        >
          <MenuItem value="PDF">PDF</MenuItem>
          <MenuItem value="EXCEL">Excel</MenuItem>
          <MenuItem value="BOTH">{t('schedules.reportParams.bothFormats')}</MenuItem>
        </Select>
      </FormControl>
      <FormControlLabel
        control={<Switch checked={notifyInApp} onChange={(e) => set('notify_in_app', e.target.checked)} />}
        label={t('schedules.reportParams.notifyInApp')}
      />
      <FormControlLabel
        control={<Switch checked={notifyEmail} onChange={(e) => set('notify_email', e.target.checked)} />}
        label={t('schedules.reportParams.notifyEmail')}
      />
    </Stack>
  );
}
