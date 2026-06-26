import { useMemo } from 'react';
import type { FC } from 'react';
import { Card, CardContent, Tooltip, Chip, Stack, Box, Typography } from '@mui/material';
import ActionButton from '../ui/ActionButton';
import SVGIcon from './SVGIcon';
import './RelationshipPathVisualizer.module.css';

interface PathHop {
  order: number;
  entity_id: string;
  entity_name: string;
  semantic_term_name?: string;
  link_type: string;
  cardinality: string;
  fk_constraint?: string;
  source_column?: string;
  target_column?: string;
  foreign_key_path?: string;
}

interface RelationshipPath {
  path_id: string;
  source_entity_id: string;
  target_entity_id: string;
  hierarchy_depth: number;
  hops: PathHop[];
  total_confidence: number;
  total_cardinality: string;
  entities?: Array<{
    order: number;
    entity_id: string;
    entity_name: string;
    semantic_term_name?: string;
    is_primary_key?: boolean;
    column_name?: string;
  }>;
}

interface RelationshipPathVisualizerProps {
  path: RelationshipPath;
  onApply: () => void;
}

const RelationshipPathVisualizer: FC<RelationshipPathVisualizerProps> = ({
  path,
  onApply,
}) => {
  // Render link type badge
  const renderLinkTypeBadge = (linkType: string) => {
    const typeColors: { [key: string]: string } = {
      DIRECT_FK: '#2196F3',
      SEMANTIC: '#9C27B0',
      MULTI_HOP: '#FF9800',
      FK_SCAN: '#4CAF50',
      PATTERN: '#9E9E9E',
    };

    return (
      <Chip
        label={linkType}
        size="small"
        sx={{
          backgroundColor: typeColors[linkType] || '#d9d9d9',
          color: '#fff',
          fontSize: '11px',
        }}
      />
    );
  };

  // Render cardinality badge
  const renderCardinalityBadge = (cardinality: string) => {
    return (
      <Tooltip title={`Cardinality: ${cardinality}`}>
        <Chip
          label={cardinality}
          size="small"
          sx={{
            backgroundColor: '#faad14',
            color: '#fff',
            fontSize: '11px',
          }}
        />
      </Tooltip>
    );
  };

  // Render confidence
  const renderConfidence = () => {
    let color = '#52c41a'; // Green
    if (path.total_confidence < 0.7) {
      color = '#faad14'; // Orange
    }
    if (path.total_confidence < 0.5) {
      color = '#f5222d'; // Red
    }

    return (
      <Tooltip title={`Total Confidence: ${(path.total_confidence * 100).toFixed(0)}%`}>
        <Chip
          label={`${(path.total_confidence * 100).toFixed(0)}%`}
          size="small"
          sx={{
            backgroundColor: color,
            color: '#fff',
            fontSize: '11px',
          }}
        />
      </Tooltip>
    );
  };

  const pathDescription = useMemo(() => {
    if (!path.hops || path.hops.length === 0) {
      return 'No path information available';
    }

    return path.hops.map((hop) => `${hop.entity_name} (${hop.cardinality})`).join(' → ');
  }, [path]);

  return (
    <Card className="relationship-path-card">
      <CardContent>
        <Box>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <SVGIcon name="grid_view" className="inline-block" ariaLabel="path" /> 
              <Typography variant="h6">Multi-Hop Path (Depth: {path.hierarchy_depth})</Typography>
            </Box>
            <Stack direction="row" spacing={1}>
              {renderConfidence()}
              <ActionButton variant="primary" size="sm" onClick={onApply}>
                <SVGIcon name="check_circle" className="inline-block mr-2 h-4 w-4" ariaLabel="apply" />
                Apply
              </ActionButton>
            </Stack>
          </Box>

          <div className="path-visualization">
            <div className="path-summary">
              <p className="path-description">{pathDescription}</p>
            </div>

            {path.hops && path.hops.length > 0 && (
              <div className="path-hops">
                {path.hops.map((hop, index) => (
                  <div key={`${hop.entity_id}-${index}`} className="hop-item">
                    <div className="hop-content">
                      <div className="hop-entity">
                        <div className="entity-name">{hop.entity_name}</div>
                        {hop.semantic_term_name && (
                          <div className="semantic-name">({hop.semantic_term_name})</div>
                        )}
                      </div>

                      <div className="hop-details">
                        <div className="detail-badges">
                          {renderLinkTypeBadge(hop.link_type)}
                          {renderCardinalityBadge(hop.cardinality)}
                        </div>

                        {hop.foreign_key_path && (
                          <div className="fk-path">
                            <code>{hop.foreign_key_path}</code>
                          </div>
                        )}

                        {(hop.source_column || hop.target_column) && (
                          <div className="columns">
                            {hop.source_column && (
                              <span className="column">{hop.source_column}</span>
                            )}
                            {hop.source_column && hop.target_column && <span>→</span>}
                            {hop.target_column && (
                              <span className="column">{hop.target_column}</span>
                            )}
                          </div>
                        )}
                      </div>
                    </div>

                    {index < path.hops.length - 1 && (
                      <div className="hop-arrow">
                        <SVGIcon name="arrow_forward" className="inline-block" ariaLabel="to" />
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}

            <div className="path-metadata">
              <div className="metadata-row">
                <span className="label">Path ID:</span>
                <span className="value mono">{path.path_id.substring(0, 8)}...</span>
              </div>
              <div className="metadata-row">
                <span className="label">Depth:</span>
                <span className="value">{path.hierarchy_depth}</span>
              </div>
              <div className="metadata-row">
                <span className="label">Total Cardinality:</span>
                <span className="value">{path.total_cardinality}</span>
              </div>
              <div className="metadata-row">
                <span className="label">Confidence:</span>
                <span className="value">{(path.total_confidence * 100).toFixed(1)}%</span>
              </div>
            </div>
          </div>
        </Box>
      </CardContent>
    </Card>
  );
};

export default RelationshipPathVisualizer;
