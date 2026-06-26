// React import removed (automatic JSX runtime in use)
import LazyReactFlow, { ReactFlowFallback, LazyReactFlowSubcomponents } from './ReactFlowLoader';
import { useHierarchicalLayout } from '../hooks/useHierarchicalLayout';
import { SchemaContainerNode, TableContainerNode, ColumnNode } from './nodes/ContainerNodes';

const nodeTypes = {
  schemaContainer: SchemaContainerNode,
  tableContainer: TableContainerNode,
  column: ColumnNode,
};

interface EnhancedDualLineageViewerProps {
  data: { nodes: any[]; edges: any[]; layout?: any };
}

const EnhancedDualLineageViewer: React.FC<EnhancedDualLineageViewerProps> = ({ data }) => {
  const { nodes, edges, layout } = data;

  const { nodes: layoutedNodes, edges: layoutedEdges } = useHierarchicalLayout(nodes, edges, layout);

  return (
    <div className="diagram-container">
      <LazyReactFlow
        nodes={layoutedNodes}
        edges={layoutedEdges}
        nodeTypes={nodeTypes}
        fitView
      />
      {/* Lazy-load and render the subcomponents inside a Suspense boundary */}
      <ReactFlowFallback>
        <LazyReactFlowSubcomponents>
          {({ Background, Controls, MiniMap }: any) => (
            <>
              <Background />
              <Controls />
              <MiniMap />
            </>
          )}
        </LazyReactFlowSubcomponents>
      </ReactFlowFallback>
    </div>
  );
};

export default EnhancedDualLineageViewer;
