import React, { useState } from 'react';
import {
  AuditEvent,
  ChangeSet,
  AIExplanation,
  useAuditEvents,
  useExplainAudit,
  useCreateChangeSetFromAI,
  useAuditExplainerFlow,
  ImpactedEntityType,
} from '@/hooks/useAuditGraphHooks';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Loader2, AlertCircle, CheckCircle, Zap } from 'lucide-react';

interface AuditExplorerGraphProps {
  scope: 'global' | 'multi-tenant-assigned' | 'tenant' | 'tenant-ops';
  role: 'GLOBAL_ADMIN' | 'GLOBAL_OPS' | 'TENANT_ADMIN' | 'TENANT_OPS';
  tenantIds: string[];
}

/**
 * AuditExplorerGraph - Main audit explorer component for role-aware audit event visualization
 */
export const AuditExplorerGraph: React.FC<AuditExplorerGraphProps> = ({ scope, role, tenantIds }) => {
  const [timeRange, setTimeRange] = useState({ from: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), to: new Date() });
  const [selectedEvent, setSelectedEvent] = useState<AuditEvent | null>(null);
  const [activeTab, setActiveTab] = useState<'timeline' | 'entities' | 'incidents' | 'compliance' | 'ai'>('timeline');

  const { data: events, isLoading: eventsLoading } = useAuditEvents(tenantIds, timeRange.from, timeRange.to);

  return (
    <div className="flex h-full gap-4 p-4">
      {/* Main timeline view */}
      <div className="flex-1">
        <Card>
          <CardHeader>
            <CardTitle>Audit Timeline</CardTitle>
            <CardDescription>Events across {tenantIds.length} tenant(s)</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {eventsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin" />
                </div>
              ) : events && events.length > 0 ? (
                <TimelineViewGraph events={events} onSelect={setSelectedEvent} selectedId={selectedEvent?.id} />
              ) : (
                <div className="py-8 text-center text-gray-500">No events found</div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Right panel: AI Explanation + ChangeSet Proposal */}
      {selectedEvent && <AIPanelWithChangeSetProposal event={selectedEvent} />}
    </div>
  );
};

/**
 * TimelineViewGraph - Renders a unified timeline of audit events
 */
const TimelineViewGraph: React.FC<{
  events: AuditEvent[];
  onSelect: (event: AuditEvent) => void;
  selectedId?: string;
}> = ({ events, onSelect, selectedId }) => {
  return (
    <div className="space-y-2">
      {events.map((event) => (
        <TimelineRowGraph
          key={event.id}
          event={event}
          onSelect={onSelect}
          isSelected={event.id === selectedId}
        />
      ))}
    </div>
  );
};

/**
 * TimelineRowGraph - Renders a single audit event row
 */
const TimelineRowGraph: React.FC<{
  event: AuditEvent;
  onSelect: (event: AuditEvent) => void;
  isSelected: boolean;
}> = ({ event, onSelect, isSelected }) => {
  const getIcon = (type: string) => {
    switch (type) {
      case 'JOB_RUN':
        return '⚙️';
      case 'DAG_RUN':
        return '🔄';
      case 'INCIDENT':
        return '⚠️';
      case 'CHANGESET_EVENT':
        return '📝';
      case 'COMPLIANCE_EVENT':
        return '🔒';
      case 'AI_SUGGESTION':
        return '💡';
      default:
        return '📋';
    }
  };

  const getStatusBadgeColor = (status?: string) => {
    if (!status) return 'gray';
    if (status.includes('SUCCESS')) return 'green';
    if (status.includes('FAIL')) return 'red';
    return 'yellow';
  };

  return (
    <div
      onClick={() => onSelect(event)}
      className={`p-3 rounded-lg border cursor-pointer transition-colors ${
        isSelected ? 'bg-blue-50 border-blue-400' : 'bg-white hover:bg-gray-50 border-gray-200'
      }`}
    >
      <div className="flex items-start gap-3">
        <div className="text-2xl">{getIcon(event.type)}</div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className="font-mono text-sm text-gray-500">{new Date(event.timestamp).toLocaleTimeString()}</span>
            <Badge variant="outline" className="text-xs">
              {event.type}
            </Badge>
            {event.status && (
              <Badge className={`text-xs bg-${getStatusBadgeColor(event.status)}-100`}>{event.status}</Badge>
            )}
          </div>
          <p className="text-sm font-medium mt-1">{event.title || event.artifactType}</p>
          {event.metadata?.error_message && (
            <p className="text-sm text-red-600 mt-1">{event.metadata.error_message}</p>
          )}
        </div>
        {isSelected && <Zap className="h-5 w-5 text-blue-500" />}
      </div>
    </div>
  );
};

