import { Box, Typography } from '@mui/material';
import { useParams } from 'react-router-dom';
import { ConsoleLayout } from '../../layout/ConsoleLayout';
import { ConsoleBreadcrumbs } from '../../layout/ConsoleBreadcrumbs';
import { ETLRunTable, ETLRunDetail } from '../../components/etl';

export enum ETLViewMode {
  List = 'list',
  Detail = 'detail',
}

export function ETLRunsPage() {
  const { runId } = useParams<{ runId?: string }>();
  const mode = runId ? ETLViewMode.Detail : ETLViewMode.List;

  return (
    <ConsoleLayout>
      {mode === ETLViewMode.List && (
        <>
          <ConsoleBreadcrumbs
            items={[
              { label: 'ETL & Execution', href: '/console/etl' },
              { label: 'ETL Runs' },
            ]}
          />
          <ETLRunTable onRowClick={(id) => (window.location.href = `/console/etl/runs/${id}`)} />
        </>
      )}

      {mode === ETLViewMode.Detail && runId && (
        <>
          <ConsoleBreadcrumbs
            items={[
              { label: 'ETL & Execution', href: '/console/etl' },
              { label: 'ETL Runs', href: '/console/etl/runs' },
              { label: runId },
            ]}
          />
          <ETLRunDetail id={runId} />
        </>
      )}
    </ConsoleLayout>
  );
}
