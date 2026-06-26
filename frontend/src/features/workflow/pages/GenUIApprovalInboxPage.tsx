import React, { useState } from 'react';
import { GenUIRenderer } from '../../../genui/Renderer';
import { useGenUIQuery } from '../../../genui/hooks';

export const GenUIApprovalInboxPage: React.FC = () => {
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string | null>(null);
  const { data: layout, loading, error } = useGenUIQuery('show my pending approvals');

  const handleFormSubmit = async (formId: string, data: Record<string, any>) => {
    if (formId === 'approval_form' && selectedWorkflowId) {
      try {
        const response = await fetch('/api/genui/approvals/signal', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            workflow_id: selectedWorkflowId,
            approved: data.action === 'approve',
            rationale: data.rationale,
          }),
        });

        if (!response.ok) throw new Error('Failed to send approval signal');
        window.location.reload();
      } catch (err) {
        console.error('Approval action failed:', err);
        alert('Failed to submit approval decision. Please try again.');
      }
    }
  };

  if (loading) return <div className="flex items-center justify-center min-h-screen"><p>Loading...</p></div>;
  if (error) return <div className="flex items-center justify-center min-h-screen"><p>Error: {error}</p></div>;

  return (
    <div className="min-h-screen bg-background-light dark:bg-background-dark p-6">
      <h1 className="text-4xl font-black mb-6">Approval Inbox (GenUI)</h1>
      {layout && <GenUIRenderer layout={layout} onFormSubmit={handleFormSubmit} />}
    </div>
  );
};
