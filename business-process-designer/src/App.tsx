import { useState, useCallback } from 'react';
import { MantineProvider, AppShell, Burger } from '@mantine/core';
import { ReactFlowProvider } from 'reactflow';
import 'reactflow/dist/style.css';

import ProcessHeader from './components/ProcessHeader';
import StepPalette from './components/StepPalette';
import ProcessCanvas from './components/ProcessCanvas';
import StepConfigPanel from './components/StepConfigPanel';
import MiniMap from './components/MiniMap';
import { businessProcessService } from './services/businessProcessService';
import { WorkflowDefinition, WorkflowNode } from '../../../bp-backend/pkg/workflow/workflow';

import './App.css';

function App() {
  const [opened, setOpened] = useState(false);
  const [selectedNode, setSelectedNode] = useState<any>(null);
  const [miniMapVisible, setMiniMapVisible] = useState(true);
  const [currentProcess, setCurrentProcess] = useState<WorkflowDefinition>({
    RootNodeID: '',
    Nodes: {},
    Edges: {}
  });
  const [saving, setSaving] = useState(false);

  const handleNodeSelect = useCallback((node: any) => {
    setSelectedNode(node);
  }, []);

  const handleNodeUpdate = useCallback((nodeId: string, updates: any) => {
    setCurrentProcess(prev => ({
      ...prev,
      Nodes: {
        ...prev.Nodes,
        [nodeId]: {
          ...prev.Nodes[nodeId],
          ...updates
        }
      }
    }))
  }, []);

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      const result = await businessProcessService.createBusinessProcess(currentProcess);
      console.log('Process saved successfully', result);
    } catch (error: any) {
      console.error('Failed to save process:', error.message);
      // TODO: Show error notification to user
    } finally {
      setSaving(false);
    }
  }, [currentProcess]);

  return (
    <MantineProvider>
      <AppShell
        header={{ height: 60 }}
        navbar={{
          width: 300,
          breakpoint: 'sm',
          collapsed: { mobile: !opened }
        }}
        aside={{
          width: 300,
          breakpoint: 'sm'
        }}
      >
        <AppShell.Header>
          <div className="header-container">
            <Burger
              opened={opened}
              onClick={() => setOpened((o) => !o)}
              size="sm"
              style={{ marginRight: 16 }}
            />
            <ProcessHeader
              processName="New Business Process"
              onSave={handleSave}
              saving={saving}
            />
          </div>
        </AppShell.Header>

        <AppShell.Navbar p="md">
          <StepPalette />
        </AppShell.Navbar>

        <AppShell.Main>
          <div className="canvas-container">
            <ReactFlowProvider>
              <ProcessCanvas
                onNodeSelect={handleNodeSelect}
                processData={currentProcess}
                onProcessChange={setCurrentProcess}
              />
              <MiniMap
                visible={miniMapVisible}
                onToggle={setMiniMapVisible}
              />
            </ReactFlowProvider>
          </div>
        </AppShell.Main>

        <AppShell.Aside p="md">
          <StepConfigPanel
            selectedNode={selectedNode}
            onUpdateNode={handleNodeUpdate}
          />
        </AppShell.Aside>
      </AppShell>
    </MantineProvider>
  );
}

export default App;