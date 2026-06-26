import React from 'react';
import { FeedItem } from '../api/feed';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Lightbulb, Zap, Newspaper } from 'lucide-react';
import { useExecuteFeedAction } from '../../wealth/api/approvals';
import { useNotification } from '../../../hooks/useNotification';

interface FeedCardProps {
  item: FeedItem;
}

export const FeedCard: React.FC<FeedCardProps> = ({ item }) => {
  const executeMutation = useExecuteFeedAction();
  const { success, error } = useNotification();

  const getIcon = () => {
    switch (item.type) {
      case 'insight':
        return <Lightbulb className="h-5 w-5 text-yellow-500" />;
      case 'action':
        return <Zap className="h-5 w-5 text-blue-500" />;
      case 'news':
        return <Newspaper className="h-5 w-5 text-gray-500" />;
      default:
        return null;
    }
  };

  const handleActionClick = async () => {
    try {
      const response = await executeMutation.mutateAsync({
        cardId: item.card_id,
        clientId: 'c_12345', // In reality, get from context
        tenantId: 'default',
        actionDetails: item.data || {},
      });
      success(`Workflow started: ${response.workflow_id}`);
    } catch (err) {
      error(`Failed to start action: ${err}`);
    }
  };

  return (
    <Card className="mb-4 hover:shadow-md transition-shadow">
      <CardHeader className="flex flex-row items-center gap-4 pb-2">
        {getIcon()}
        <CardTitle className="text-lg font-semibold">{item.title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-gray-600">{item.content}</p>
      </CardContent>
      {item.action_label && (
        <CardFooter>
          <Button
            variant="default"
            size="sm"
            onClick={handleActionClick}
            disabled={executeMutation.isPending}
          >
            {executeMutation.isPending ? 'Starting...' : item.action_label}
          </Button>
        </CardFooter>
      )}
    </Card>
  );
};
