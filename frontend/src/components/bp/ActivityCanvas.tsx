import React, { useRef, useState, useCallback } from 'react';
import { Activity as ActivityType, Transition } from './ProcessBuilder';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Trash2, Plus } from 'lucide-react';
import { devDebug } from '../../utils/devLogger';

interface ActivityCanvasProps {
  activities: ActivityType[];
  transitions: Transition[];
  selectedActivity: string | null;
  onSelectActivity: (id: string | null) => void;
  onUpdateActivity: (id: string, updates: Partial<ActivityType>) => void;
  onDeleteActivity: (id: string) => void;
  onAddTransition: (from: string, to: string) => void;
}

export const ActivityCanvas: React.FC<ActivityCanvasProps> = ({
  activities,
  transitions,
  selectedActivity,
  onSelectActivity,
  onUpdateActivity,
  onDeleteActivity,
  onAddTransition,
}) => {
  const canvasRef = useRef<HTMLDivElement>(null);
  const [draggedActivity, setDraggedActivity] = useState<string | null>(null);
  const [connectingFrom, setConnectingFrom] = useState<string | null>(null);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    const activityType = e.dataTransfer.getData('activityType');
    
    if (activityType && canvasRef.current) {
      const rect = canvasRef.current.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;

      // This would be handled by parent component through onAddActivity
      devDebug(`Add ${activityType} at (${x}, ${y})`);
    }
  }, []);

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
  };

  const handleActivityDragStart = (e: React.DragEvent, activityId: string) => {
    setDraggedActivity(activityId);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleActivityDrag = useCallback((e: React.DragEvent, activityId: string) => {
    if (!draggedActivity || !canvasRef.current) return;

    const rect = canvasRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left - 75; // Half of card width
    const y = e.clientY - rect.top - 50; // Half of card height

    if (x > 0 && y > 0) {
      onUpdateActivity(activityId, {
        position: { x, y },
      });
    }
  }, [draggedActivity, onUpdateActivity]);

  const handleActivityDragEnd = () => {
    setDraggedActivity(null);
  };

  const handleStartConnection = (activityId: string) => {
    if (connectingFrom === activityId) {
      setConnectingFrom(null);
    } else {
      setConnectingFrom(activityId);
    }
  };

  const handleCompleteConnection = (toActivityId: string) => {
    if (connectingFrom && connectingFrom !== toActivityId) {
      onAddTransition(connectingFrom, toActivityId);
      setConnectingFrom(null);
    }
  };

  const getActivityColor = (type: string) => {
    const colors: Record<string, string> = {
      manual_task: 'bg-blue-100 border-blue-300',
      automated_task: 'bg-green-100 border-green-300',
      approval: 'bg-yellow-100 border-yellow-300',
      notification: 'bg-purple-100 border-purple-300',
      data_collection: 'bg-pink-100 border-pink-300',
      integration: 'bg-orange-100 border-orange-300',
      wait: 'bg-gray-100 border-gray-300',
    };
    return colors[type] || 'bg-gray-100 border-gray-300';
  };

  // Calculate transition line coordinates
  const getTransitionPath = (transition: Transition) => {
    const fromActivity = activities.find(a => a.id === transition.from);
    const toActivity = activities.find(a => a.id === transition.to);

    if (!fromActivity || !toActivity) return '';

    const fromX = fromActivity.position.x + 75; // Center X
    const fromY = fromActivity.position.y + 50; // Bottom
    const toX = toActivity.position.x + 75;     // Center X
    const toY = toActivity.position.y;          // Top

    return `M ${fromX} ${fromY} L ${toX} ${toY}`;
  };

  return (
    <div
      ref={canvasRef}
      className="relative w-full h-full bg-gray-50 overflow-auto"
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      onClick={() => onSelectActivity(null)}
    >
      {/* SVG Layer for Transitions */}
      <svg className="absolute inset-0 pointer-events-none" style={{ zIndex: 1 }}>
        {transitions.map(transition => (
          <g key={transition.id}>
            <path
              d={getTransitionPath(transition)}
              stroke="#3b82f6"
              strokeWidth="2"
              fill="none"
              markerEnd="url(#arrowhead)"
            />
          </g>
        ))}
        <defs>
          <marker
            id="arrowhead"
            markerWidth="10"
            markerHeight="10"
            refX="9"
            refY="3"
            orient="auto"
          >
            <polygon points="0 0, 10 3, 0 6" fill="#3b82f6" />
          </marker>
        </defs>
      </svg>

      {/* Activity Nodes */}
      {activities.map(activity => (
        <Card
          key={activity.id}
          draggable
          onDragStart={(e) => handleActivityDragStart(e, activity.id)}
          onDrag={(e) => handleActivityDrag(e, activity.id)}
          onDragEnd={handleActivityDragEnd}
          onClick={(e) => {
            e.stopPropagation();
            if (connectingFrom) {
              handleCompleteConnection(activity.id);
            } else {
              onSelectActivity(activity.id);
            }
          }}
          className={`
            absolute w-40 cursor-move select-none transition-shadow
            ${getActivityColor(activity.type)}
            ${selectedActivity === activity.id ? 'ring-2 ring-blue-500 shadow-lg' : ''}
            ${connectingFrom === activity.id ? 'ring-2 ring-green-500' : ''}
          `}
          style={{
            left: activity.position.x,
            top: activity.position.y,
            zIndex: 10,
          }}
        >
          <div className="p-3">
            <div className="font-medium text-sm mb-1">{activity.name}</div>
            <div className="text-xs text-gray-600">{activity.type}</div>
            
            <div className="flex gap-1 mt-2">
              <Button
                size="sm"
                variant="outline"
                className="h-6 px-2 text-xs"
                onClick={(e) => {
                  e.stopPropagation();
                  handleStartConnection(activity.id);
                }}
              >
                <Plus size={12} />
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="h-6 px-2 text-xs text-red-600"
                onClick={(e) => {
                  e.stopPropagation();
                  onDeleteActivity(activity.id);
                }}
              >
                <Trash2 size={12} />
              </Button>
            </div>
          </div>
        </Card>
      ))}

      {/* Empty State */}
      {activities.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center text-gray-400">
          <div className="text-center">
            <p className="text-lg font-medium">Drag activities here to start building</p>
            <p className="text-sm mt-1">Or click activities in the palette</p>
          </div>
        </div>
      )}
    </div>
  );
};