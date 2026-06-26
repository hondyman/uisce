import React, { useState, useCallback } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ActivityPalette } from './ActivityPalette';
import { ActivityCanvas } from './ActivityCanvas';
import { TransitionEditor } from './TransitionEditor';
import { ProcessPreview } from './ProcessPreview';
import { Save, Play, Download, Upload } from 'lucide-react';
import { useMutation } from '@tanstack/react-query';

export interface Activity {
  id: string;
  type: string;
  name: string;
  config: Record<string, any>;
  position: { x: number; y: number };
}

export interface Transition {
  id: string;
  from: string;
  to: string;
  condition?: string;
  label?: string;
}

export interface BusinessProcess {
  id?: string;
  name: string;
  description: string;
  activities: Activity[];
  transitions: Transition[];
  metadata: Record<string, any>;
}

export const ProcessBuilder: React.FC = () => {
  const [process, setProcess] = useState<BusinessProcess>({
    name: '',
    description: '',
    activities: [],
    transitions: [],
    metadata: {},
  });

  const [selectedActivity, setSelectedActivity] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);

  // Save process mutation
  const saveProcess = useMutation({
    mutationFn: (data: BusinessProcess) =>
      fetch('/api/bp/processes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }).then(r => r.json()),
    onSuccess: (data) => {
      setProcess(prev => ({ ...prev, id: data.id }));
      setIsDirty(false);
    },
  });

  // Deploy process mutation
  const deployProcess = useMutation({
    mutationFn: (processId: string) =>
      fetch(`/api/bp/processes/${processId}/deploy`, {
        method: 'POST',
      }).then(r => r.json()),
  });

  const handleAddActivity = useCallback((activityType: string) => {
    const newActivity: Activity = {
      id: `activity-${Date.now()}`,
      type: activityType,
      name: `${activityType} ${process.activities.length + 1}`,
      config: {},
      position: {
        x: 100 + (process.activities.length % 3) * 250,
        y: 100 + Math.floor(process.activities.length / 3) * 150,
      },
    };

    setProcess(prev => ({
      ...prev,
      activities: [...prev.activities, newActivity],
    }));
    setIsDirty(true);
  }, [process.activities.length]);

  const handleUpdateActivity = useCallback((activityId: string, updates: Partial<Activity>) => {
    setProcess(prev => ({
      ...prev,
      activities: prev.activities.map(a =>
        a.id === activityId ? { ...a, ...updates } : a
      ),
    }));
    setIsDirty(true);
  }, []);

  const handleDeleteActivity = useCallback((activityId: string) => {
    setProcess(prev => ({
      ...prev,
      activities: prev.activities.filter(a => a.id !== activityId),
      transitions: prev.transitions.filter(
        t => t.from !== activityId && t.to !== activityId
      ),
    }));
    setIsDirty(true);
  }, []);

  const handleAddTransition = useCallback((from: string, to: string) => {
    const newTransition: Transition = {
      id: `transition-${Date.now()}`,
      from,
      to,
      label: 'Next',
    };

    setProcess(prev => ({
      ...prev,
      transitions: [...prev.transitions, newTransition],
    }));
    setIsDirty(true);
  }, []);

  const handleUpdateTransition = useCallback((transitionId: string, updates: Partial<Transition>) => {
    setProcess(prev => ({
      ...prev,
      transitions: prev.transitions.map(t =>
        t.id === transitionId ? { ...t, ...updates } : t
      ),
    }));
    setIsDirty(true);
  }, []);

  const handleDeleteTransition = useCallback((transitionId: string) => {
    setProcess(prev => ({
      ...prev,
      transitions: prev.transitions.filter(t => t.id !== transitionId),
    }));
    setIsDirty(true);
  }, []);

  const handleSave = () => {
    saveProcess.mutate(process);
  };

  const handleDeploy = () => {
    if (process.id) {
      deployProcess.mutate(process.id);
    }
  };

  const handleExport = () => {
    const dataStr = JSON.stringify(process, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${process.name || 'process'}.json`;
    link.click();
  };

  const handleImport = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        try {
          const imported = JSON.parse(e.target?.result as string);
          setProcess(imported);
          setIsDirty(true);
        } catch (error) {
          console.error('Failed to import process:', error);
        }
      };
      reader.readAsText(file);
    }
  };

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <div className="border-b bg-white p-4">
        <div className="flex items-center justify-between">
          <div className="flex-1 max-w-md">
            <Input
              value={process.name}
              onChange={(e) => {
                setProcess(prev => ({ ...prev, name: e.target.value }));
                setIsDirty(true);
              }}
              placeholder="Process Name"
              className="text-lg font-semibold"
            />
          </div>

          <div className="flex items-center gap-2">
            {isDirty && (
              <span className="text-sm text-orange-600">Unsaved changes</span>
            )}
            
            <input
              type="file"
              id="import-file"
              accept=".json"
              onChange={handleImport}
              className="hidden"
            />
            <Button
              variant="outline"
              size="sm"
              onClick={() => document.getElementById('import-file')?.click()}
            >
              <Upload size={16} className="mr-1" />
              Import
            </Button>

            <Button variant="outline" size="sm" onClick={handleExport}>
              <Download size={16} className="mr-1" />
              Export
            </Button>

            <Button
              variant="outline"
              size="sm"
              onClick={handleSave}
              disabled={!isDirty || saveProcess.isLoading}
            >
              <Save size={16} className="mr-1" />
              {saveProcess.isLoading ? 'Saving...' : 'Save'}
            </Button>

            <Button
              size="sm"
              onClick={handleDeploy}
              disabled={!process.id || deployProcess.isLoading}
            >
              <Play size={16} className="mr-1" />
              {deployProcess.isLoading ? 'Deploying...' : 'Deploy'}
            </Button>
          </div>
        </div>

        {process.description && (
          <p className="text-sm text-gray-600 mt-2">{process.description}</p>
        )}
      </div>

      {/* Main Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left Sidebar - Activity Palette */}
        <div className="w-64 border-r bg-gray-50 overflow-y-auto">
          <ActivityPalette onAddActivity={handleAddActivity} />
        </div>

        {/* Center - Canvas */}
        <div className="flex-1 overflow-hidden">
          <Tabs defaultValue="canvas" className="h-full flex flex-col">
            <TabsList className="mx-4 mt-2">
              <TabsTrigger value="canvas">Canvas</TabsTrigger>
              <TabsTrigger value="transitions">Transitions</TabsTrigger>
              <TabsTrigger value="preview">Preview</TabsTrigger>
            </TabsList>

            <TabsContent value="canvas" className="flex-1 overflow-hidden">
              <ActivityCanvas
                activities={process.activities}
                transitions={process.transitions}
                selectedActivity={selectedActivity}
                onSelectActivity={setSelectedActivity}
                onUpdateActivity={handleUpdateActivity}
                onDeleteActivity={handleDeleteActivity}
                onAddTransition={handleAddTransition}
              />
            </TabsContent>

            <TabsContent value="transitions" className="flex-1 overflow-auto p-4">
              <TransitionEditor
                transitions={process.transitions}
                activities={process.activities}
                onUpdateTransition={handleUpdateTransition}
                onDeleteTransition={handleDeleteTransition}
              />
            </TabsContent>

            <TabsContent value="preview" className="flex-1 overflow-auto p-4">
              <ProcessPreview process={process} />
            </TabsContent>
          </Tabs>
        </div>

        {/* Right Sidebar - Property Editor */}
        {selectedActivity && (
          <div className="w-80 border-l bg-white overflow-y-auto">
            <Card className="border-0 rounded-none">
              <CardHeader>
                <CardTitle>Activity Properties</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {(() => {
                  const activity = process.activities.find(a => a.id === selectedActivity);
                  if (!activity) return null;

                  return (
                    <>
                      <div>
                        <Label>Name</Label>
                        <Input
                          value={activity.name}
                          onChange={(e) =>
                            handleUpdateActivity(selectedActivity, { name: e.target.value })
                          }
                        />
                      </div>

                      <div>
                        <Label>Type</Label>
                        <Input value={activity.type} disabled />
                      </div>

                      <div>
                        <Label>Description</Label>
                        <Input
                          value={activity.config.description || ''}
                          onChange={(e) =>
                            handleUpdateActivity(selectedActivity, {
                              config: { ...activity.config, description: e.target.value },
                            })
                          }
                          placeholder="Activity description"
                        />
                      </div>

                      <div className="pt-4 border-t">
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => {
                            handleDeleteActivity(selectedActivity);
                            setSelectedActivity(null);
                          }}
                          className="w-full"
                        >
                          Delete Activity
                        </Button>
                      </div>
                    </>
                  );
                })()}
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  );
};
