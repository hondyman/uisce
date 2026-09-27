import React from 'react';
import { useTranslation } from 'react-i18next';
import { Chip } from '@mui/material';
import { GoldenStatus, RunStatus } from './api';

const GOLDEN: Record<GoldenStatus, 'success' | 'warning' | 'default' | 'error' | 'info'> = {
  PUBLISHED: 'success', REVIEW: 'warning', SUPERSEDED: 'default', DRAFT: 'info', RETRACTED: 'error',
};
const RUN: Record<RunStatus, 'success' | 'warning' | 'info' | 'error'> = {
  COMPLETED: 'success', PARTIAL: 'warning', RUNNING: 'info', FAILED: 'error',
};

export function GoldenStatusChip({ status }: { status: GoldenStatus }) {
  const { t } = useTranslation();
  return <Chip size="small" color={GOLDEN[status] ?? 'default'} label={t(`mastering.goldenStatus.${status}`, status)} />;
}

export function RunStatusChip({ status }: { status: RunStatus }) {
  const { t } = useTranslation();
  return <Chip size="small" color={RUN[status] ?? 'default'} label={t(`mastering.runStatus.${status}`, status)} />;
}
