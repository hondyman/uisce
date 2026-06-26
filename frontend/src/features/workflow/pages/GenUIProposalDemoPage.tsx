import React from "react";
import { ProposalCard } from "../../../genui/components/ProposalCard";
import { mockRebalanceProposal } from "../../../genui/mocks";

export const GenUIProposalDemoPage: React.FC = () => {
  const handleApprove = () => {
    alert("Approved! In a real app, this would trigger the saga execution.");
  };

  const handleReject = () => {
    alert("Rejected. Feedback logged to UAR.");
  };

  const handleClarify = () => {
    alert("Clarification requested. GenUI would open a chat interface.");
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 p-8">
      <div className="max-w-3xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">GenUI Rebalancing Proposal Demo</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-2">
            This page demonstrates the GenUI-generated rebalancing proposal card.
            In production, this component is hydrated via the GenUI layout engine based on the "rebalancing_proposal" intent.
          </p>
        </div>

        <ProposalCard
          data={mockRebalanceProposal}
          onApprove={handleApprove}
          onReject={handleReject}
          onClarify={handleClarify}
        />
      </div>
    </div>
  );
};
