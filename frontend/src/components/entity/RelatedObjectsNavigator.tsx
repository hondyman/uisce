/**
 * RelatedObjectsNavigator Component
 * 
 * Displays business objects' links to/from relationships with dot-notation
 * traversal support for multi-level object graphs.
 */

import React, { useState, useCallback } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../ui/card';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Badge } from '../ui/badge';
import { Alert, AlertDescription } from '../ui/alert';
import { AlertCircle, ChevronRight, ArrowRight, ArrowLeft, Search } from 'lucide-react';
import type { SemanticModelMetadata } from '../../services/businessEntitySemanticService';
import './RelatedObjectsNavigator.css';

interface RelatedObjectsNavigatorProps {
  linksTo: SemanticModelMetadata[];
  linksFrom: SemanticModelMetadata[];
  isLoading: boolean;
  error?: Error | null;
  businessEntityName: string;
  onTraverse: (dotPath: string) => Promise<any>;
}

const RelatedObjectsNavigator: React.FC<RelatedObjectsNavigatorProps> = ({
  linksTo,
  linksFrom,
  isLoading,
  error,
  businessEntityName,
  onTraverse,
}) => {
  const [dotPathInput, setDotPathInput] = useState('');
  const [traversing, setTraversing] = useState(false);
  const [_traversalResult, _setTraversalResult] = useState<{ nodes: any[]; edges: any[] } | null>(null);

  const handleTraverse = useCallback(async () => {
    if (!dotPathInput.trim()) return;

    setTraversing(true);
    try {
      await onTraverse(dotPathInput);
      // In a real scenario, this would return and set traversalResult
      // For now, we'll just clear the input and show feedback
      setDotPathInput('');
    } finally {
      setTraversing(false);
    }
  }, [dotPathInput, onTraverse]);

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && dotPathInput.trim()) {
      handleTraverse();
    }
  };

  if (isLoading) {
    return (
      <Card className="related-objects-navigator">
        <CardHeader>
          <CardTitle className="text-base">Related Objects</CardTitle>
          <CardDescription>Business object relationships and graph traversal</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center gap-2 py-8">
            <div className="spinner" />
            <p className="text-sm text-gray-600">Loading related objects...</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="related-objects-navigator">
        <CardHeader>
          <CardTitle className="text-base">Related Objects</CardTitle>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error.message}</AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  const hasRelationships = linksTo.length > 0 || linksFrom.length > 0;

  return (
    <Card className="related-objects-navigator">
      <CardHeader>
        <CardTitle className="text-base">Related Objects</CardTitle>
        <CardDescription>
          {businessEntityName} connected objects in the graph
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Dot-path Traversal */}
        <div className="traversal-section">
          <h4 className="traversal-title">Graph Traversal (Dot Notation)</h4>
          <p className="traversal-description">
            Navigate related objects using dot notation (e.g., <code>Employee.department.company.name</code>)
          </p>
          <div className="traversal-input-group">
            <Input
              placeholder="e.g., Employee.department.company"
              value={dotPathInput}
              onChange={(e) => setDotPathInput(e.target.value)}
              onKeyPress={handleKeyPress}
              disabled={traversing}
              className="traversal-input"
            />
            <Button
              onClick={handleTraverse}
              disabled={!dotPathInput.trim() || traversing}
              size="sm"
              className="traversal-button"
            >
              {traversing ? (
                <>
                  <div className="spinner-small" />
                  Traversing...
                </>
              ) : (
                <>
                  <Search className="h-4 w-4 mr-2" />
                  Traverse
                </>
              )}
            </Button>
          </div>
        </div>

        {!hasRelationships ? (
          <div className="no-relationships">
            <p className="text-sm text-gray-600">
              No related objects found for {businessEntityName}.
            </p>
          </div>
        ) : (
          <>
            {/* Links To (Many-to-One) */}
            {linksTo.length > 0 && (
              <div className="relationships-section">
                <div className="relationships-header">
                  <ArrowRight className="h-4 w-4" />
                  <h4 className="relationships-title">Links To ({linksTo.length})</h4>
                </div>
                <p className="relationships-description">
                  {businessEntityName} references these objects
                </p>
                <div className="relationships-list">
                  {linksTo.map((model) => (
                    <RelatedObjectCard
                      key={model.id}
                      model={model}
                      direction="to"
                      fromEntity={businessEntityName}
                    />
                  ))}
                </div>
              </div>
            )}

            {/* Links From (One-to-Many) */}
            {linksFrom.length > 0 && (
              <div className="relationships-section">
                <div className="relationships-header">
                  <ArrowLeft className="h-4 w-4" />
                  <h4 className="relationships-title">Links From ({linksFrom.length})</h4>
                </div>
                <p className="relationships-description">
                  These objects reference {businessEntityName}
                </p>
                <div className="relationships-list">
                  {linksFrom.map((model) => (
                    <RelatedObjectCard
                      key={model.id}
                      model={model}
                      direction="from"
                      toEntity={businessEntityName}
                    />
                  ))}
                </div>
              </div>
            )}
          </>
        )}

        {/* Traversal Result */}
        {_traversalResult && (
          <div className="traversal-result">
            <h4 className="result-title">Traversal Result</h4>
            <p className="result-description">
              Found {_traversalResult.nodes.length} nodes connected via {_traversalResult.edges.length} edges
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
};

interface RelatedObjectCardProps {
  model: SemanticModelMetadata;
  direction: 'to' | 'from';
  fromEntity?: string;
  toEntity?: string;
}

const RelatedObjectCard: React.FC<RelatedObjectCardProps> = ({
  model,
  direction,
  fromEntity,
  toEntity,
}) => {
  return (
    <div className="related-object-card">
      <div className="card-header">
        <div className="flex items-center gap-2 flex-1">
          {direction === 'to' ? (
            <>
              <span className="entity-name">{fromEntity}</span>
              <ChevronRight className="h-4 w-4 text-gray-400" />
              <span className="entity-name">{model.node_name}</span>
            </>
          ) : (
            <>
              <span className="entity-name">{model.node_name}</span>
              <ChevronRight className="h-4 w-4 text-gray-400" />
              <span className="entity-name">{toEntity}</span>
            </>
          )}
        </div>
        <Badge variant="outline" className="text-xs">
          {direction === 'to' ? 'references' : 'referenced by'}
        </Badge>
      </div>
      {model.description && (
        <p className="card-description">{model.description}</p>
      )}
      {model.properties?.source_tables && (
        <div className="card-tables">
          {(model.properties.source_tables as string[]).map((table) => (
            <Badge key={table} variant="secondary" className="text-xs">
              {table}
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
};

export default RelatedObjectsNavigator;
