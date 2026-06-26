import React, { useEffect, useState } from 'react';

interface ApprovalEvent {
  timestamp: string;
  action: string;
  approverRole: string;
  level: number;
}

interface ApprovalProgressProps {
  workflowId: string;
  stepKey: string;
  escalations: any[]; // Definition from node
  slaExpr: string;
}

export const ApprovalProgressPanel: React.FC<ApprovalProgressProps> = ({ workflowId, stepKey, escalations, slaExpr }) => {
  const [history, setHistory] = useState<ApprovalEvent[]>([]);
  const [currentLevel, setCurrentLevel] = useState(0);

  useEffect(() => {
    // Poll for updates
    const interval = setInterval(async () => {
      try {
        const res = await fetch(`/api/workflows/${workflowId}/events?step=${stepKey}`);
        if (res.ok) {
            const data = await res.json();
            setHistory(data.escalationHistory || []);
            setCurrentLevel(data.currentEscalationLevel || 0);
        }
      } catch (e) {
        console.error("Failed to fetch approval progress", e);
      }
    }, 3000); // 3s poll

    return () => clearInterval(interval);
  }, [workflowId, stepKey]);

  return (
    <div className="p-4 border rounded bg-white shadow-sm mt-4">
      <h3 className="text-lg font-semibold mb-2">Approval Progress</h3>
      
      {/* Timeline Visualization */}
      <div className="relative">
        {/* Connector Line */}
        <div className="absolute left-2 top-2 bottom-2 w-0.5 bg-gray-200"></div>

        {/* Steps */}
        <div className="space-y-6">
            {/* Initial Request */}
            <TimelineItem 
                active={currentLevel >= 0} 
                completed={currentLevel > 0}
                title="Initial Request"
                description="Waiting for Manager"
                timestamp={getEventTime(history, 0)}
            />

            {/* Escalation Steps */}
            {escalations.map((esc, idx) => (
                <TimelineItem 
                    key={idx}
                    active={currentLevel >= idx + 1}
                    completed={currentLevel > idx + 1}
                    title={`Escalation Level ${idx + 1}`}
                    description={`Escalated to ${esc.targetActorRole}`}
                    timestamp={getEventTime(history, idx + 1)}
                    isEscalation
                />
            ))}
        </div>
      </div>

       {/* SLA Status */}
       {slaExpr && (
         <div className="mt-4 text-xs text-gray-500 border-t pt-2">
            Overall SLA: <span className="font-mono">{slaExpr}</span>
         </div>
       )}
    </div>
  );
};

const TimelineItem = ({ active, completed, title, description, timestamp, isEscalation }: any) => (
    <div className="relative pl-8">
        {/* Dot */}
        <div className={`absolute left-0 top-1.5 w-4 h-4 rounded-full border-2 
            ${completed ? 'bg-green-500 border-green-500' : 
              active ? 'bg-blue-500 border-blue-500 animate-pulse' : 
              'bg-white border-gray-300'}`}>
        </div>
        
        <div className={`${active ? 'text-gray-900' : 'text-gray-400'}`}>
            <div className="font-medium text-sm">{title}</div>
            <div className="text-xs">{description}</div>
            {timestamp && <div className="text-xs text-gray-500 mt-1">{new Date(timestamp).toLocaleTimeString()}</div>}
        </div>
    </div>
);

function getEventTime(history: ApprovalEvent[], level: number) {
    // Find event that triggered this level
    // Logic: Initial is level 0. Escalation 1 is level 1 event.
    // We assume history contains transition events.
    // Simplified: just match level
    const event = history.find(e => e.level === level);
    return event ? event.timestamp : null;
}
