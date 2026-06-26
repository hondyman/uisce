import { useState, useRef, useCallback } from 'react';

/**
 * usePageMetrics
 * Collects and surfaces real-time performance indicators for the active page.
 */
export function usePageMetrics(pageId: string, tenantId: string) {
    const [metrics, setMetrics] = useState({
        renderTimeMs: 0,
        apiLatencyAvg: 0,
        dataFanout: 0,
        errorCount: 0,
        sloStatus: 'healthy' as 'healthy' | 'warning' | 'violated'
    });

    const startTime = useRef(performance.now());
    const apiCalls = useRef<number>(0);
    const apiErrors = useRef<number>(0);

    const registerApiCall = useCallback(() => {
        apiCalls.current += 1;
    }, []);

    const reportError = useCallback((err: any) => {
        apiErrors.current += 1;
        console.error("Page metric error:", err);
    }, []);

    const reportRenderComplete = useCallback(() => {
        const endTime = performance.now();
        setMetrics(prev => ({
            ...prev,
            renderTimeMs: Math.round(endTime - startTime.current),
            dataFanout: apiCalls.current,
            errorCount: apiErrors.current,
            sloStatus: apiErrors.current > 0 ? 'violated' : 'healthy'
        }));
    }, []);

    return {
        ...metrics,
        registerApiCall,
        reportRenderComplete,
        reportError
    };
}
