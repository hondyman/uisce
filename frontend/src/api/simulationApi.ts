import { fetchAPI } from '@/api';

export interface SimulationScenario {
  id: string;
  tenantId: string;
  name: string;
  description: string;
  scenarioType: string;
  status: string;
  createdAt: string;
}

export interface SimulationDelta {
  scenarioId: string;
  boId: string;
  deltaType: string;
  changes: any;
}

export interface SimulationResult {
  id: string;
  runId: string;
  scenarioId: string;
  summary: any; // { navDelta: number, ... }
  complianceSummary: any; // { newIssues: [], ... }
  metrics: any[];
  createdAt: string;
}

export const simulationApi = {
  createScenario: async (scenario: Partial<SimulationScenario>) => {
    return fetchAPI<SimulationScenario>('/api/simulation/scenarios', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(scenario)
    });
  },

  listScenarios: async () => {
    return fetchAPI<SimulationScenario[]>('/api/simulation/scenarios');
  },

  getScenario: async (id: string) => {
    return fetchAPI<SimulationScenario>(`/api/simulation/scenarios/${id}`);
  },

  addDelta: async (scenarioId: string, delta: SimulationDelta) => {
    return fetchAPI<SimulationDelta>(`/api/simulation/scenarios/${scenarioId}/deltas`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(delta)
    });
  },

  getDeltas: async (scenarioId: string) => {
    return fetchAPI<SimulationDelta[]>(`/api/simulation/scenarios/${scenarioId}/deltas`);
  },

  runSimulation: async (scenarioId: string) => {
    return fetchAPI<SimulationResult>(`/api/simulation/scenarios/${scenarioId}/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    });
  },

  getLatestResult: async (scenarioId: string) => {
    return fetchAPI<SimulationResult>(`/api/simulation/scenarios/${scenarioId}/result`);
  },

  createChangeSet: async (scenarioId: string) => {
    return fetchAPI<{ changeset_id: string }>(`/api/simulation/scenarios/${scenarioId}/changeset`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    });
  }
};
