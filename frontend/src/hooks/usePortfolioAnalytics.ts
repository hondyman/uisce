import { useState, useEffect } from 'react';

export interface SecurityMasterRecord {
    id: string;
    security_id: string;
    name: string;
    ticker?: string;
    isin?: string;
    cusip?: string;
    sedol?: string;
    asset_class: string;
    sector?: string;
    industry?: string;
    region?: string;
    country?: string;
    currency: string;
    last_price?: number;
}

export interface SecurityPosition {
    id: string;
    portfolio_id: string;
    security_id: string;
    security: SecurityMasterRecord;
    quantity: number;
    cost_basis: number;
    market_value: number;
    weight: number;
    confidence: number;
    source_systems: string[];
}

export interface PortfolioAnalytics {
    portfolio_id: string;
    portfolio_name: string;
    portfolio_code: string;
    base_currency: string;
    total_value: number;
    total_positions: number;
    confidence_score: number;
    asset_class_breakdown: Record<string, number>;
    sector_exposure: Record<string, number>;
    region_exposure: Record<string, number>;
    currency_exposure: Record<string, number>;
    top_holdings: any[];
}

export interface LineageEdge {
    source: string;
    target: string;
    type: string;
    label: string;
    properties?: Record<string, any>;
}

export interface DataLineage {
    nodes: any[];
    edges: LineageEdge[];
}

export const usePortfolioAnalytics = (portfolioId: string | null) => {
    const [analytics, setAnalytics] = useState<PortfolioAnalytics | null>(null);
    const [lineage, setLineage] = useState<DataLineage | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const fetchAnalytics = async () => {
        if (!portfolioId) return;
        setLoading(true);
        try {
            const response = await fetch(`/api/v1/portfolio/analytics/${portfolioId}`, {
                headers: {
                    'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' // Default tenant for now
                }
            });
            if (!response.ok) throw new Error('Failed to fetch analytics');
            const data = await response.json();
            setAnalytics(data);
        } catch (err: any) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    const fetchSecurityLineage = async (securityId: string) => {
        try {
            const response = await fetch(`/api/v1/security/${securityId}/lineage`, {
                headers: {
                    'X-Tenant-ID': '00000000-0000-0000-0000-000000000001'
                }
            });
            if (!response.ok) throw new Error('Failed to fetch lineage');
            const data = await response.json();
            setLineage(data);
            return data;
        } catch (err: any) {
            setError(err.message);
            return null;
        }
    };

    useEffect(() => {
        if (portfolioId) {
            fetchAnalytics();
        }
    }, [portfolioId]);

    return {
        analytics,
        lineage,
        loading,
        error,
        refresh: fetchAnalytics,
        getSecurityLineage: fetchSecurityLineage
    };
};
