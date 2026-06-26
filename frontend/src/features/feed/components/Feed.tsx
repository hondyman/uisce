import React from 'react';
import { useFeed, FeedItem } from '../api/feed';
import { FeedCard } from './FeedCard';
import { Loader2 } from 'lucide-react';

export const Feed: React.FC = () => {
  const { data: feedItems, isLoading, error } = useFeed();

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-red-500 p-4 border border-red-200 rounded bg-red-50">
        Failed to load feed: {error.message}
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto p-4">
      <h1 className="text-2xl font-bold mb-6">WealthStream Feed</h1>
      {feedItems?.length === 0 ? (
        <p className="text-gray-500 text-center">No updates at this time.</p>
      ) : (
        feedItems?.map((item: FeedItem) => <FeedCard key={item.card_id} item={item} />)
      )}
    </div>
  );
};
