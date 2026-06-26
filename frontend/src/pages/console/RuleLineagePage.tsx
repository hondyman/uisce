import { useParams } from 'react-router-dom';
import { ConsoleLayout } from '../../layout/ConsoleLayout';
import { ConsoleBreadcrumbs } from '../../layout/ConsoleBreadcrumbs';
import { RuleLineageTable } from '../../components/lineage';
import { TrendChart } from '../../components/charts/TrendChart';
import { Box } from '@mui/material';
import { useRuleLineage } from '../../api/ruleLineage';

export function RuleLineagePage() {
  const { ruleId = 'MAX_ISSUER_5' } = useParams<{ ruleId?: string }>();
  const { data } = useRuleLineage(ruleId);

  return (
    <ConsoleLayout>
      <ConsoleBreadcrumbs
        items={[
          { label: 'Compliance', href: '/console/compliance' },
          { label: 'Rules', href: '/console/compliance/rules' },
          { label: `Rule: ${ruleId}`, href: `/console/compliance/rules/${ruleId}` },
          { label: 'Lineage' },
        ]}
      />

      {data && data.length > 0 && (
        <Box sx={{ mb: 3, p: 2, backgroundColor: '#f5f5f5', borderRadius: 1 }}>
          <TrendChart
            data={data}
            metricKey="metric_value"
            threshold={data[0].threshold_value}
          />
        </Box>
      )}

      <RuleLineageTable ruleId={ruleId} />
    </ConsoleLayout>
  );
}