/**
 * AIPanelWithChangeSetProposal - Right-side panel showing AI explanation and ChangeSet proposal
 */
const AIPanelWithChangeSetProposal: React.FC<{ event: AuditEvent }> = ({ event }) => {
  const [explanation, setExplanation] = useState<AIExplanation | null>(null);
  const [showChangeSetModal, setShowChangeSetModal] = useState(false);
  const explainFlow = useAuditExplainerFlow(event);

  const handleExplain = async () => {
    const result = await explainFlow.explain.mutate();
    if (result) {
      setExplanation(result);
    }
  };

  return (
    <div className="w-96 space-y-4">
      {/* Explanation Card */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">AI Analysis</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {explanation ? (
            <div className="space-y-3 text-sm">
              <div>
                <p className="font-semibold text-gray-700 mb-1">Narrative</p>
                <p className="text-gray-600">{explanation.narrative}</p>
              </div>
              <div>
                <p className="font-semibold text-gray-700 mb-1">Root Cause</p>
                <p className="text-gray-600">{explanation.rootCause}</p>
              </div>
              <div>
                <p className="font-semibold text-gray-700 mb-1">Blast Radius</p>
                <p className="text-gray-600">{explanation.blastRadius}</p>
              </div>
              <div>
                <p className="font-semibold text-gray-700 mb-1">Recommended Fix</p>
                <p className="text-gray-600">{explanation.recommendedFix}</p>
              </div>
              {explanation.confidence && (
                <div className="flex items-center gap-2 pt-2">
                  <div className="flex-1 bg-gray-200 rounded h-2">
                    <div className="bg-blue-500 h-2 rounded" style={{ width: `${explanation.confidence * 100}%` }} />
                  </div>
                  <span className="text-xs text-gray-500">{Math.round(explanation.confidence * 100)}%</span>
                </div>
              )}
              <Button
                onClick={() => setShowChangeSetModal(true)}
                className="w-full mt-3"
                variant="default"
                size="sm"
              >
                Propose ChangeSet
              </Button>
            </div>
          ) : (
            <Button onClick={handleExplain} disabled={explainFlow.explain.isLoading} className="w-full" size="sm">
              {explainFlow.explain.isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Analyzing...
                </>
              ) : (
                <>
                  <Zap className="mr-2 h-4 w-4" />
                  Explain with AI
                </>
              )}
            </Button>
          )}
        </CardContent>
      </Card>

      {/* ChangeSet Proposal Modal */}
      {showChangeSetModal && explanation && (
        <ChangeSetProposalModalGraph
          explanation={explanation}
          event={event}
          onClose={() => setShowChangeSetModal(false)}
        />
      )}
    </div>
  );
};

/**
 * ChangeSetProposalModalGraph - Modal for creating a ChangeSet from AI suggestion
 */
const ChangeSetProposalModalGraph: React.FC<{
  explanation: AIExplanation;
  event: AuditEvent;
  onClose: () => void;
}> = ({ explanation, event, onClose }) => {
  const [title, setTitle] = useState(explanation.suggestedChangeSetSummary);
  const [description, setDescription] = useState(
    `${explanation.narrative}\n\nRoot Cause: ${explanation.rootCause}\n\nRecommendation: ${explanation.recommendedFix}`
  );
  const createChangeSetMutation = useCreateChangeSetFromAI();

  const handleCreateChangeSet = async () => {
    await createChangeSetMutation.mutateAsync({
      title,
      description,
      tenantId: event.tenantId,
      sourceEventId: event.id,
      impactedEntities: [], // In real impl, extract from explanation/graph
    });
    onClose();
  };

  return (
    <Card className="border-blue-200 bg-blue-50">
      <CardHeader>
        <CardTitle className="text-base">Propose ChangeSet</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div>
          <label className="text-sm font-semibold">Title</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full mt-1 p-2 border rounded text-sm"
          />
        </div>
        <div>
          <label className="text-sm font-semibold">Description</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full mt-1 p-2 border rounded text-sm h-24"
          />
        </div>
        <div className="flex gap-2">
          <Button onClick={handleCreateChangeSet} disabled={createChangeSetMutation.isPending} className="flex-1" size="sm">
            {createChangeSetMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            Create ChangeSet
          </Button>
          <Button onClick={onClose} variant="outline" className="flex-1" size="sm">
            Cancel
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};

export default AuditExplorerGraph;
