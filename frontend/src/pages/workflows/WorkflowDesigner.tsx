import React, { useState } from 'react';
import { devError } from '../../utils/devLogger';
import { useMutation, useQuery, gql } from '@apollo/client';
import ActionButton from '../../components/ui/ActionButton';
import './WorkflowDesigner.css';
import { useNotification } from '../../hooks/useNotification';

// ============================================================================
// GRAPHQL OPERATIONS
// ============================================================================

const CREATE_WORKFLOW_RULE = gql`
  mutation CreateWorkflowRule(
    $tenantId: uuid!
    $workflowName: String!
    $stepName: String!
    $stepOrder: Int!
    $conditionJson: jsonb!
    $actionOnSuccess: String!
    $actionOnFailure: String!
    $errorMessage: String!
  ) {
    insert_workflow_rules_one(
      object: {
        tenant_id: $tenantId
        workflow_name: $workflowName
        step_name: $stepName
        step_order: $stepOrder
        condition_json: $conditionJson
        action_on_success: $actionOnSuccess
        action_on_failure: $actionOnFailure
        error_message: $errorMessage
        is_active: true
        created_by: "user-id"
      }
    ) {
      id
      workflow_name
      step_name
    }
  }
`;

const GET_WORKFLOW_RULES = gql`
  query GetWorkflowRules($tenantId: uuid!, $workflowName: String!) {
    workflow_rules(
      where: {
        tenant_id: { _eq: $tenantId }
        workflow_name: { _eq: $workflowName }
      }
      order_by: { step_order: asc }
    ) {
      id
      workflow_name
      step_name
      step_order
      condition_json
      action_on_success
      action_on_failure
      error_message
      is_active
    }
  }
`;

// ============================================================================
// TYPES
// ============================================================================

interface Condition {
  field: string;
  operator: string;
  value: any;
}

interface _WorkflowStep {
  stepName: string;
  stepOrder: number;
  condition: Condition;
  actionOnSuccess: string;
  actionOnFailure: string;
  errorMessage: string;
}

interface WorkflowDesignerProps {
  tenantId: string;
  workflowName: string;
  onRuleCreated?: (ruleId: string) => void;
}

// ============================================================================
// OPERATOR SELECT COMPONENT
// ============================================================================

interface OperatorSelectProps {
  value: string;
  onChange: (op: string) => void;
  fieldType?: string;
}

const OperatorSelect: React.FC<OperatorSelectProps> = ({ value, onChange, fieldType: _fieldType }) => {
  return (
    <select
      id="operator-select"
      aria-label="Select comparison operator"
      title="Select comparison operator"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="operator-select"
    >
      <option value="=">=</option>
      <option value="!=">!=</option>
      <option value=">=">{'>='}</option>
      <option value=">">{'>'}</option>
      <option value="<=">{' <='}</option>
      <option value="<">{'<'}</option>
      <option value="contains">contains</option>
      <option value="is_null">is_null</option>
      <option value="not_null">not_null</option>
    </select>
  );
};

// ============================================================================
// MAIN WORKFLOW DESIGNER COMPONENT
// ============================================================================

