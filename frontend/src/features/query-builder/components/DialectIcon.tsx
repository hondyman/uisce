/**
 * DialectIcon — maps backend datasource identifiers to MUI visual assets.
 *
 * Provides a glanceable indication of which physical engine (Postgres,
 * StarRocks, Snowflake, SQL Server) is processing each plan node.
 */

import React from 'react';
import StorageIcon from '@mui/icons-material/Storage';
import CloudIcon from '@mui/icons-material/Cloud';
import AcUnitIcon from '@mui/icons-material/AcUnit';
import DnsIcon from '@mui/icons-material/Dns';
import { SvgIconProps } from '@mui/material';

interface Props {
  dialect?: string;
  size?: SvgIconProps['fontSize'];
}

function normalizeDialect(dialect?: string): string {
  return (dialect || '').toLowerCase().replace(/[^a-z0-9]/g, '');
}

export const DialectIcon: React.FC<Props> = ({ dialect, size = 'small' }) => {
  const normalized = normalizeDialect(dialect);

  // Snowflake
  if (normalized.includes('snowflake')) {
    return <AcUnitIcon fontSize={size} sx={{ color: '#29B5E8' }} />;
  }

  // SQL Server
  if (
    normalized.includes('sqlserver') ||
    normalized.includes('mssql') ||
    normalized.includes('tsql')
  ) {
    return <DnsIcon fontSize={size} sx={{ color: '#A91D22' }} />;
  }

  // Postgres / StarRocks (both use postgres-compatible icons)
  if (
    normalized.includes('postgres') ||
    normalized.includes('postgresql') ||
    normalized.includes('psql') ||
    normalized.includes('starrocks')
  ) {
    return <StorageIcon fontSize={size} sx={{ color: '#336791' }} />;
  }

  // Generic cloud / warehouse
  if (normalized.includes('cloud') || normalized.includes('warehouse')) {
    return <CloudIcon fontSize={size} sx={{ color: '#757575' }} />;
  }

  // Default
  return <StorageIcon fontSize={size} sx={{ color: '#9e9e9e' }} />;
};

export default DialectIcon;
