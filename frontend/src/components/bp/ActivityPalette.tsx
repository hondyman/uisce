import React from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Activity, Clock, Mail, CheckSquare, Workflow, Database, Zap } from 'lucide-react';

interface ActivityType {
  id: string;
  name: string;
  icon: React.ReactNode;
  description: string;
  category: string;
}

const activityTypes: ActivityType[] = [
  {
    id: 'manual_task',
    name: 'Manual Task',
    icon: <Activity size={20} />,
    description: 'Human-performed task',
    category: 'Tasks',
  },
  {
    id: 'automated_task',
    name: 'Automated Task',
    icon: <Zap size={20} />,
    description: 'System-executed task',
    category: 'Tasks',
  },
  {
    id: 'approval',
    name: 'Approval',
    icon: <CheckSquare size={20} />,
    description: 'Approval decision point',
    category: 'Tasks',
  },
  {
    id: 'notification',
    name: 'Notification',
    icon: <Mail size={20} />,
    description: 'Send notification',
    category: 'Communication',
  },
  {
    id: 'data_collection',
    name: 'Data Collection',
    icon: <Database size={20} />,
    description: 'Collect data from user',
    category: 'Data',
  },
  {
    id: 'integration',
    name: 'Integration',
    icon: <Workflow size={20} />,
    description: 'External system integration',
    category: 'Integrations',
  },
  {
    id: 'wait',
    name: 'Wait',
    icon: <Clock size={20} />,
    description: 'Wait for duration or event',
    category: 'Control',
  },
];

interface ActivityPaletteProps {
  onAddActivity: (activityType: string) => void;
}

export const ActivityPalette: React.FC<ActivityPaletteProps> = ({ onAddActivity }) => {
  const categories = Array.from(new Set(activityTypes.map(a => a.category)));

  const handleDragStart = (e: React.DragEvent, activityType: string) => {
    e.dataTransfer.setData('activityType', activityType);
    e.dataTransfer.effectAllowed = 'copy';
  };

  return (
    <div className="p-4 space-y-4">
      <div>
        <h3 className="font-semibold text-sm text-gray-700 mb-2">Activity Palette</h3>
        <p className="text-xs text-gray-500">Drag activities onto the canvas</p>
      </div>

      {categories.map(category => (
        <div key={category}>
          <h4 className="font-medium text-xs text-gray-600 mb-2 uppercase">{category}</h4>
          <div className="space-y-2">
            {activityTypes
              .filter(a => a.category === category)
              .map(activity => (
                <Card
                  key={activity.id}
                  draggable
                  onDragStart={(e) => handleDragStart(e, activity.id)}
                  onClick={() => onAddActivity(activity.id)}
                  className="cursor-move hover:shadow-md transition-shadow"
                >
                  <CardContent className="p-3">
                    <div className="flex items-start gap-2">
                      <div className="text-gray-600 mt-0.5">
                        {activity.icon}
                      </div>
                      <div className="flex-1">
                        <div className="font-medium text-sm">{activity.name}</div>
                        <div className="text-xs text-gray-500">{activity.description}</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
          </div>
        </div>
      ))}
    </div>
  );
};
