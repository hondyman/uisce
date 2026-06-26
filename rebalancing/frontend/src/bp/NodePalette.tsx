import React from "react";

export const NodePalette = () => {
  const onDragStart = (event: React.DragEvent, nodeType: string, label: string) => {
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData(
      "application/reactflow",
      JSON.stringify({ type: nodeType, label })
    );
  };

  return (
    <div className="node-palette" style={{ padding: '10px' }}>
      <div style={{ marginBottom: '10px', fontWeight: 'bold', color: '#333' }}>Components</div>
      <div
        draggable
        onDragStart={(e) => onDragStart(e, "task", "Activity")}
        className="palette-item"
        style={{ padding: '8px', border: '1px solid #ddd', marginBottom: '5px', borderRadius: '4px', cursor: 'grab', background: 'white' }}
      >
        📋 Task
      </div>
      <div
        draggable
        onDragStart={(e) => onDragStart(e, "approval", "Approval")}
        className="palette-item"
        style={{ padding: '8px', border: '1px solid #ddd', marginBottom: '5px', borderRadius: '4px', cursor: 'grab', background: 'white' }}
      >
        ✅ Approval
      </div>
      <div
        draggable
        onDragStart={(e) => onDragStart(e, "branch", "Branch")}
        className="palette-item"
        style={{ padding: '8px', border: '1px solid #ddd', marginBottom: '5px', borderRadius: '4px', cursor: 'grab', background: 'white' }}
      >
        🔀 Branch
      </div>
      <div
        draggable
        onDragStart={(e) => onDragStart(e, "delay", "Delay")}
        className="palette-item"
        style={{ padding: '8px', border: '1px solid #ddd', marginBottom: '5px', borderRadius: '4px', cursor: 'grab', background: 'white' }}
      >
        ⏱️ Delay
      </div>
    </div>
  );
};
