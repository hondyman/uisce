import React from 'react';
import { BusinessProcess } from './ProcessBuilder';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { CheckCircle2, AlertCircle, Info } from 'lucide-react';

interface ProcessPreviewProps {
  process: BusinessProcess;
}

export const ProcessPreview: React.FC<ProcessPreviewProps> = ({ process }) => {
  const validateProcess = () => {
    const errors: string[] = [];
    const warnings: string[] = [];

    if (!process.name) {
      errors.push('Process name is required');
    }

    if (process.activities.length === 0) {
      errors.push('Process must have at least one activity');
    }

    if (process.transitions.length === 0 && process.activities.length > 1) {
      warnings.push('Multiple activities with no transitions');
    }

    // Check for orphaned activities (no incoming or outgoing transitions)
    const connectedActivities = new Set<string>();
    process.transitions.forEach(t => {
      connectedActivities.add(t.from);
      connectedActivities.add(t.to);
    });

    const orphanedCount = process.activities.filter(
      a => !connectedActivities.has(a.id)
    ).length;

    if (orphanedCount > 0) {
      warnings.push(`${orphanedCount} orphaned activity/activities`);
    }

    return { errors, warnings };
  };

  const { errors, warnings } = validateProcess();

  const getActivityStats = () => {
    const types = process.activities.reduce((acc, activity) => {
      acc[activity.type] = (acc[activity.type] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    return types;
  };

  const stats = getActivityStats();

  return (
    <div className="space-y-6">
      {/* Validation Status */}
      <Card>
        <CardHeader>
          <CardTitle>Validation Status</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {errors.length === 0 && warnings.length === 0 ? (
            <div className="flex items-center text-green-600">
              <CheckCircle2 className="mr-2" size={20} />
              <span>Process is valid and ready to deploy</span>
            </div>
          ) : (
            <>
              {errors.map((error, idx) => (
                <Alert key={`error-${idx}`} variant="destructive">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              ))}
              
              {warnings.map((warning, idx) => (
                <Alert key={`warning-${idx}`}>
                  <Info className="h-4 w-4" />
                  <AlertDescription>{warning}</AlertDescription>
                </Alert>
              ))}
            </>
          )}
        </CardContent>
      </Card>

      {/* Process Summary */}
      <Card>
        <CardHeader>
          <CardTitle>Process Summary</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <div className="text-sm text-gray-600">Total Activities</div>
              <div className="text-2xl font-bold">{process.activities.length}</div>
            </div>
            <div>
              <div className="text-sm text-gray-600">Transitions</div>
              <div className="text-2xl font-bold">{process.transitions.length}</div>
            </div>
          </div>

          {Object.keys(stats).length > 0 && (
            <div>
              <div className="text-sm font-medium mb-2">Activity Types</div>
              <div className="flex flex-wrap gap-2">
                {Object.entries(stats).map(([type, count]) => (
                  <Badge key={type} variant="secondary">
                    {type}: {count}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Process Flow */}
      <Card>
        <CardHeader>
          <CardTitle>Process Flow</CardTitle>
        </CardHeader>
        <CardContent>
          {process.activities.length > 0 ? (
            <div className="space-y-2">
              {process.activities.map((activity, idx) => (
                <div
                  key={activity.id}
                  className="flex items-center p-2 border rounded hover:bg-gray-50"
                >
                  <div className="flex-shrink-0 w-8 h-8 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center text-sm font-medium mr-3">
                    {idx + 1}
                  </div>
                  <div className="flex-1">
                    <div className="font-medium text-sm">{activity.name}</div>
                    <div className="text-xs text-gray-500">{activity.type}</div>
                  </div>
                  <Badge variant="outline" className="text-xs">
                    {process.transitions.filter(t => t.from === activity.id).length} out
                  </Badge>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center text-gray-500 py-8">
              No activities to preview
            </div>
          )}
        </CardContent>
      </Card>

      {/* Temporal Deployment Info */}
      {process.id && (
        <Card>
          <CardHeader>
            <CardTitle>Deployment Information</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-gray-600">Process ID:</span>
                <span className="font-mono">{process.id}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Workflow Type:</span>
                <span className="font-mono">BusinessProcess</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Task Queue:</span>
                <span className="font-mono">business-processes</span>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};
