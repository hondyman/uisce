import { useParams } from 'react-router-dom';
import { ConsoleLayout } from '../../layout/ConsoleLayout';
import { ConsoleBreadcrumbs } from '../../layout/ConsoleBreadcrumbs';
import { WASMVersionTable } from '../../components/wasm';

export function WASMVersionsPage() {
  const { moduleName = 'risk-engine' } = useParams<{ moduleName?: string }>();

  return (
    <ConsoleLayout>
      <ConsoleBreadcrumbs
        items={[
          { label: 'ETL & Execution', href: '/console/etl' },
          { label: 'WASM Versions' },
        ]}
      />
      <WASMVersionTable moduleName={moduleName} />
    </ConsoleLayout>
  );
}
