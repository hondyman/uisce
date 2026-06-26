import React, { useState, useEffect } from 'react';
import { useConfirm } from '../components/ConfirmProvider';
import { Plus, Download, Trash2, RefreshCw, Search, Filter, Eye } from 'lucide-react';
import { HouseholdReportBuilder } from '../components/HouseholdReportBuilder';
import { devDebug } from '../utils/devLogger';

// ============================================================================
// TYPES
// ============================================================================

interface Household {
  id: string;
  name: string;
  description?: string;
  householdType: string;
  status: string;
}

interface HouseholdReport {
  id: string;
  householdId: string;
  reportName: string;
  reportType: string;
  status: string;
  pageCount: number;
  generatedAt?: string;
  createdAt: string;
}

interface SemanticView {
  id: string;
  name: string;
  description?: string;
  entity_count?: number;
}

// ============================================================================
// PAGE COMPONENT
// ============================================================================

export const HouseholdReportsPage: React.FC = () => {
  // ========================================================================
  // STATE
  // ========================================================================

  const [activeTab, setActiveTab] = useState<'reports' | 'builder'>('reports');
  const [households, setHouseholds] = useState<Household[]>([]);
  const [reports, setReports] = useState<HouseholdReport[]>([]);
  const [semanticViews, setSemanticViews] = useState<SemanticView[]>([]);
  const [selectedHousehold, setSelectedHousehold] = useState<string>('');
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [showBuilder, setShowBuilder] = useState(false);
  const [previewReport, setPreviewReport] = useState<HouseholdReport | null>(null);

  // ========================================================================
  // EFFECTS
  // ========================================================================

  useEffect(() => {
    loadHouseholds();
    loadSemanticViews();
  }, []);

  useEffect(() => {
    if (selectedHousehold) {
      loadReports();
    }
  }, [selectedHousehold]);

  // ========================================================================
  // DATA LOADING
  // ========================================================================

  const loadHouseholds = async () => {
    setIsLoading(true);
    try {
      // In real app, fetch from API
      // const response = await fetch('/api/households');
      // const data = await response.json();
      // setHouseholds(data.households);
      
      // Mock data for demo
      setHouseholds([
        {
          id: '1',
          name: 'Smith Family Office',
          description: 'Primary family holding company',
          householdType: 'family',
          status: 'active',
        },
        {
          id: '2',
          name: 'Johnson Trust',
          description: 'Charitable giving vehicle',
          householdType: 'trust',
          status: 'active',
        },
      ]);
    } catch (error) {
      console.error('Error loading households:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const loadReports = async () => {
    if (!selectedHousehold) return;

    setIsLoading(true);
    try {
      // In real app: fetch `/api/reports/household?household_id=${selectedHousehold}`
      // Mock data for demo
      setReports([
        {
          id: '1',
          householdId: selectedHousehold,
          reportName: 'Q4 Holdings Summary',
          reportType: 'summary',
          status: 'generated',
          pageCount: 1,
          generatedAt: new Date(Date.now() - 86400000).toISOString(),
          createdAt: new Date(Date.now() - 172800000).toISOString(),
        },
        {
          id: '2',
          householdId: selectedHousehold,
          reportName: 'Detailed Asset Breakdown',
          reportType: 'detailed',
          status: 'generated',
          pageCount: 5,
          generatedAt: new Date().toISOString(),
          createdAt: new Date().toISOString(),
        },
      ]);
    } catch (error) {
      console.error('Error loading reports:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const loadSemanticViews = async () => {
    try {
      // In real app: fetch from GraphQL/Hasura
      // Mock data
      setSemanticViews([
        { id: '1', name: 'Holdings View', entity_count: 152 },
        { id: '2', name: 'Performance View', entity_count: 45 },
        { id: '3', name: 'Allocations View', entity_count: 28 },
      ]);
    } catch (error) {
      console.error('Error loading semantic views:', error);
    }
  };

  // ========================================================================
  // ACTIONS
  // ========================================================================

  const handleSaveReport = async (config: any) => {
    try {
      // In real app: POST to `/api/reports/household`
      devDebug('Saving report:', config);
      
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 500));
      
      setShowBuilder(false);
      setActiveTab('reports');
      loadReports();
    } catch (error) {
      console.error('Error saving report:', error);
    }
  };

  const handleDeleteReport = async (reportId: string) => {
    const confirm = useConfirm();
    if (!(await confirm({ title: 'Delete report', description: 'Are you sure you want to delete this report?' }))) return;

    try {
      // In real app: DELETE `/api/reports/household/${reportId}`
      devDebug('Deleting report:', reportId);
      
      setReports((prev) => prev.filter((r) => r.id !== reportId));
    } catch (error) {
      console.error('Error deleting report:', error);
    }
  };

  const handleDownloadPDF = async (reportId: string) => {
    try {
      // In real app: GET `/api/reports/household/${reportId}/pdf`
      devDebug('Downloading PDF for report:', reportId);
      
      // Simulate download
      const link = document.createElement('a');
      link.href = '#';
      link.download = `report_${reportId}.pdf`;
      link.click();
    } catch (error) {
      console.error('Error downloading PDF:', error);
    }
  };

  // ========================================================================
  // FILTERING & SEARCHING
  // ========================================================================

  const filteredReports = reports.filter((report) => {
    const matchesSearch =
      report.reportName.toLowerCase().includes(searchTerm.toLowerCase()) ||
      report.reportType.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesFilter = !filterType || report.reportType === filterType;
    return matchesSearch && matchesFilter;
  });

  const reportTypes = Array.from(new Set(reports.map((r) => r.reportType)));

  // ========================================================================
  // RENDER
  // ========================================================================

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      {/* Header */}
      <div className="bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold text-slate-900 dark:text-white">
                Household Reports
              </h1>
              <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                Generate and manage household-scoped reports with AI semantic cubes
              </p>
            </div>

            <button
              onClick={() => {
                setShowBuilder(true);
                setActiveTab('builder');
              }}
              className="flex items-center space-x-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors self-start sm:self-auto"
            >
              <Plus size={18} />
              <span>New Report</span>
            </button>
          </div>

          {/* Tabs */}
          <div className="mt-6 border-b border-slate-200 dark:border-slate-800 flex space-x-8">
            <button
              onClick={() => setActiveTab('reports')}
              className={`py-2 px-1 border-b-2 font-medium text-sm transition-colors ${
                activeTab === 'reports'
                  ? 'border-blue-600 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-300'
              }`}
            >
              Reports
            </button>
            {showBuilder && (
              <button
                onClick={() => setActiveTab('builder')}
                className={`py-2 px-1 border-b-2 font-medium text-sm transition-colors ${
                  activeTab === 'builder'
                    ? 'border-blue-600 text-blue-600 dark:text-blue-400'
                    : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-300'
                }`}
              >
                Builder
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === 'builder' && showBuilder ? (
          <div>
            <HouseholdReportBuilder
              onSave={handleSaveReport}
              households={households}
              semanticViews={semanticViews}
            />
          </div>
        ) : (
          <div className="space-y-6">
            {/* Household Selector */}
            <div className="bg-white dark:bg-slate-900 p-6 rounded-lg shadow border border-slate-200 dark:border-slate-800">
              <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">
                Select Household
              </label>
              <select
                value={selectedHousehold}
                onChange={(e) => setSelectedHousehold(e.target.value)}
                aria-label="Select household"
                className="w-full md:w-96 px-4 py-2 rounded-lg border border-slate-300 dark:bg-slate-800 dark:text-white dark:border-slate-600 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:outline-none"
              >
                <option value="">-- Choose a household --</option>
                {households.map((h) => (
                  <option key={h.id} value={h.id}>
                    {h.name} ({h.householdType})
                  </option>
                ))}
              </select>
            </div>

            {/* Reports Section */}
            {selectedHousehold && (
              <div className="space-y-6">
                {/* Controls */}
                <div className="flex flex-col md:flex-row gap-4 items-start md:items-end">
                  {/* Search */}
                  <div className="flex-1 relative">
                    <Search className="absolute left-3 top-3 text-slate-400" size={18} />
                    <input
                      type="text"
                      placeholder="Search reports..."
                      value={searchTerm}
                      onChange={(e) => setSearchTerm(e.target.value)}
                      className="w-full pl-10 pr-4 py-2 rounded-lg border border-slate-300 dark:bg-slate-800 dark:text-white dark:border-slate-600 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                    />
                  </div>

                  {/* Filter */}
                  <div className="flex items-center space-x-2">
                    <Filter size={18} className="text-slate-600 dark:text-slate-400" />
                    <select
                      value={filterType}
                      onChange={(e) => setFilterType(e.target.value)}
                      aria-label="Filter by report type"
                      className="px-4 py-2 rounded-lg border border-slate-300 dark:bg-slate-800 dark:text-white dark:border-slate-600 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                    >
                      <option value="">All Types</option>
                      {reportTypes.map((type) => (
                        <option key={type} value={type}>
                          {type.charAt(0).toUpperCase() + type.slice(1)}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Refresh */}
                  <button
                    onClick={() => loadReports()}
                    disabled={isLoading}
                    className="flex items-center space-x-2 px-4 py-2 text-slate-700 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
                  >
                    <RefreshCw size={18} className={isLoading ? 'animate-spin' : ''} />
                    <span>Refresh</span>
                  </button>
                </div>

                {/* Reports List */}
                {filteredReports.length > 0 ? (
                  <div className="space-y-4">
                    {filteredReports.map((report) => (
                      <div
                        key={report.id}
                        className="bg-white dark:bg-slate-900 p-6 rounded-lg shadow border border-slate-200 dark:border-slate-800 hover:shadow-lg transition-shadow"
                      >
                        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                          <div className="flex-1">
                            <div className="flex items-center space-x-3">
                              <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                                {report.reportName}
                              </h3>
                              <span className="px-2 py-1 text-xs font-medium rounded-full bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-200">
                                {report.reportType}
                              </span>
                              <span
                                className={`px-2 py-1 text-xs font-medium rounded-full ${
                                  report.status === 'generated'
                                    ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-200'
                                    : 'bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-200'
                                }`}
                              >
                                {report.status}
                              </span>
                            </div>

                            <div className="mt-2 text-sm text-slate-600 dark:text-slate-400 space-y-1">
                              <p>Pages: {report.pageCount}</p>
                              <p>
                                Created:{' '}
                                {new Date(report.createdAt).toLocaleDateString()}
                              </p>
                              {report.generatedAt && (
                                <p>
                                  Generated:{' '}
                                  {new Date(report.generatedAt).toLocaleDateString()}
                                </p>
                              )}
                            </div>
                          </div>

                          <div className="flex flex-wrap gap-2 justify-end md:justify-start">
                            <button
                              onClick={() => setPreviewReport(report)}
                              className="flex items-center space-x-1 px-3 py-2 text-sm bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg transition-colors"
                            >
                              <Eye size={16} />
                              <span>Preview</span>
                            </button>

                            {report.status === 'generated' && (
                              <button
                                onClick={() => handleDownloadPDF(report.id)}
                                className="flex items-center space-x-1 px-3 py-2 text-sm bg-green-100 dark:bg-green-900 hover:bg-green-200 dark:hover:bg-green-800 text-green-700 dark:text-green-200 rounded-lg transition-colors"
                              >
                                <Download size={16} />
                                <span>PDF</span>
                              </button>
                            )}

                            <button
                              onClick={() => handleDeleteReport(report.id)}
                              className="flex items-center space-x-1 px-3 py-2 text-sm bg-red-100 dark:bg-red-900 hover:bg-red-200 dark:hover:bg-red-800 text-red-700 dark:text-red-200 rounded-lg transition-colors"
                            >
                              <Trash2 size={16} />
                              <span>Delete</span>
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-12 bg-white dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-800">
                    <p className="text-slate-600 dark:text-slate-400 mb-4">
                      {searchTerm || filterType ? 'No reports match your filter' : 'No reports yet'}
                    </p>
                    <button
                      onClick={() => {
                        setShowBuilder(true);
                        setActiveTab('builder');
                      }}
                      className="inline-flex items-center space-x-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
                    >
                      <Plus size={18} />
                      <span>Create First Report</span>
                    </button>
                  </div>
                )}
              </div>
            )}

            {!selectedHousehold && (
              <div className="text-center py-12 bg-white dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-800">
                <p className="text-slate-600 dark:text-slate-400">
                  Select a household above to view its reports
                </p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Preview Modal */}
      {previewReport && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-slate-900 rounded-lg shadow-xl max-w-2xl w-full max-h-[80vh] overflow-y-auto">
            <div className="sticky top-0 bg-slate-100 dark:bg-slate-800 p-4 border-b border-slate-200 dark:border-slate-700 flex justify-between items-center">
              <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                {previewReport.reportName}
              </h2>
              <button
                onClick={() => setPreviewReport(null)}
                className="text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white text-xl"
              >
                ✕
              </button>
            </div>

            <div className="p-6 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                    Report Type
                  </p>
                  <p className="text-slate-900 dark:text-white">
                    {previewReport.reportType}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                    Status
                  </p>
                  <p className="text-slate-900 dark:text-white">
                    {previewReport.status}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                    Pages
                  </p>
                  <p className="text-slate-900 dark:text-white">
                    {previewReport.pageCount}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                    Created
                  </p>
                  <p className="text-slate-900 dark:text-white">
                    {new Date(previewReport.createdAt).toLocaleDateString()}
                  </p>
                </div>
              </div>

              <div className="text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-800 p-3 rounded">
                Full report details would be displayed here. In production, this would show
                the generated report structure, drill-down information, and semantic cube data.
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default HouseholdReportsPage;