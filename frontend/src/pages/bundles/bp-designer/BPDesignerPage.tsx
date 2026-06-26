/**
 * BPDesignerPage.tsx
 * Main Business Process Designer page with Workday-inspired layout
 * Allows users to create processes by dragging steps onto a canvas
 * and configuring validation rules without code
 */

import { useState, useRef, useEffect } from 'react';
import type { FC, DragEvent } from 'react';
import { useParams } from 'react-router-dom';
import { StepPalette } from './StepPalette';
import { RuleBuilderModal } from './RuleBuilderModal';
import {
  useStepTypes,
  useValidationOperators,
  useWorkflowEvents,
  useBusinessObjects,
  useProcess,
  useUpdateProcess,
} from './useBPDesignerAPI';
import { ProcessNode, ProcessEdge } from './types';
import styles from './BPDesigner.module.css';
import { v4 as uuidv4 } from 'uuid';

interface DraggingState {
  isActive: boolean;
  stepType?: unknown;
}

export const BPDesignerPage: FC = () => {
  const { id } = useParams<{ id?: string }>();
  const canvasRef = useRef<HTMLDivElement>(null);

  // State
  const [nodes, setNodes] = useState<ProcessNode[]>([]);
  const [edges, setEdges] = useState<ProcessEdge[]>([]);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [processName, setProcessName] = useState('New Process');
  const [dragging, setDragging] = useState<DraggingState>({ isActive: false });
  const [showRuleModal, setShowRuleModal] = useState(false);
  const [isUnsaved, setIsUnsaved] = useState(false);

  // Queries
  useStepTypes();
  const { data: operators = [] } = useValidationOperators();
  const { data: events = [], isError: eventsError, isLoading: eventsLoading } = useWorkflowEvents();
  const { data: businessObjects = [] } = useBusinessObjects();
  const { data: process } = useProcess(id || null);
  const updateProcessMutation = useUpdateProcess();

  // Load existing process
  useEffect(() => {
    if (process) {
      setNodes(process.nodes || []);
      setEdges(process.edges || []);
      setProcessName(process.name);
    }
  }, [process]);

  // Handlers
  const handleDragStart = (e: DragEvent<Element>, stepType: unknown) => {
    setDragging({ isActive: true, stepType });
    e.dataTransfer.effectAllowed = 'copy';
  };

  const handleDragOver = (e: DragEvent<Element>) => {
    if (dragging.isActive) {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'copy';
    }
  };

  const handleDrop = (e: DragEvent<Element>) => {
    e.preventDefault();
    if (!dragging.stepType || !canvasRef.current) return;

    const rect = canvasRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    // Narrow the unknown stepType into a local typed shape before reading fields
    const st = dragging.stepType as { label?: string; key?: string; default_data?: unknown } | undefined;
    const newNode: ProcessNode = {
      id: `node-${uuidv4()}`,
      type: 'default',
      data: {
        label: String(st?.label ?? ''),
        stepKey: String(st?.key ?? ''),
  config: (st?.default_data ?? {}) as Record<string, unknown>,
      },
      position: { x, y },
    };

    setNodes([...nodes, newNode]);
    setIsUnsaved(true);
    setDragging({ isActive: false });
  };

  const handleNodeClick = (nodeId: string) => {
    setSelectedNodeId(nodeId);
  };

  const handleDeleteNode = (nodeId: string) => {
    setNodes(nodes.filter((n) => n.id !== nodeId));
    setEdges(edges.filter((e) => e.source !== nodeId && e.target !== nodeId));
    setSelectedNodeId(null);
    setIsUnsaved(true);
  };

  const handleSaveProcess = async () => {
    if (!id) {
      const notification = useNotification();
      notification.error('Process ID not found');
      return;
    }

    try {
      await updateProcessMutation.mutateAsync({
        id,
        nodes,
        edges,
      });
      setIsUnsaved(false);
      const notification = useNotification();
      notification.success('Process saved successfully');
    } catch (error) {
      const notification = useNotification();
      notification.error(`Failed to save process: ${error}`);
    }
  };


  const selectedNode = nodes.find((n) => n.id === selectedNodeId);
  // Safely read rules array from the selected node's config without using `as any`
  const selectedNodeRules: unknown[] = Array.isArray(selectedNode?.data?.config?.rules)
    ? (selectedNode!.data.config.rules as unknown[])
    : [];

  return (
    <div className={styles.container}>
      {/* Header */}
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <span className={styles.headerIcon}>📊</span>
          <h1 className={styles.headerTitle}>{processName}</h1>
          <div className={styles.headerMeta}>
            <span className={styles.versionBadge}>v1.0 Draft</span>
            {isUnsaved && <div className={styles.unsavedIndicator} title="Unsaved Changes" />}
          </div>
        </div>

        <div className={styles.headerRight}>
          <div className={styles.buttonGroup}>
            <button
              className={styles.buttonSecondary}
              onClick={handleSaveProcess}
              disabled={!isUnsaved}
            >
              Save
            </button>
            <button className={styles.buttonPrimary}>Publish</button>
            <button className={styles.toolbarButton}>⏱</button>
          </div>
          <div className={styles.userAvatar} />
        </div>
      </header>

      {/* Main Content */}
      <main className={styles.main}>
        {/* Left Sidebar - Step Palette */}
        <StepPalette onDragStart={handleDragStart} />

        {/* Canvas */}
        <div
          className={styles.canvas}
          ref={canvasRef}
          onDragOver={handleDragOver}
          onDrop={handleDrop}
        >
          {/* Canvas Toolbar */}
          <div className={styles.canvasToolbar}>
            <button className={styles.toolbarButton} title="Zoom In">
              ➕
            </button>
            <button className={styles.toolbarButton} title="Zoom Out">
              ➖
            </button>
            <button className={styles.toolbarButton} title="Fit to Screen">
              ↔
            </button>
            <button className={styles.toolbarButton} title="Undo">
              ↶
            </button>
            <button className={styles.toolbarButton} title="Redo">
              ↷
            </button>
          </div>

          {/* Canvas Area */}
          <div className={styles.canvasRelative}>
            {nodes.length === 0 ? (
              <div className={styles.canvasPlaceholder}>
                <div className={styles.canvasPlaceholderIcon}>📋</div>
                <p className={styles.canvasPlaceholderText}>Process Canvas</p>
                <p className={styles.canvasPlaceholderHint}>
                  Drag a step from the palette to get started
                </p>
              </div>
            ) : (
              <div className={styles.nodesList}>
                {nodes.map((node) => (
                  <div
                    key={node.id}
                    className={`${styles.nodeCard} ${
                      selectedNodeId === node.id ? styles.selected : ''
                    }`}
                    data-x={node.position.x}
                    data-y={node.position.y}
                  >
                    <div
                      className={styles.nodeCardContent}
                      onClick={() => handleNodeClick(node.id)}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          handleNodeClick(node.id);
                        }
                      }}
                    >
                      <h4>{node.data.label}</h4>
                      <p className={styles.nodeDesc}>Type: {node.data.stepKey}</p>
                    </div>
                    <button
                      className={styles.iconButton}
                      onClick={() => handleDeleteNode(node.id)}
                      title="Delete node"
                      aria-label="Delete node"
                    >
                      🗑
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right Panel - Configuration */}
        <aside className={styles.rightPanel}>
          <div className={styles.panelContent}>
            {!selectedNode ? (
              <div className={styles.panelEmpty}>
                <div className={styles.panelEmptyBox}>
                  <p className={styles.panelEmptyTitle}>Select a step to configure</p>
                  <p className={styles.panelEmptyText}>
                    Choose a step on the canvas to see and edit its details here.
                  </p>
                </div>
                <button className={styles.buttonSecondary}>Learn More</button>
              </div>
            ) : (
              <>
                <div className={styles.panelHeader}>
                  <h3 className={styles.panelTitle}>Configure Step</h3>
                </div>

                <div className={styles.formGroup}>
                  <label htmlFor="step-name-input">Step Name</label>
                  <input
                    id="step-name-input"
                    type="text"
                    className={styles.input}
                    value={selectedNode.data.label}
                    placeholder="Enter step name"
                    onChange={(e) => {
                      const updated = [...nodes];
                      const idx = updated.findIndex((n) => n.id === selectedNodeId);
                      if (idx >= 0) {
                        updated[idx].data.label = e.target.value;
                        setNodes(updated);
                        setIsUnsaved(true);
                      }
                    }}
                  />
                </div>

                {selectedNode.data.stepKey === 'validate' && (
                  <>
                    <div className={styles.formGroup}>
                      <label htmlFor="trigger-event-select">Trigger Event</label>
                      {/* Event selector: show helpful messages when events fail to load */}
                      {eventsLoading ? (
                        <div className={styles.infoText}>Loading events...</div>
                      ) : eventsError ? (
                        <div className={styles.errorText}>
                          Events failed to load. Make sure a tenant & datasource are selected (top-right) and you are authenticated.
                          <br />Check browser console / Network tab for /api/events errors.
                        </div>
                      ) : (
                        <select
                          id="trigger-event-select"
                          className={styles.select}
                          onChange={(e) => {
                            const updated = [...nodes];
                            const idx = updated.findIndex((n) => n.id === selectedNodeId);
                            if (idx >= 0) {
                              updated[idx].data.config = {
                                ...updated[idx].data.config,
                                eventId: e.target.value,
                              };
                              setNodes(updated);
                              setIsUnsaved(true);
                            }
                          }}
                        >
                          <option value="">Select an event...</option>
                          {events.map((evt: any) => (
                            <option key={evt.id} value={evt.id}>
                              {evt.label}
                            </option>
                          ))}
                        </select>
                      )}
                    </div>

                    <button
                      className={`${styles.buttonSecondary} ${styles.addRuleButton}`}
                      onClick={() => setShowRuleModal(true)}
                    >
                      + Add Validation Rule
                    </button>

                    {selectedNodeRules.length > 0 && (
                      <table className={styles.rulesTable}>
                        <thead>
                          <tr>
                            <th>Field</th>
                            <th>Operator</th>
                            <th>Message</th>
                            <th>Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          {selectedNodeRules.map((rule, idx: number) => {
                            const r = rule as Record<string, unknown>;
                            return (
                              <tr key={idx}>
                                <td>{String(r['field_label'] ?? '')}</td>
                                <td>{String(r['op_label'] ?? '')}</td>
                                <td>{String(r['message'] ?? '')}</td>
                                <td>
                                  <div className={styles.ruleActions}>
                                    <button
                                      type="button"
                                      className={`${styles.iconButton} ${styles.iconButtonDanger}`}
                                      onClick={() => {
                                        const updated = [...nodes];
                                        const nodeIdx = updated.findIndex(
                                          (n) => n.id === selectedNodeId
                                        );
                                        if (
                                          nodeIdx >= 0 &&
                                          updated[nodeIdx].data.config &&
                                          Array.isArray(updated[nodeIdx].data.config.rules)
                                        ) {
                                          const filteredRules = (
                                            updated[nodeIdx].data.config.rules as unknown[]
                                          ).filter((_, i: number) => i !== idx);
                                          updated[nodeIdx].data.config.rules = filteredRules;
                                          setNodes(updated);
                                          setIsUnsaved(true);
                                        }
                                      }}
                                      aria-label="Delete rule"
                                    >
                                      🗑
                                    </button>
                                  </div>
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    )}

                    {showRuleModal && (
                      <RuleBuilderModal
                        objects={businessObjects}
                        operators={operators}
                        onSave={(newRule) => {
                          const updated = [...nodes];
                          const idx = updated.findIndex((n) => n.id === selectedNodeId);
                          if (idx >= 0) {
                            const existing = updated[idx].data.config?.rules;
                            const rules = Array.isArray(existing) ? (existing as unknown[]) : [];
                            updated[idx].data.config = {
                              ...updated[idx].data.config,
                              rules: [...rules, newRule],
                            };
                            setNodes(updated);
                            setIsUnsaved(true);
                          }
                          setShowRuleModal(false);
                        }}
                        onCancel={() => setShowRuleModal(false)}
                      />
                    )}
                  </>
                )}
              </>
            )}
          </div>
        </aside>
      </main>

      {/* Footer - Event Triggers & Global Rules */}
      <footer className={styles.footer}>
        <div className={styles.footerContent}>
          <details className={styles.footerSection}>
            <summary className={styles.footerSummary}>
              <p className={styles.footerSummaryTitle}>Event Triggers</p>
              <span className={styles.footerSummaryIcon}>▼</span>
            </summary>
            <p className={styles.footerDetails}>
              Configure actions based on process events like "On Process Start" or "On Final
              Approval."
            </p>
          </details>

          <details className={styles.footerSection} open>
            <summary className={styles.footerSummary}>
              <p className={styles.footerSummaryTitle}>Global Validation Rules</p>
              <span className={styles.footerSummaryIcon}>▼</span>
            </summary>
            <p className={styles.footerDetails}>
              Ensure all documents are uploaded before the final approval step.
            </p>
          </details>
        </div>
      </footer>
    </div>
  );
};
