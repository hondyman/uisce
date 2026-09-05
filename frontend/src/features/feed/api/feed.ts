import { useQuery } from '@tanstack/react-query';
import { apiGet } from '../../../utils/api';

export interface FeedItem {
    card_id: string;
    title: string;
    content: string;
    type: 'action' | 'insight' | 'news';
    score: number;
    action_workflow_id?: string;
    action_label?: string;
    data?: Record<string, any>;
}

export const fetchFeed = async (): Promise<FeedItem[]> => {
    const response = await apiGet('wealth/feed');
    if (!response || typeof response !== 'object') return [];
    if (Array.isArray(response)) return response;
    if (response && typeof response === 'object' && 'data' in response && Array.isArray((response as any).data)) {
        return (response as any).data;
    }
    return [];
};

export const useFeed = () => {
    return useQuery({
        queryKey: ['wealth-feed'],
        queryFn: fetchFeed,
        refetchInterval: 30000, // Poll every 30 seconds for new signals
    });
};
