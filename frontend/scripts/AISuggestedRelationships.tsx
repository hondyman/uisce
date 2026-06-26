// frontend/src/components/ai/AISuggestedRelationships.tsx

import React, { useState, useEffect } from 'react';
import { Card, List, Tag, Spin, Alert, Tooltip } from 'antd';
import { BrainCircuit, Link } from 'lucide-react';

// ============================================================================
// TYPES
// ============================================================================

interface AISuggestedRelationship {
  targetEntityId: string;
  targetEntityName: string;
  reason: string;
  sharedTerms: string[];
  confidenceScore: number;
}

interface AISuggestedRelationshipsProps {
  entityId: string;
  tenantId: string;
  datasourceId: string;
}

// ============================================================================
// MOCK API - Replace with actual API call
// ============================================================================

const fetchSuggestedRelationships = async (
  entityId: string
): Promise<AISuggestedRelationship[]> => {
  console.log(`AI: Discovering relationships for entity ${entityId}...`);
  // This is a mock. In a real app, this would be a GraphQL query or REST call
  // to an endpoint that uses the RelationshipDiscoverer service.
  await new Promise(resolve => setTimeout(resolve, 1500)); // Simulate network delay

  if (entityId === 'client_investor') {
    return [
      {
        targetEntityId: 'risk_profile',
        targetEntityName: 'Risk Profile',
        reason: 'Shares common business concepts',
        sharedTerms: ['Customer', 'Portfolio'],
        confidenceScore: 0.85,
      },
      {
        targetEntityId: 'trades_fact',
        targetEntityName: 'Trades Fact Table',
        reason: 'Shares common business concepts',
        sharedTerms: ['Investment', 'Portfolio'],
        confidenceScore: 0.65,
      },
      {
        targetEntityId: 'market_data',
        targetEntityName: 'Market Data',
        reason: 'Shares common business concepts',
        sharedTerms: ['Investment'],
        confidenceScore: 0.40,
      },
    ];
  }
  return [];
};

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const AISuggestedRelationships: React.FC<AISuggestedRelationshipsProps> = ({
  entityId,
  tenantId,
  datasourceId,
}) => {
  const [suggestions, setSuggestions] = useState<AISuggestedRelationship[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadSuggestions = async () => {
      try {
        setLoading(true);
        const result = await fetchSuggestedRelationships(entityId);
        setSuggestions(result);
      } catch (e) {
        setError('Failed to load AI suggestions.');
        console.error(e);
      } finally {
        setLoading(false);
      }
    };

    loadSuggestions();
  }, [entityId, tenantId, datasourceId]);

  const getConfidenceColor = (score: number) => {
    if (score > 0.7) return 'green';
    if (score > 0.5) return 'blue';
    return 'gold';
  };

  if (loading) {
    return <Spin tip="AI is analyzing relationships..." />;
  }

  if (error) {
    return <Alert message={error} type="error" showIcon />;
  }

  if (suggestions.length === 0) {
    return <Alert message="No semantic relationships discovered by AI." type="info" showIcon />;
  }

  return (
    <Card
      title={
        <div className="flex items-center gap-2">
          <BrainCircuit className="text-purple-600" />
          <span>AI-Discovered Relationships</span>
        </div>
      }
    >
      <List
        itemLayout="horizontal"
        dataSource={suggestions}
        renderItem={(item) => (
          <List.Item
            actions={[<Tag color={getConfidenceColor(item.confidenceScore)}>{`${Math.round(item.confidenceScore * 100)}% Match`}</Tag>]}
          >
            <List.Item.Meta
              avatar={<Link className="text-gray-500 mt-1" />}
              title={<a href={`/entities/${item.targetEntityId}`}>{item.targetEntityName}</a>}
              description={
                <Tooltip title={`Reason: ${item.reason}`}>
                  <span>Shared concepts: {item.sharedTerms.join(', ')}</span>
                </Tooltip>
              }
            />
          </List.Item>
        )}
      />
    </Card>
  );
};