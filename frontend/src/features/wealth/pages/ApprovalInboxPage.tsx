import React, { useState } from 'react';
import { usePendingApprovals, useApproveRequest, useRejectRequest, ApprovalRequest } from '../api/approvals';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Loader2, CheckCircle, XCircle, Clock } from 'lucide-react';
import { useNotification } from '../../../hooks/useNotification';

export const ApprovalInboxPage: React.FC = () => {
  const { data: approvals, isLoading, error: loadError } = usePendingApprovals();
  const approveMutation = useApproveRequest();
  const rejectMutation = useRejectRequest();
  const { success, error: showError } = useNotification();
  const [comment, setComment] = useState<Record<string, string>>({});

  const handleApprove = async (approval: ApprovalRequest) => {
    try {
      await approveMutation.mutateAsync({
        id: approval.id,
        data: {
          approver_id: 'current-user', // In reality, get from auth context
          comment: comment[approval.id] || 'Approved',
        },
      });
      success('Request approved successfully');
      setComment(prev => ({ ...prev, [approval.id]: '' }));
    } catch (err) {
      showError(`Failed to approve: ${err}`);
    }
  };

  const handleReject = async (approval: ApprovalRequest) => {
    try {
      await rejectMutation.mutateAsync({
        id: approval.id,
        data: {
          approver_id: 'current-user',
          comment: comment[approval.id] || 'Rejected',
        },
      });
      success('Request rejected');
      setComment(prev => ({ ...prev, [approval.id]: '' }));
    } catch (err) {
      showError(`Failed to reject: ${err}`);
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (loadError) {
    return (
      <div className="text-red-500 p-4 border border-red-200 rounded bg-red-50">
        Failed to load approvals: {loadError.message}
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto p-4">
      <h1 className="text-2xl font-bold mb-6">Approval Inbox</h1>
      
      {approvals?.length === 0 ? (
        <div className="text-center text-gray-500 p-8">
          <Clock className="h-12 w-12 mx-auto mb-4 text-gray-300" />
          <p>No pending approvals</p>
        </div>
      ) : (
        <div className="space-y-4">
          {approvals?.map((approval) => (
            <Card key={approval.id} className="border-l-4 border-l-yellow-500">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Clock className="h-5 w-5 text-yellow-500" />
                  {formatActionType(approval.action_type)}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div><span className="font-semibold">Client ID:</span> {approval.client_id}</div>
                  <div><span className="font-semibold">Requested:</span> {new Date(approval.created_at).toLocaleString()}</div>
                  <div><span className="font-semibold">Workflow ID:</span> {approval.workflow_id}</div>
                  {Object.keys(approval.action_details || {}).length > 0 && (
                    <div className="mt-2 ">
                      <span className="font-semibold">Details:</span>
                      <pre className="mt-1 p-2 bg-gray-50 rounded text-xs overflow-auto">
                        {JSON.stringify(approval.action_details, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
                <div className="mt-4">
                  <label className="block text-sm font-medium mb-1">Comment</label>
                  <input
                    type="text"
                    className="w-full px-3 py-2 border rounded"
                    placeholder="Add a comment..."
                    value={comment[approval.id] || ''}
                    onChange={(e) => setComment(prev => ({ ...prev, [approval.id]: e.target.value }))}
                  />
                </div>
              </CardContent>
              <CardFooter className="flex gap-2">
                <Button
                  onClick={() => handleApprove(approval)}
                  disabled={approveMutation.isPending}
                  className="bg-green-600 hover:bg-green-700"
                >
                  <CheckCircle className="h-4 w-4 mr-2" />
                  Approve
                </Button>
                <Button
                  onClick={() => handleReject(approval)}
                  disabled={rejectMutation.isPending}
                  variant="destructive"
                >
                  <XCircle className="h-4 w-4 mr-2" />
                  Reject
                </Button>
              </CardFooter>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};

function formatActionType(actionType: string): string {
  const mapping: Record<string, string> = {
    tax_loss_harvest: 'Tax Loss Harvesting Request',
    rebalance: 'Portfolio Rebalancing Request',
  };
  return mapping[actionType] || actionType;
}
