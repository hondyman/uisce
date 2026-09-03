import { ComponentType } from 'react';
import { NodeProps, NodeTypes } from 'reactflow';

// Generic fallback node used when a graph node's `type` has no registered renderer.
const GenericNode: ComponentType<NodeProps> = ({ data, selected }) => {
  const label = (data && (data.label || data.name)) || 'Node';
  return (
    <div
      style={{
        padding: '10px 16px',
        borderRadius: 8,
        border: `2px solid ${selected ? '#3b82f6' : '#cbd5e1'}`,
        background: '#ffffff',
        boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
        fontSize: 13,
        fontWeight: 500,
        color: '#1e293b',
        minWidth: 120,
        textAlign: 'center',
      }}
    >
      {label}
    </div>
  );
};

const registry = new Map<string, ComponentType<NodeProps>>();

export function registerNodeType(type: string, component: ComponentType<NodeProps>): void {
  registry.set(type, component);
}

export function getRegisteredNodeType(type: string): ComponentType<NodeProps> | undefined {
  return registry.get(type);
}

/**
 * Builds the `nodeTypes` map handed to ReactFlow: globally registered types,
 * overlaid with any per-view overrides, and a `default` fallback renderer.
 */
export function getNodeTypes(overrides?: Record<string, ComponentType<NodeProps>>): NodeTypes {
  const merged: NodeTypes = { default: GenericNode };
  for (const [type, component] of registry.entries()) {
    merged[type] = component;
  }
  if (overrides) {
    for (const [type, component] of Object.entries(overrides)) {
      merged[type] = component;
    }
  }
  return merged;
}

export { GenericNode };
