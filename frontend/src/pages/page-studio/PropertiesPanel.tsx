import React, { useMemo } from 'react';
import { Box, Typography, Paper, Divider } from '@mui/material';
import { CorePageDefinition, LayoutNode } from '../../types/pageStudio';
import ExpressionBuilder from '../../components/ExpressionBuilder/ExpressionBuilder';
import { ConditionGroup } from '../../components/ExpressionBuilder/AdvancedConditionBuilder';

interface PropertiesPanelProps {
  selectedId: string | null;
  draft: CorePageDefinition;
  setDraft: (updater: (prev: CorePageDefinition) => CorePageDefinition) => void;
  tenantId?: string;
}

function findNode(nodes: LayoutNode[], id: string): LayoutNode | null {
  for (const node of nodes) {
    if (node.id === id) return node;
    if (node.children) {
      const found = findNode(node.children, id);
      if (found) return found;
    }
  }
  return null;
}

function replaceNode(nodes: LayoutNode[], id: string, updater: (n: LayoutNode) => LayoutNode): LayoutNode[] {
  return nodes.map((node) => {
    if (node.id === id) return updater(node);
    if (node.children) return { ...node, children: replaceNode(node.children, id, updater) };
    return node;
  });
}

const PropertiesPanel: React.FC<PropertiesPanelProps> = ({ selectedId, draft, setDraft }) => {
  const selectedNode = useMemo(
    () => (selectedId ? findNode(draft.layout || [], selectedId) : null),
    [selectedId, draft.layout]
  );

  // BO field metadata isn't loaded in this panel; ExpressionBuilder falls
  // back to its own default field list when availableFields is empty.
  const availableFields = useMemo(() => [], []);

  if (!selectedNode) {
    return (
      <Paper elevation={0} sx={{ p: 2, height: '100%', bgcolor: 'background.paper' }}>
        <Typography variant="body2" color="text.secondary">
          Select a component to edit its properties
        </Typography>
      </Paper>
    );
  }

  const handleRuleChange = (conditionJson: ConditionGroup) => {
    setDraft((prev) => ({
      ...prev,
      layout: replaceNode(prev.layout || [], selectedNode.id, (n) => ({
        ...n,
        props: { ...n.props, fieldChangeRule: conditionJson },
      })),
    }));
  };

  return (
    <Paper elevation={0} sx={{ p: 2, height: '100%', bgcolor: 'background.paper', overflowY: 'auto' }}>
      <Typography variant="subtitle2" sx={{ mb: 2 }}>
        Properties: {selectedNode.componentId || 'Unknown'}
      </Typography>
      <Divider sx={{ mb: 2 }} />
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Component ID: {selectedNode.id}
      </Typography>

      <Divider sx={{ mb: 2 }} />
      <Typography variant="subtitle2" sx={{ mb: 1 }}>
        Field Change Rule
      </Typography>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
        Runs against the field_change trigger event whenever this component's value changes.
      </Typography>
      <ExpressionBuilder
        ruleName={`page-${draft.slug}-${selectedNode.id}`}
        targetEntity={selectedNode.componentId}
        availableFields={availableFields}
        onChange={handleRuleChange}
      />
    </Paper>
  );
};

export default PropertiesPanel;
