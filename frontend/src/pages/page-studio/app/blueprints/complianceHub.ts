import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';

/**
 * Compliance & Exception Governance Page Studio Blueprint:
 * 
 * Provides:
 * 1. Gold-Copy Core Rule Library Explorer (94 Rules across 5 packs)
 * 2. Tenant Rule Activation Matrix & Threshold Customizer (CRUD / Repinning)
 * 3. Pre-Trade Real-Time Decision Blotter (Explainability & RFC 8785 hashes)
 * 4. Post-Trade Surveillance & Exception Findings Queue (Statutory deadlines)
 * 5. Regulatory Change Triage & Steward Review Queue
 * 6. Compliance Calendar (Statutory filings & Dealing cutoffs)
 * 7. Limit Utilization & Headroom Dashboard (Proximity gauges & Capacity buffers)
 */

type Spec = Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>;
const layout = (root: string, spec: Spec): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

export function complianceHubBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const reg = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}): string => {
    components[id] = { id, type, props, ...extra };
    return id;
  };

  // Header
  reg('hdr', 'PageHeader', {
    icon: 'security',
    title: 'Compliance & Exception Governance',
    subtitle: 'Unified console: 94-rule core library, tenant activation matrix, pre-trade blotter, and surveillance finding queues',
  }, { style: { flex: '1 1 320px' } });

  // Tabs
  const tabs: PageTab[] = [
    {
      id: 'library',
      label: 'Core Rule Library (94 Rules)',
      layout: layout('lib_root', {
        lib_root: { type: 'Column', children: ['lib_widget'], style: { gap: '16px' } },
      }),
    },
    {
      id: 'matrix',
      label: 'Tenant Rule Matrix',
      layout: layout('matrix_root', {
        matrix_root: { type: 'Column', children: ['matrix_widget'], style: { gap: '16px' } },
      }),
    },
    {
      id: 'blotter',
      label: 'Pre-Trade Blotter',
      layout: layout('blotter_root', {
        blotter_root: { type: 'Column', children: ['blotter_widget'], style: { gap: '16px' } },
      }),
    },
    {
      id: 'surveillance',
      label: 'Surveillance & Exceptions',
      layout: layout('surv_root', {
        surv_root: { type: 'Column', children: ['surv_widget'], style: { gap: '16px' } },
      }),
    },
    {
      id: 'regulatory',
      label: 'Regulatory Triage',
      layout: layout('reg_root', {
        reg_root: { type: 'Column', children: ['reg_widget'], style: { gap: '16px' } },
      }),
    },
    {
      id: 'calendar',
      label: 'Compliance Calendar',
      layout: layout('cal_root', {
        cal_root: { type: 'Column', children: ['cal_widget'], style: { gap: '16px' } },
      }),
    },
    {
      id: 'limits',
      label: 'Limit Utilization',
      layout: layout('limits_root', {
        limits_root: { type: 'Column', children: ['limits_widget'], style: { gap: '16px' } },
      }),
    },
  ];

  reg('lib_widget', 'DomainComponent', {
    component: 'compliance.RuleLibraryExplorer',
    inputs: {},
  });

  reg('matrix_widget', 'DomainComponent', {
    component: 'compliance.RuleActivationMatrix',
    inputs: {},
  });

  reg('blotter_widget', 'DomainComponent', {
    component: 'compliance.DecisionBlotter',
    inputs: {},
  });

  reg('surv_widget', 'DomainComponent', {
    component: 'compliance.SurveillanceQueue',
    inputs: {},
  });

  reg('reg_widget', 'DomainComponent', {
    component: 'compliance.RegulatoryQueue',
    inputs: {},
  });

  reg('cal_widget', 'DomainComponent', {
    component: 'compliance.Calendar',
    inputs: {},
  });

  reg('limits_widget', 'DomainComponent', {
    component: 'compliance.LimitDashboard',
    inputs: {},
  });

  return {
    name: 'Compliance & Exception Governance',
    slug: 'compliance-hub',
    description: 'Unified compliance console: 94 core rules, tenant overrides, pre-trade blotter, post-trade exception surveillance, regulatory change queue, compliance calendar, and limit utilization.',
    icon: 'security',
    category: 'governance',
    tabs,
    layout: layout('root', {
      root: { type: 'Column', children: ['hdr'], style: { gap: '16px' } },
    }),
    components,
    variables: {},
    queries: {},
  };
}
