import React from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, AlertColor, Chip, Typography } from '@mui/material';
import { CatalogError } from '../../utils/catalogError';
import { Change, Severity } from './api';

export function alertSeverity(s: Severity): AlertColor {
  switch (s) {
    case 'Message':
      return 'info';
    case 'Warning':
      return 'warning';
    default:
      return 'error';
  }
}

const SEVERITY_COLOR: Record<Severity, 'default' | 'info' | 'warning' | 'error'> = {
  Message: 'info',
  Warning: 'warning',
  Error: 'error',
  Fatal: 'error',
};

export function SeverityChip({ severity }: { severity: Severity }) {
  const { t } = useTranslation();
  return (
    <Chip size="small" color={SEVERITY_COLOR[severity] ?? 'default'} variant={severity === 'Fatal' ? 'filled' : 'outlined'}
      label={t(`messageCatalog.severities.${severity}`)} />
  );
}

const STATUS_COLOR: Record<Change['status'], 'default' | 'success' | 'warning' | 'error'> = {
  pending: 'warning',
  applied: 'success',
  rejected: 'error',
  withdrawn: 'default',
};

export function StatusChip({ status }: { status: Change['status'] }) {
  const { t } = useTranslation();
  return <Chip size="small" color={STATUS_COLOR[status]} label={t(`messageCatalog.status.${status}`)} />;
}

/** Shows an error from the catalog API: its message, what to do, the code and the reference. */
export function CatalogErrorAlert({ error }: { error: unknown }) {
  const { t } = useTranslation();
  if (error instanceof CatalogError) {
    return (
      <Alert severity={error.severity === 'Warning' ? 'warning' : 'error'}>
        <Typography variant="body2">{error.message}</Typography>
        {error.userAction && <Typography variant="body2" sx={{ mt: 0.5 }}>{error.userAction}</Typography>}
        <Typography variant="caption" sx={{ display: 'block', mt: 0.5, opacity: 0.75 }}>
          {t('messageCatalog.errorRef', { code: error.code, ref: error.correlationId })}
        </Typography>
      </Alert>
    );
  }
  return <Alert severity="error">{error instanceof Error ? error.message : String(error)}</Alert>;
}
