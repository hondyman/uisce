import React from 'react';
import { Activity, Transition } from './ProcessBuilder';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import { Trash2 } from 'lucide-react';

interface TransitionEditorProps {
  transitions: Transition[];
  activities: Activity[];
  onUpdateTransition: (id: string, updates: Partial<Transition>) => void;
  onDeleteTransition: (id: string) => void;
}

export const TransitionEditor: React.FC<TransitionEditorProps> = ({
  transitions,
  activities,
  onUpdateTransition,
  onDeleteTransition,
}) => {
  const getActivityName = (id: string) => {
    return activities.find(a => a.id === id)?.name || 'Unknown';
  };

  if (transitions.length === 0) {
    return (
      <div className="text-center text-gray-500 py-12">
        <p>No transitions yet.</p>
        <p className="text-sm mt-1">Connect activities on the canvas to create transitions.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h3 className="font-semibold">Process Transitions</h3>

      {transitions.map((transition, idx) => (
        <Card key={transition.id}>
          <CardContent className="p-4">
            <div className="flex items-start justify-between mb-4">
              <div>
                <div className="text-sm font-medium">
                  {getActivityName(transition.from)} → {getActivityName(transition.to)}
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  Transition {idx + 1}
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDeleteTransition(transition.id)}
                className="text-red-600"
              >
                <Trash2 size={16} />
              </Button>
            </div>

            <div className="space-y-3">
              <div>
                <Label className="text-xs">Label</Label>
                <Input
                  value={transition.label || ''}
                  onChange={(e) =>
                    onUpdateTransition(transition.id, { label: e.target.value })
                  }
                  placeholder="e.g., Approved, Rejected"
                  className="text-sm"
                />
              </div>

              <div>
                <Label className="text-xs">Condition (optional)</Label>
                <Input
                  value={transition.condition || ''}
                  onChange={(e) =>
                    onUpdateTransition(transition.id, { condition: e.target.value })
                  }
                  placeholder="e.g., status === 'approved'"
                  className="text-sm font-mono"
                />
                <p className="text-xs text-gray-500 mt-1">
                  JavaScript expression that evaluates to true/false
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
};
