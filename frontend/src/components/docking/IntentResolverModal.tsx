import React from 'react';
import { IntentTarget, StandardIntent } from '../../services/fdc3/intentTypes';
import { Fdc3Context } from '../../services/fdc3/types';

interface IntentResolverModalProps {
  isOpen: boolean;
  intent: StandardIntent | null;
  context: Fdc3Context | null;
  targets: IntentTarget[];
  onSelect: (target: IntentTarget) => void;
  onCancel: () => void;
}

export const IntentResolverModal: React.FC<IntentResolverModalProps> = ({
  isOpen,
  intent,
  context,
  targets,
  onSelect,
  onCancel,
}) => {
  if (!isOpen || !intent) return null;

  const contextLabel = context?.name || context?.id?.ticker || context?.id?.orderId || context?.type || 'Context payload';

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.75)',
        backdropFilter: 'blur(4px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 99999,
      }}
    >
      <div
        style={{
          width: '440px',
          maxWidth: '90vw',
          backgroundColor: '#091428',
          border: '1px solid #1e293b',
          borderRadius: '8px',
          boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.5)',
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          color: '#e2e8f0',
          fontFamily: 'system-ui, -apple-system, sans-serif',
        }}
      >
        {/* Header */}
        <div
          style={{
            padding: '12px 16px',
            borderBottom: '1px solid #1e293b',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            background: '#0a192f',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{ fontSize: '13px', fontWeight: 700, color: '#38bdf8', letterSpacing: '0.05em' }}>
              INTENT RESOLUTION
            </span>
            <span
              style={{
                fontSize: '11px',
                padding: '2px 6px',
                borderRadius: '4px',
                backgroundColor: '#0284c720',
                border: '1px solid #0284c750',
                color: '#38bdf8',
                fontWeight: 600,
              }}
            >
              {intent}
            </span>
          </div>
          <button
            onClick={onCancel}
            style={{
              background: 'transparent',
              border: 'none',
              color: '#64748b',
              fontSize: '18px',
              cursor: 'pointer',
              lineHeight: 1,
            }}
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div style={{ padding: '16px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <div style={{ fontSize: '12px', color: '#94a3b8' }}>
            Multiple active views can handle this intent with context <strong style={{ color: '#e2e8f0' }}>{contextLabel}</strong>. Select destination target:
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '240px', overflowY: 'auto' }}>
            {targets.map((target) => (
              <button
                key={`${target.windowId}:${target.viewId}`}
                onClick={() => onSelect(target)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '10px 14px',
                  backgroundColor: '#0f172a',
                  border: '1px solid #1e293b',
                  borderRadius: '6px',
                  color: '#f8fafc',
                  cursor: 'pointer',
                  textAlign: 'left',
                  transition: 'background 0.15s, border-color 0.15s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.backgroundColor = '#1e293b';
                  e.currentTarget.style.borderColor = '#38bdf8';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = '#0f172a';
                  e.currentTarget.style.borderColor = '#1e293b';
                }}
              >
                <div>
                  <div style={{ fontSize: '13px', fontWeight: 600, color: '#f1f5f9' }}>{target.title}</div>
                  <div style={{ fontSize: '11px', color: '#64748b', marginTop: '2px' }}>
                    View: {target.viewId} | Window: {target.windowId}
                  </div>
                </div>
                <span style={{ fontSize: '12px', color: '#38bdf8', fontWeight: 600 }}>Route →</span>
              </button>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div
          style={{
            padding: '10px 16px',
            borderTop: '1px solid #1e293b',
            display: 'flex',
            justifyContent: 'flex-end',
            background: '#07101e',
          }}
        >
          <button
            onClick={onCancel}
            style={{
              padding: '6px 14px',
              backgroundColor: '#1e293b',
              border: '1px solid #334155',
              borderRadius: '4px',
              color: '#94a3b8',
              fontSize: '12px',
              fontWeight: 500,
              cursor: 'pointer',
            }}
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
};
