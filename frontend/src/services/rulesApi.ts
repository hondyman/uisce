export const rulesApi = {
  fetchRuleDiff: async (boId: string, ruleId: string, tenantId?: string, datasourceId?: string) => {
    return { diffs: [] };
  },
  getRulesForTerm: async (termId: string) => {
    return [];
  },
};