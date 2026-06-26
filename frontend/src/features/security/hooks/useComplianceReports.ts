import { useState, useEffect } from 'react';
import { ComplianceReport } from '../types/security';

// Mock data for compliance reports
const MOCK_REPORTS: ComplianceReport[] = [
    {
        id: 'rep_1',
        title: 'Q4 2025 SOC2 Audit Report',
        type: 'SOC2',
        status: 'published',
        created_at: '2025-12-15T10:00:00Z',
        created_by: 'admin',
        download_url: '/api/reports/download/rep_1'
    },
    {
        id: 'rep_2',
        title: 'Weekly Access Review',
        type: 'Internal',
        status: 'generated',
        created_at: '2025-12-28T09:30:00Z',
        created_by: 'security_analyst',
        download_url: '/api/reports/download/rep_2'
    },
    {
        id: 'rep_3',
        title: 'GDPR Data Processing Impact Assessment',
        type: 'GDPR',
        status: 'draft',
        created_at: '2025-12-30T14:15:00Z',
        created_by: 'dpo'
    }
];

export function useComplianceReports() {
    const [reports, setReports] = useState<ComplianceReport[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    const fetchReports = async () => {
        setLoading(true);
        try {
            // Simulate API call
            await new Promise(resolve => setTimeout(resolve, 800));
            setReports(MOCK_REPORTS);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err : new Error('Failed to fetch reports'));
        } finally {
            setLoading(false);
        }
    };

    const generateReport = async (type: string, title: string) => {
        setLoading(true);
        try {
            await new Promise(resolve => setTimeout(resolve, 1500));
            const newReport: ComplianceReport = {
                id: `rep_${Date.now()}`,
                title,
                type: type as any,
                status: 'generated',
                created_at: new Date().toISOString(),
                created_by: 'current_user'
            };
            setReports(prev => [newReport, ...prev]);
        } catch (err) {
            setError(err instanceof Error ? err : new Error('Failed to generate report'));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchReports();
    }, []);

    return {
        reports,
        loading,
        error,
        fetchReports,
        generateReport
    };
}