export const WorkflowDesigner: React.FC<WorkflowDesignerProps> = ({
  tenantId,
  workflowName,
  onRuleCreated,
}) => {
  const [stepName, setStepName] = useState('');
  const [stepOrder, setStepOrder] = useState(1);
  const [field, setField] = useState('order_total');
  const [operator, setOperator] = useState('>=');
  const [conditionValue, setConditionValue] = useState('');
  const [actionOnSuccess, setActionOnSuccess] = useState('route:order_approved.queue');
  const [actionOnFailure, setActionOnFailure] = useState('notify:manager');
  const [errorMessage, setErrorMessage] = useState('');

  const [createRule] = useMutation(CREATE_WORKFLOW_RULE);
  const { data: rulesData, loading: rulesLoading } = useQuery(GET_WORKFLOW_RULES, {
    variables: { tenantId, workflowName },
  });

  const handleCreateRule = async () => {
    const notification = useNotification();
    if (!stepName.trim() || !field || !errorMessage.trim()) {
      notification.error('Please fill in all required fields');
      return;
    }

    try {
      const condition = {
        and: [
          {
            field,
            operator,
            value: operator === 'is_null' || operator === 'not_null' ? null : conditionValue,
          },
        ],
      };

      const result = await createRule({
        variables: {
          tenantId,
          workflowName,
          stepName,
          stepOrder,
          conditionJson: condition,
          actionOnSuccess,
          actionOnFailure,
          errorMessage,
        },
        refetchQueries: [{ query: GET_WORKFLOW_RULES, variables: { tenantId, workflowName } }],
      });

      if (result.data?.insert_workflow_rules_one?.id) {
        notification.success(`Workflow step "${stepName}" created successfully!`);
        if (onRuleCreated) {
          onRuleCreated(result.data.insert_workflow_rules_one.id);
        }
        // Reset form
        setStepName('');
        setStepOrder((stepOrder) => stepOrder + 1);
        setErrorMessage('');
        setConditionValue('');
      }
    } catch (error) {
      devError('Error creating rule:', error);
      notification.error(`Failed to create rule: ${error}`);
    }
  };

  const rules = rulesData?.workflow_rules || [];

  return (
    <div className="workflow-designer">
      <div className="designer-header">
        <h2>⚙️ Workday-Style Workflow Designer</h2>
        <p>Create low-code business rules and workflows</p>
      </div>

      <div className="designer-container">
        {/* Rule Configuration Panel */}
        <div className="rule-config-panel">
          <h3>Create Workflow Step</h3>

          <div className="form-group">
            <label htmlFor="stepNameInput">Step Name *</label>
            <input
              id="stepNameInput"
              type="text"
              value={stepName}
              onChange={(e) => setStepName(e.target.value)}
              placeholder="e.g., ApproveOrder"
            />
          </div>

          <div className="form-group">
            <label htmlFor="stepOrderInput">Step Order *</label>
            <input
              id="stepOrderInput"
              type="number"
              value={stepOrder}
              onChange={(e) => setStepOrder(parseInt(e.target.value))}
              min="1"
            />
          </div>

          <div className="condition-builder">
            <h4>Condition</h4>

            <div className="form-group">
              <label htmlFor="fieldSelect">Field *</label>
              <select
                id="fieldSelect"
                title="Select the field to check"
                value={field}
                onChange={(e) => setField(e.target.value)}
              >
                <option value="order_total">Order Total</option>
                <option value="hire_date">Hire Date</option>
                <option value="stock_change">Stock Change</option>
                <option value="status">Status</option>
                <option value="amount">Amount</option>
                <option value="customer_id">Customer ID</option>
              </select>
            </div>

            <div className="form-group">
              <label htmlFor="operatorSelect">Operator *</label>
              <OperatorSelect
                value={operator}
                onChange={setOperator}
                fieldType="numeric"
              />
            </div>

            {operator !== 'is_null' && operator !== 'not_null' && (
              <div className="form-group">
                <label htmlFor="conditionValueInput">Value *</label>
                <input
                  id="conditionValueInput"
                  type="text"
                  value={conditionValue}
                  onChange={(e) => setConditionValue(e.target.value)}
                  placeholder="Enter value to compare"
                />
              </div>
            )}
          </div>

          <div className="actions-builder">
            <h4>Actions</h4>

            <div className="form-group">
              <label htmlFor="successActionInput">On Success</label>
              <input
                id="successActionInput"
                type="text"
                value={actionOnSuccess}
                onChange={(e) => setActionOnSuccess(e.target.value)}
                placeholder="e.g., route:queue_name"
              />
              <small>Format: route:queue_name or notify:target</small>
            </div>

            <div className="form-group">
              <label htmlFor="failureActionInput">On Failure</label>
              <input
                id="failureActionInput"
                type="text"
                value={actionOnFailure}
                onChange={(e) => setActionOnFailure(e.target.value)}
                placeholder="e.g., notify:manager"
              />
            </div>

            <div className="form-group">
              <label htmlFor="errorMessageInput">Error Message *</label>
              <textarea
                id="errorMessageInput"
                value={errorMessage}
                onChange={(e) => setErrorMessage(e.target.value)}
                placeholder="User-friendly error message"
                rows={3}
              />
            </div>
          </div>

          <ActionButton variant="primary" onClick={handleCreateRule}>
            ✚ Create Workflow Step
          </ActionButton>
        </div>

        {/* Rules List Panel */}
        <div className="rules-list-panel">
          <h3>Workflow Steps ({rules.length})</h3>

          {rulesLoading ? (
            <p className="loading">Loading workflow steps...</p>
          ) : rules.length === 0 ? (
            <p className="empty-state">No workflow steps created yet</p>
          ) : (
            <div className="rules-list">
              {rules.map((rule: any, _index: number) => (
                <div key={rule.id} className="rule-card">
                  <div className="rule-header">
                    <span className="rule-order">Step {rule.step_order}</span>
                    <h4>{rule.step_name}</h4>
                    <span className={`status ${rule.is_active ? 'active' : 'inactive'}`}>
                      {rule.is_active ? '✓ Active' : 'Inactive'}
                    </span>
                  </div>

                  <div className="rule-content">
                    <div className="rule-section">
                      <strong>Condition:</strong>
                      <code>{JSON.stringify(rule.condition_json, null, 2)}</code>
                    </div>

                    <div className="rule-section">
                      <strong>On Success:</strong>
                      <code>{rule.action_on_success || 'No action'}</code>
                    </div>

                    <div className="rule-section">
                      <strong>On Failure:</strong>
                      <code>{rule.action_on_failure || 'No action'}</code>
                    </div>

                    <div className="rule-section">
                      <strong>Error Message:</strong>
                      <p>{rule.error_message}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default WorkflowDesigner;
