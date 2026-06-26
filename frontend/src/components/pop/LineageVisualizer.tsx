import React, { useEffect, useRef, useState } from 'react';
import CytoscapeComponent from 'react-cytoscapejs';
import cytoscape from 'cytoscape';
import dagre from 'cytoscape-dagre';
import { Card, CardContent, Stack, Tooltip, CircularProgress, Box } from '@mui/material';
import ActionButton from '../ui/ActionButton';
import styles from './LineageVisualizer.module.css';
import { useNotification } from '../../hooks/useNotification';

// Register dagre layout
cytoscape.use(dagre);

interface LineageNode {
  data: {
    id: string;
    label: string;
    type: string;
  };
}

interface LineageEdge {
  data: {
    source: string;
    target: string;
    type: string;
  };
}

interface LineageData {
  elements: (LineageNode | LineageEdge)[];
  lineage: {
    node_id: string;
    node_type: string;
    name: string;
    description: string;
    source_tables: string[];
    downstream_consumers: string[];
    upstream_transformations: string[];
    data_quality_checks: string[];
  };
}

interface LineageVisualizerProps {
  nodeId: string;
  onNodeClick?: (nodeId: string) => void;
}

const LineageVisualizer: React.FC<LineageVisualizerProps> = ({
  nodeId,
  onNodeClick
}) => {
  const notification = useNotification();
  const cyRef = useRef<ReturnType<typeof cytoscape> | null>(null);
  const [loading, setLoading] = useState(false);
  const [lineageData, setLineageData] = useState<LineageData | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchLineageData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`/api/lineage/${nodeId}/graph`);
      if (!response.ok) {
        throw new Error('Failed to fetch lineage data');
      }
      const data = await response.json();
      setLineageData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      notification.error('Failed to load lineage visualization');
    } finally {
      setLoading(false);
    }
  }, [nodeId, notification]);

  useEffect(() => {
    if (nodeId) {
      fetchLineageData();
    }
  }, [nodeId, fetchLineageData]);

  const handleCyInit = (cy: ReturnType<typeof cytoscape>) => {
    cyRef.current = cy;

    // Configure layout
    const layout = cy.layout({
      name: 'dagre',
      rankDir: 'TB',
      nodeSep: 50,
      edgeSep: 10,
      rankSep: 100,
    });

    layout.run();

    // Add event listeners
    cy.on('tap', 'node', (event: any) => {
      const node = event.target;
      const nodeId = node.data('id');
      if (onNodeClick) {
        onNodeClick(nodeId);
      }
    });

    // Style nodes based on type
    cy.style()
      .selector('node[type="source_table"]')
      .style({
        'background-color': '#52c41a',
        'label': 'data(label)',
        'text-valign': 'center',
        'text-halign': 'center',
        'font-size': '10px',
        'width': '80px',
        'height': '40px',
        'shape': 'round-rectangle'
      })
      .selector('node[type="consumer"]')
      .style({
        'background-color': '#1890ff',
        'label': 'data(label)',
        'text-valign': 'center',
        'text-halign': 'center',
        'font-size': '10px',
        'width': '80px',
        'height': '40px',
        'shape': 'round-rectangle'
      })
      .selector('node')
      .style({
        'background-color': '#722ed1',
        'label': 'data(label)',
        'text-valign': 'center',
        'text-halign': 'center',
        'font-size': '10px',
        'width': '100px',
        'height': '50px',
        'shape': 'round-rectangle',
        'border-width': '2px',
        'border-color': '#d3adf7'
      })
      .selector('edge')
      .style({
        'width': '2px',
        'line-color': '#d9d9d9',
        'target-arrow-color': '#d9d9d9',
        'target-arrow-shape': 'triangle',
        'curve-style': 'bezier'
      })
      .selector('edge[type="transformation"]')
      .style({
        'line-color': '#52c41a',
        'target-arrow-color': '#52c41a'
      })
      .selector('edge[type="consumption"]')
      .style({
        'line-color': '#1890ff',
        'target-arrow-color': '#1890ff'
      })
      .update();
  };

  const handleZoomIn = () => {
    if (cyRef.current) {
      cyRef.current.zoom(cyRef.current.zoom() * 1.2);
    }
  };

  const handleZoomOut = () => {
    if (cyRef.current) {
      cyRef.current.zoom(cyRef.current.zoom() * 0.8);
    }
  };

  const handleFit = () => {
    if (cyRef.current) {
      cyRef.current.fit();
    }
  };

  const handleRefresh = () => {
    fetchLineageData();
  };

  if (loading) {
    return (
      <Card>
        <CardContent>
          <Box className={styles.loadingContainer} sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
            <CircularProgress />
            <p>Loading lineage data...</p>
          </Box>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent>
          <Box className={styles.errorContainer} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <p>Error: {error}</p>
            <ActionButton size="sm" variant="primary" onClick={fetchLineageData}>Retry</ActionButton>
          </Box>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent>
        <Stack spacing={2}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <h3>{`Lineage: ${lineageData?.lineage.name || nodeId}`}</h3>
            <Stack direction="row" spacing={1}>
              <Tooltip title="Zoom In">
                <ActionButton size="sm" variant="ghost" iconName="zoom_in" onClick={handleZoomIn} />
              </Tooltip>
              <Tooltip title="Zoom Out">
                <ActionButton size="sm" variant="ghost" iconName="zoom_out" onClick={handleZoomOut} />
              </Tooltip>
              <Tooltip title="Fit to Screen">
                <ActionButton size="sm" variant="ghost" iconName="fit_screen" onClick={handleFit} />
              </Tooltip>
              <Tooltip title="Refresh">
                <ActionButton size="sm" variant="ghost" iconName="refresh" onClick={handleRefresh} />
              </Tooltip>
            </Stack>
          </Box>

          <div className={styles.visualizerContainer}>
            {lineageData && (
              <CytoscapeComponent
                elements={lineageData.elements}
                style={{ width: '100%', height: '400px' }}
                cy={handleCyInit}
                layout={{ name: 'preset' }}
              />
            )}
          </div>

          {lineageData && (
            <div className={styles.lineageInfo}>
              <h4>Lineage Information</h4>
              <p><strong>Type:</strong> {lineageData.lineage.node_type}</p>
              <p><strong>Description:</strong> {lineageData.lineage.description}</p>

              {lineageData.lineage.source_tables.length > 0 && (
                <div>
                  <strong>Source Tables:</strong>
                  <ul>
                    {lineageData.lineage.source_tables.map(table => (
                      <li key={table}>{table}</li>
                    ))}
                  </ul>
                </div>
              )}

              {lineageData.lineage.upstream_transformations.length > 0 && (
                <div>
                  <strong>Transformations:</strong>
                  <ul>
                    {lineageData.lineage.upstream_transformations.map(transform => (
                      <li key={transform}>{transform}</li>
                    ))}
                  </ul>
                </div>
              )}

              {lineageData.lineage.data_quality_checks.length > 0 && (
                <div>
                  <strong>Data Quality:</strong>
                  <ul>
                    {lineageData.lineage.data_quality_checks.map(check => (
                      <li key={check}>{check}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
};

export default LineageVisualizer;
