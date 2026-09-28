import React from 'react';
import { useTranslation } from 'react-i18next';
import { Chip } from '@mui/material';
import { GoldenStatus } from './api';

const GOLDEN: Record<GoldenStatus, 'success' | 'warning' | 'default' | 'error' | 'info'> = {
  PUBLISHED: 'success', REVIEW: 'warning', SUPERSEDED: 'default', DRAFT: 'info', RETRACTED: 'error',
};

export function GoldenStatusChip({ status }: { status: GoldenStatus }) {
  const { t } = useTranslation();
  return <Chip size="small" color={GOLDEN[status] ?? 'default'} label={t(`mastering.goldenStatus.${status}`, status)} />;
}

