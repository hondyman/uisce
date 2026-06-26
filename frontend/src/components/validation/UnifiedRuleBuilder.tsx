import { useState } from 'react';
import type { FC } from 'react';
import { ValidationRuleCreator } from '../ValidationRuleCreator';
import AdvancedRuleConfiguration from './AdvancedRuleConfiguration';
// Some named exports referenced previously are not present in the current file.
// Use any for cross-entity condition to avoid a hard type dependency in this refactor pass.
type CrossEntityCondition = any;
import type { ValidationRule as SharedValidationRule } from './types';

// Unified builder composes the refactor (stepper modal/inline) and the legacy advanced config
const UnifiedRuleBuilder: FC<{
  rules?: SharedValidationRule[];
  onRulesUpdate?: (rules: SharedValidationRule[]) => void;
  onCrossEntitySave?: (condition: CrossEntityCondition) => void;
  mode?: 'inline' | 'modal';
}> = ({ rules, onRulesUpdate, onCrossEntitySave, mode = 'inline' }) => {
  const [editingRule, setEditingRule] = useState<SharedValidationRule | null>(null);

  const handleSaveFromCreator = (rule: SharedValidationRule) => {
    const existing = rules || [];
    const ruleWithId: SharedValidationRule = {
      ...rule,
      id: rule.id || `rule_${Date.now()}`
    };

    let updated: SharedValidationRule[];
    if (editingRule) {
      // replace existing rule by id
      updated = existing.map(r => (r.id === ruleWithId.id ? ruleWithId : r));
    } else {
      // prepend new
      updated = [ruleWithId, ...existing];
    }

    onRulesUpdate?.(updated);
    // clear edit state
    setEditingRule(null);
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div>
        {/* Left: new/refactored creation UI (modal by default but show inline here) */}
        <div className="bg-white rounded-lg shadow p-4">
          <h3 className="text-lg font-semibold mb-2">Create / Edit Rule</h3>
          <ValidationRuleCreator
            isOpen={true}
            initialRule={editingRule || undefined}
            onClose={() => {
              /* no-op in embedded mode */
              setEditingRule(null);
            }}
            onSave={handleSaveFromCreator}
            availableEntities={["Employee","Department","Position","Organization","Location","Cost Center"]}
            displayMode={mode === 'modal' ? 'modal' : 'inline'}
          />
        </div>
      </div>

      <div>
        {/* Right: advanced configuration from legacy (dependencies & cross-entity) */}
        <div className="bg-white rounded-lg shadow p-4">
          <h3 className="text-lg font-semibold mb-2">Advanced Configuration</h3>
          <AdvancedRuleConfiguration
            // Adapt SharedValidationRule to the local AdvancedRuleConfiguration.ValidationRule shape
            rules={
              rules
                ? rules
                    .filter((r) => r.id)
                    .map((r) => ({
                      id: r.id!,
                      name: r.name ?? r.rule_name ?? 'Unnamed Rule',
                      entity: r.entity ?? r.target_entity ?? r.target_entities?.[0] ?? 'Unknown',
                      description: r.description ?? '',
                      severity: ((): 'info' | 'warning' | 'error' => {
                        const val: unknown = r.severity;
                        const isValid = typeof val === 'string' && (val === 'info' || val === 'warning' || val === 'error');
                        return isValid ? (val as 'info' | 'warning' | 'error') : 'info';
                      })(),
                      dependent_rule_ids: r.dependent_rule_ids ?? [],
                    }))
                : undefined
            }
            onRulesUpdate={onRulesUpdate}
            onCrossEntitySave={onCrossEntitySave}
          />
        </div>
      </div>
    </div>
  );
};

export default UnifiedRuleBuilder;
