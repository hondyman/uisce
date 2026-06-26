import { useParams } from 'react-router-dom';
import { ConsoleLayout } from '../../layout/ConsoleLayout';
import { ConsoleBreadcrumbs } from '../../layout/ConsoleBreadcrumbs';
import { ScenarioLineageTable } from '../../components/lineage';
import { TrendChart } from '../../components/charts/TrendChart';
import { Box } from '@mui/material';
import { useScenarioLineage } from '../../api/scenarioLineage';

export function ScenarioLineagePage() {
  const { scenarioId = 'equity-shock-20' } = useParams<{ scenarioId?: string }>();
  const { data } = useScenarioLineage(scenarioId);

  return (
    <ConsoleLayout>
      <ConsoleBreadcrumbs
        items={[
          { label: 'Risk', href: '/console/risk' },
          { label: 'Scenarios', href: '/console/risk/scenarios' },
          { label: `Scenario: ${scenarioId}`, href: `/console/risk/scenarios/${scenarioId}` },
          { label: 'Lineage' },
        ]}
      />

      {data && data.length > 0 && (
        <Box sx={{ mb: 3, p: 2, backgroundColor: '#f5f5f5', borderRadius: 1 }}>
          <TrendChart data={data} metricKey="pnl" />
        </Box>
      )}

      <ScenarioLineageTable scenarioId={scenarioId} />
    </ConsoleLayout>
  );
}
