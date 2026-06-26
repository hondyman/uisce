import { useState, useCallback } from 'react';
import { useNotification } from './useNotification';

interface UseChartRefreshOptions {
    datasourceId: string;
    onSuccess?: () => void;
}

interface ChartRefreshResult {
    refreshCharts: () => Promise<void>;
    isRefreshing: boolean;
    error: string | null;
}

/**
 * Custom hook to handle chart refresh operations
 * Calls the backend API to regenerate all charts for a datasource
 */
export const useChartRefresh = ({
    datasourceId,
    onSuccess
}: UseChartRefreshOptions): ChartRefreshResult => {
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const notification = useNotification();

    const refreshCharts = useCallback(async () => {
        if (!datasourceId) {
            notification.error('Datasource ID is required');
            return;
        }

        setIsRefreshing(true);
        setError(null);

        try {
            const response = await fetch(`/api/charts/${datasourceId}/refresh`, {
                method: 'POST',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(
                    errorData.error || `Failed to refresh charts: ${response.statusText}`
                );
            }

            const result = await response.json();

            if (!result.success) {
                throw new Error(result.error || 'Chart refresh failed');
            }

            notification.success('Charts regenerated successfully');

            // Call success callback to trigger data refetch
            if (onSuccess) {
                onSuccess();
            }
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
            setError(errorMessage);
            notification.error(`Failed to regenerate charts: ${errorMessage}`);
            console.error('Chart refresh error:', err);
        } finally {
            setIsRefreshing(false);
        }
    }, [datasourceId, onSuccess, notification]);

    return {
        refreshCharts,
        isRefreshing,
        error,
    };
};
