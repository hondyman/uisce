/**
 * RegulatoryDeadlineIntelligence.tsx
 * 
 * Regulatory Deadline Intelligence with Penalty Risk Scoring:
 * - SEC, FINRA, state regulatory deadline tracking
 * - Penalty risk scoring based on AUM, violation history
 * - Automated reminder workflows with escalation
 * - Compliance calendar integration
 */

import React, { useState, useMemo } from 'react';
import {
  Calendar,
  AlertTriangle,
  Clock,
  Shield,
  DollarSign,
  Bell,
  CheckCircle,
  Filter,
  Search,
  ChevronRight,
  TrendingUp,
  FileText,
  AlertCircle,
  Settings
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface RegulatoryDeadline {
  id: string;
  name: string;
  regulatoryBody: RegulatoryBody;
  filingType: string;
  description: string;
  dueDate: Date;
  status: DeadlineStatus;
  penaltyRiskScore: number; // 0-100
  estimatedPenalty: number;
  affectedAccounts: number;
  affectedAUM: number;
  assignedTo: string[];
  dependencies: string[];
  completionProgress: number;
  notes?: string;
  reminderSchedule: ReminderSchedule[];
  history: DeadlineHistoryEntry[];
}

type RegulatoryBody = 'SEC' | 'FINRA' | 'STATE' | 'DOL' | 'IRS' | 'CFTC' | 'FED';

type DeadlineStatus = 
  | 'UPCOMING'
  | 'IN_PROGRESS'
  | 'UNDER_REVIEW'
  | 'SUBMITTED'
  | 'APPROVED'
  | 'OVERDUE'
  | 'EXTENDED';

interface ReminderSchedule {
  daysBeforeDue: number;
  recipientType: 'ASSIGNEE' | 'MANAGER' | 'COMPLIANCE_OFFICER' | 'EXECUTIVE';
  notificationType: 'EMAIL' | 'SMS' | 'SYSTEM' | 'ALL';
  sent: boolean;
  sentAt?: Date;
}

interface DeadlineHistoryEntry {
  id: string;
  action: string;
  user: string;
  timestamp: Date;
  details?: string;
}

interface PenaltyCalculation {
  baseAmount: number;
  aumMultiplier: number;
  historyMultiplier: number;
  daysOverdueMultiplier: number;
  totalEstimate: number;
  factors: PenaltyFactor[];
}

interface PenaltyFactor {
  name: string;
  impact: 'INCREASE' | 'DECREASE' | 'NEUTRAL';
  description: string;
  weight: number;
}

// ============================================================================
// Constants
// ============================================================================

const REGULATORY_BODY_CONFIG: Record<RegulatoryBody, { 
  label: string; 
  color: string; 
  bgColor: string;
  icon: string;
}> = {
  SEC: { label: 'SEC', color: 'text-blue-700', bgColor: 'bg-blue-100', icon: '🏛️' },
  FINRA: { label: 'FINRA', color: 'text-purple-700', bgColor: 'bg-purple-100', icon: '📊' },
  STATE: { label: 'State', color: 'text-green-700', bgColor: 'bg-green-100', icon: '🏢' },
  DOL: { label: 'DOL', color: 'text-orange-700', bgColor: 'bg-orange-100', icon: '👷' },
  IRS: { label: 'IRS', color: 'text-red-700', bgColor: 'bg-red-100', icon: '💰' },
  CFTC: { label: 'CFTC', color: 'text-cyan-700', bgColor: 'bg-cyan-100', icon: '📈' },
  FED: { label: 'Federal Reserve', color: 'text-indigo-700', bgColor: 'bg-indigo-100', icon: '🏦' }
};

const STATUS_CONFIG: Record<DeadlineStatus, { label: string; color: string; bgColor: string }> = {
  UPCOMING: { label: 'Upcoming', color: 'text-blue-700', bgColor: 'bg-blue-100' },
  IN_PROGRESS: { label: 'In Progress', color: 'text-yellow-700', bgColor: 'bg-yellow-100' },
  UNDER_REVIEW: { label: 'Under Review', color: 'text-purple-700', bgColor: 'bg-purple-100' },
  SUBMITTED: { label: 'Submitted', color: 'text-green-700', bgColor: 'bg-green-100' },
  APPROVED: { label: 'Approved', color: 'text-emerald-700', bgColor: 'bg-emerald-100' },
  OVERDUE: { label: 'Overdue', color: 'text-red-700', bgColor: 'bg-red-100' },
  EXTENDED: { label: 'Extended', color: 'text-orange-700', bgColor: 'bg-orange-100' }
};

// ============================================================================
// Mock Data
// ============================================================================

const MOCK_DEADLINES: RegulatoryDeadline[] = [
  {
    id: 'dl1',
    name: 'Form ADV Annual Amendment',
    regulatoryBody: 'SEC',
    filingType: 'ADV',
    description: 'Annual update to Form ADV including Part 1A, Part 2A (Brochure), and Part 2B (Brochure Supplement)',
    dueDate: new Date(Date.now() + 15 * 24 * 60 * 60 * 1000),
    status: 'IN_PROGRESS',
    penaltyRiskScore: 72,
    estimatedPenalty: 125000,
    affectedAccounts: 1250,
    affectedAUM: 2500000000,
    assignedTo: ['Sarah Mitchell', 'James Chen'],
    dependencies: ['Client data reconciliation', 'Fee schedule review'],
    completionProgress: 65,
    reminderSchedule: [
      { daysBeforeDue: 30, recipientType: 'ASSIGNEE', notificationType: 'EMAIL', sent: true, sentAt: new Date(Date.now() - 15 * 24 * 60 * 60 * 1000) },
      { daysBeforeDue: 14, recipientType: 'MANAGER', notificationType: 'ALL', sent: true, sentAt: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000) },
      { daysBeforeDue: 7, recipientType: 'COMPLIANCE_OFFICER', notificationType: 'ALL', sent: false },
      { daysBeforeDue: 1, recipientType: 'EXECUTIVE', notificationType: 'ALL', sent: false }
    ],
    history: [
      { id: 'h1', action: 'Started preparation', user: 'Sarah Mitchell', timestamp: new Date(Date.now() - 20 * 24 * 60 * 60 * 1000) },
      { id: 'h2', action: 'Client data reconciliation completed', user: 'James Chen', timestamp: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000) }
    ]
  },
  {
    id: 'dl2',
    name: 'Form 13F Quarterly Report',
    regulatoryBody: 'SEC',
    filingType: '13F',
    description: 'Quarterly report of equity holdings for institutional investment managers',
    dueDate: new Date(Date.now() + 5 * 24 * 60 * 60 * 1000),
    status: 'UNDER_REVIEW',
    penaltyRiskScore: 45,
    estimatedPenalty: 50000,
    affectedAccounts: 850,
    affectedAUM: 1800000000,
    assignedTo: ['Emily Rodriguez'],
    dependencies: [],
    completionProgress: 90,
    reminderSchedule: [
      { daysBeforeDue: 14, recipientType: 'ASSIGNEE', notificationType: 'EMAIL', sent: true },
      { daysBeforeDue: 7, recipientType: 'MANAGER', notificationType: 'ALL', sent: true },
      { daysBeforeDue: 3, recipientType: 'COMPLIANCE_OFFICER', notificationType: 'ALL', sent: false }
    ],
    history: []
  },
  {
    id: 'dl3',
    name: 'FINRA Annual Audit',
    regulatoryBody: 'FINRA',
    filingType: 'AUDIT',
    description: 'Annual independent audit and FOCUS report filing',
    dueDate: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000),
    status: 'OVERDUE',
    penaltyRiskScore: 95,
    estimatedPenalty: 350000,
    affectedAccounts: 2100,
    affectedAUM: 4200000000,
    assignedTo: ['Sarah Mitchell', 'External Auditor'],
    dependencies: ['External audit completion', 'Financial reconciliation'],
    completionProgress: 75,
    reminderSchedule: [],
    history: []
  },
  {
    id: 'dl4',
    name: 'State Registration Renewal - CA',
    regulatoryBody: 'STATE',
    filingType: 'REGISTRATION',
    description: 'California state investment adviser registration renewal',
    dueDate: new Date(Date.now() + 45 * 24 * 60 * 60 * 1000),
    status: 'UPCOMING',
    penaltyRiskScore: 25,
    estimatedPenalty: 15000,
    affectedAccounts: 320,
    affectedAUM: 650000000,
    assignedTo: ['James Chen'],
    dependencies: ['Form ADV completion'],
    completionProgress: 0,
    reminderSchedule: [
      { daysBeforeDue: 30, recipientType: 'ASSIGNEE', notificationType: 'EMAIL', sent: false }
    ],
    history: []
  },
  {
    id: 'dl5',
    name: 'DOL 5500 Filing',
    regulatoryBody: 'DOL',
    filingType: '5500',
    description: 'Annual retirement plan filing for ERISA-covered plans',
    dueDate: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
    status: 'IN_PROGRESS',
    penaltyRiskScore: 55,
    estimatedPenalty: 75000,
    affectedAccounts: 45,
    affectedAUM: 890000000,
    assignedTo: ['Emily Rodriguez', 'Sarah Mitchell'],
    dependencies: ['Plan audit completion', 'Participant data'],
    completionProgress: 40,
    reminderSchedule: [],
    history: []
  }
];

// ============================================================================
// Helper Components  
// ============================================================================

const RiskProgressBar: React.FC<{ avgRisk: number }> = ({ avgRisk }) => {
  const colorClass = avgRisk >= 70 ? 'bg-red-500' : avgRisk >= 40 ? 'bg-yellow-500' : 'bg-green-500';
  const widthClass = avgRisk >= 100 ? 'w-full' : 
    avgRisk >= 90 ? 'w-[90%]' : 
    avgRisk >= 80 ? 'w-[80%]' : 
    avgRisk >= 70 ? 'w-[70%]' : 
    avgRisk >= 60 ? 'w-[60%]' : 
    avgRisk >= 50 ? 'w-[50%]' : 
    avgRisk >= 40 ? 'w-[40%]' : 
    avgRisk >= 30 ? 'w-[30%]' : 
    avgRisk >= 20 ? 'w-[20%]' : 
    avgRisk >= 10 ? 'w-[10%]' : 'w-0';
  return <div className={`h-full rounded-full ${colorClass} ${widthClass}`} />;
};

const RiskScoreBadge: React.FC<{ score: number }> = ({ score }) => {
  const getColor = () => {
    if (score >= 80) return 'bg-red-100 text-red-800 border-red-200';
    if (score >= 60) return 'bg-orange-100 text-orange-800 border-orange-200';
    if (score >= 40) return 'bg-yellow-100 text-yellow-800 border-yellow-200';
    return 'bg-green-100 text-green-800 border-green-200';
  };

  return (
    <div className={`flex items-center gap-1 px-2 py-1 rounded border ${getColor()}`}>
      <AlertTriangle className="w-3 h-3" />
      <span className="text-xs font-medium">Risk: {score}</span>
    </div>
  );
};

const ProgressBar: React.FC<{ progress: number }> = ({ progress }) => {
  const widthClass = progress >= 100 ? 'w-full' : 
    progress >= 90 ? 'w-[90%]' : 
    progress >= 80 ? 'w-[80%]' : 
    progress >= 70 ? 'w-[70%]' : 
    progress >= 60 ? 'w-[60%]' : 
    progress >= 50 ? 'w-[50%]' : 
    progress >= 40 ? 'w-[40%]' : 
    progress >= 30 ? 'w-[30%]' : 
    progress >= 20 ? 'w-[20%]' : 
    progress >= 10 ? 'w-[10%]' : 'w-0';
  
  const colorClass = progress >= 80 ? 'bg-green-500' : 
    progress >= 50 ? 'bg-yellow-500' : 
    progress >= 25 ? 'bg-orange-500' : 'bg-red-500';

  return (
    <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
      <div className={`h-full rounded-full transition-all ${colorClass} ${widthClass}`} />
    </div>
  );
};

const DaysRemaining: React.FC<{ dueDate: Date }> = ({ dueDate }) => {
  const days = Math.ceil((dueDate.getTime() - Date.now()) / (24 * 60 * 60 * 1000));
  
  if (days < 0) {
    return (
      <div className="flex items-center gap-1 text-red-600">
        <AlertCircle className="w-4 h-4" />
        <span className="font-medium">{Math.abs(days)} days overdue</span>
      </div>
    );
  }
  
  if (days <= 7) {
    return (
      <div className="flex items-center gap-1 text-orange-600">
        <Clock className="w-4 h-4" />
        <span className="font-medium">{days} days remaining</span>
      </div>
    );
  }
  
  return (
    <div className="flex items-center gap-1 text-gray-600">
      <Calendar className="w-4 h-4" />
      <span>{days} days remaining</span>
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

interface RegulatoryDeadlineIntelligenceProps {
  tenantId?: string;
  datasourceId?: string;
}

export const RegulatoryDeadlineIntelligence: React.FC<RegulatoryDeadlineIntelligenceProps> = ({
  tenantId: _tenantId,
  datasourceId: _datasourceId
}) => {
  // State
  const [deadlines] = useState<RegulatoryDeadline[]>(MOCK_DEADLINES);
  const [selectedDeadline, setSelectedDeadline] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'deadlines' | 'calendar' | 'risk' | 'reminders'>('deadlines');
  const [filterBody, setFilterBody] = useState<string>('ALL');
  const [filterStatus, setFilterStatus] = useState<string>('ALL');
  const [searchQuery, setSearchQuery] = useState('');

  // Derived state
  const metrics = useMemo(() => ({
    total: deadlines.length,
    overdue: deadlines.filter(d => d.status === 'OVERDUE').length,
    upcoming7Days: deadlines.filter(d => {
      const days = Math.ceil((d.dueDate.getTime() - Date.now()) / (24 * 60 * 60 * 1000));
      return days > 0 && days <= 7 && d.status !== 'SUBMITTED' && d.status !== 'APPROVED';
    }).length,
    highRisk: deadlines.filter(d => d.penaltyRiskScore >= 70).length,
    totalPenaltyExposure: deadlines.reduce((sum, d) => sum + d.estimatedPenalty, 0),
    avgCompletion: deadlines.reduce((sum, d) => sum + d.completionProgress, 0) / Math.max(deadlines.length, 1)
  }), [deadlines]);

  const filteredDeadlines = useMemo(() => {
    return deadlines.filter(d => {
      if (filterBody !== 'ALL' && d.regulatoryBody !== filterBody) return false;
      if (filterStatus !== 'ALL' && d.status !== filterStatus) return false;
      if (searchQuery && !d.name.toLowerCase().includes(searchQuery.toLowerCase()) && 
          !d.filingType.toLowerCase().includes(searchQuery.toLowerCase())) return false;
      return true;
    }).sort((a, b) => a.dueDate.getTime() - b.dueDate.getTime());
  }, [deadlines, filterBody, filterStatus, searchQuery]);

  // Calculate penalty breakdown for selected deadline
  const calculatePenalty = (deadline: RegulatoryDeadline): PenaltyCalculation => {
    const baseAmount = 10000;
    const aumMultiplier = deadline.affectedAUM / 1000000000;
    const historyMultiplier = 1.2; // Based on past compliance
    const daysOverdue = Math.max(0, Math.ceil((Date.now() - deadline.dueDate.getTime()) / (24 * 60 * 60 * 1000)));
    const daysOverdueMultiplier = 1 + (daysOverdue * 0.1);
    
    const totalEstimate = baseAmount * aumMultiplier * historyMultiplier * daysOverdueMultiplier;
    
    return {
      baseAmount,
      aumMultiplier,
      historyMultiplier,
      daysOverdueMultiplier,
      totalEstimate,
      factors: [
        { name: 'AUM Scale', impact: aumMultiplier > 2 ? 'INCREASE' : 'NEUTRAL', description: `$${(deadline.affectedAUM / 1000000000).toFixed(1)}B under management`, weight: aumMultiplier },
        { name: 'Compliance History', impact: 'INCREASE', description: 'Prior late filings in past 3 years', weight: historyMultiplier },
        { name: 'Days Overdue', impact: daysOverdue > 0 ? 'INCREASE' : 'NEUTRAL', description: daysOverdue > 0 ? `${daysOverdue} days past due` : 'On time', weight: daysOverdueMultiplier },
        { name: 'Affected Clients', impact: deadline.affectedAccounts > 1000 ? 'INCREASE' : 'NEUTRAL', description: `${deadline.affectedAccounts} client accounts impacted`, weight: 1.0 }
      ]
    };
  };

  // Render deadlines tab
  const renderDeadlines = () => (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search deadlines..."
            className="w-full pl-10 pr-4 py-2 border rounded-lg text-sm"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-gray-500" />
          <select
            value={filterBody}
            onChange={(e) => setFilterBody(e.target.value)}
            className="border rounded-lg px-3 py-2 text-sm"
            title="Filter by regulatory body"
          >
            <option value="ALL">All Bodies</option>
            {Object.entries(REGULATORY_BODY_CONFIG).map(([key, config]) => (
              <option key={key} value={key}>{config.label}</option>
            ))}
          </select>
          <select
            value={filterStatus}
            onChange={(e) => setFilterStatus(e.target.value)}
            className="border rounded-lg px-3 py-2 text-sm"
            title="Filter by status"
          >
            <option value="ALL">All Status</option>
            {Object.entries(STATUS_CONFIG).map(([key, config]) => (
              <option key={key} value={key}>{config.label}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Deadline list */}
      <div className="space-y-3">
        {filteredDeadlines.map(deadline => {
          const bodyConfig = REGULATORY_BODY_CONFIG[deadline.regulatoryBody];
          const statusConfig = STATUS_CONFIG[deadline.status];
          
          return (
            <div
              key={deadline.id}
              className={`bg-white rounded-lg border p-4 cursor-pointer transition-all hover:shadow-md ${
                selectedDeadline === deadline.id ? 'ring-2 ring-blue-500' : ''
              } ${deadline.status === 'OVERDUE' ? 'border-red-200 bg-red-50' : ''}`}
              onClick={() => setSelectedDeadline(selectedDeadline === deadline.id ? null : deadline.id)}
              onKeyDown={(e) => e.key === 'Enter' && setSelectedDeadline(selectedDeadline === deadline.id ? null : deadline.id)}
              tabIndex={0}
              role="button"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <span className={`px-2 py-0.5 rounded text-xs ${bodyConfig.bgColor} ${bodyConfig.color}`}>
                      {bodyConfig.icon} {bodyConfig.label}
                    </span>
                    <h3 className="font-medium">{deadline.name}</h3>
                    <span className={`px-2 py-0.5 rounded text-xs ${statusConfig.bgColor} ${statusConfig.color}`}>
                      {statusConfig.label}
                    </span>
                  </div>
                  <p className="text-sm text-gray-600 mt-1">{deadline.description}</p>
                  <div className="flex items-center gap-6 mt-3">
                    <DaysRemaining dueDate={deadline.dueDate} />
                    <RiskScoreBadge score={deadline.penaltyRiskScore} />
                    <div className="flex items-center gap-1 text-gray-500">
                      <DollarSign className="w-4 h-4" />
                      <span className="text-sm">${(deadline.estimatedPenalty / 1000).toFixed(0)}K exposure</span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <div className="text-right">
                    <div className="text-sm text-gray-500">Progress</div>
                    <div className="font-medium">{deadline.completionProgress}%</div>
                  </div>
                  <ChevronRight className={`w-5 h-5 text-gray-400 transition-transform ${selectedDeadline === deadline.id ? 'rotate-90' : ''}`} />
                </div>
              </div>

              {/* Progress bar */}
              <div className="mt-3">
                <ProgressBar progress={deadline.completionProgress} />
              </div>

              {/* Expanded details */}
              {selectedDeadline === deadline.id && (
                <div className="mt-4 pt-4 border-t">
                  <div className="grid grid-cols-3 gap-6">
                    <div>
                      <h4 className="text-xs font-medium text-gray-500 mb-2">Assigned Team</h4>
                      <div className="flex flex-wrap gap-1">
                        {deadline.assignedTo.map(person => (
                          <span key={person} className="px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs">
                            {person}
                          </span>
                        ))}
                      </div>
                    </div>
                    <div>
                      <h4 className="text-xs font-medium text-gray-500 mb-2">Dependencies</h4>
                      <div className="space-y-1">
                        {deadline.dependencies.length > 0 ? (
                          deadline.dependencies.map(dep => (
                            <div key={dep} className="flex items-center gap-1 text-xs">
                              <span className="w-1.5 h-1.5 rounded-full bg-gray-400" />
                              {dep}
                            </div>
                          ))
                        ) : (
                          <span className="text-xs text-gray-400">No dependencies</span>
                        )}
                      </div>
                    </div>
                    <div>
                      <h4 className="text-xs font-medium text-gray-500 mb-2">Impact</h4>
                      <div className="space-y-1 text-sm">
                        <div className="flex justify-between">
                          <span className="text-gray-500">Accounts:</span>
                          <span className="font-medium">{deadline.affectedAccounts.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-gray-500">AUM:</span>
                          <span className="font-medium">${(deadline.affectedAUM / 1000000000).toFixed(2)}B</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Penalty calculation */}
                  <div className="mt-4 p-3 bg-red-50 rounded-lg border border-red-100">
                    <h4 className="text-xs font-medium text-red-700 mb-2">Penalty Risk Analysis</h4>
                    {(() => {
                      const penalty = calculatePenalty(deadline);
                      return (
                        <div className="space-y-2">
                          <div className="grid grid-cols-4 gap-2 text-xs">
                            {penalty.factors.map(factor => (
                              <div key={factor.name} className={`p-2 rounded ${
                                factor.impact === 'INCREASE' ? 'bg-red-100' : 'bg-gray-100'
                              }`}>
                                <div className="font-medium">{factor.name}</div>
                                <div className="text-gray-600">{factor.description}</div>
                              </div>
                            ))}
                          </div>
                          <div className="flex items-center justify-between pt-2 border-t border-red-200">
                            <span className="text-sm text-red-700">Estimated Total Penalty:</span>
                            <span className="text-lg font-bold text-red-800">
                              ${penalty.totalEstimate.toLocaleString(undefined, { maximumFractionDigits: 0 })}
                            </span>
                          </div>
                        </div>
                      );
                    })()}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );

  // Render calendar tab
  const renderCalendar = () => {
    const months = Array.from({ length: 3 }, (_, i) => {
      const date = new Date();
      date.setMonth(date.getMonth() + i);
      return date;
    });

    return (
      <div className="space-y-6">
        {months.map((month, idx) => {
          const monthDeadlines = deadlines.filter(d => {
            const dDate = new Date(d.dueDate);
            return dDate.getMonth() === month.getMonth() && dDate.getFullYear() === month.getFullYear();
          });

          return (
            <div key={idx} className="bg-white rounded-lg border p-4">
              <h3 className="font-semibold mb-4">
                {month.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
              </h3>
              {monthDeadlines.length > 0 ? (
                <div className="space-y-2">
                  {monthDeadlines.map(d => (
                    <div key={d.id} className="flex items-center justify-between p-2 bg-gray-50 rounded">
                      <div className="flex items-center gap-3">
                        <span className="text-sm font-medium text-gray-700">
                          {d.dueDate.toLocaleDateString('en-US', { day: 'numeric', weekday: 'short' })}
                        </span>
                        <span className={`px-2 py-0.5 rounded text-xs ${REGULATORY_BODY_CONFIG[d.regulatoryBody].bgColor} ${REGULATORY_BODY_CONFIG[d.regulatoryBody].color}`}>
                          {REGULATORY_BODY_CONFIG[d.regulatoryBody].label}
                        </span>
                        <span className="text-sm">{d.name}</span>
                      </div>
                      <RiskScoreBadge score={d.penaltyRiskScore} />
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-400">No deadlines this month</p>
              )}
            </div>
          );
        })}
      </div>
    );
  };

  // Render risk tab
  const renderRisk = () => (
    <div className="space-y-6">
      {/* Risk overview */}
      <div className="bg-white rounded-lg border p-6">
        <h3 className="font-semibold mb-4">Total Penalty Exposure</h3>
        <div className="grid grid-cols-3 gap-6">
          <div className="text-center">
            <div className="text-4xl font-bold text-red-600">
              ${(metrics.totalPenaltyExposure / 1000000).toFixed(2)}M
            </div>
            <div className="text-sm text-gray-500">Maximum Exposure</div>
          </div>
          <div className="text-center">
            <div className="text-4xl font-bold text-orange-600">
              {metrics.highRisk}
            </div>
            <div className="text-sm text-gray-500">High Risk Filings</div>
          </div>
          <div className="text-center">
            <div className="text-4xl font-bold text-red-600">
              {metrics.overdue}
            </div>
            <div className="text-sm text-gray-500">Overdue Items</div>
          </div>
        </div>
      </div>

      {/* Risk breakdown by body */}
      <div className="bg-white rounded-lg border p-6">
        <h3 className="font-semibold mb-4">Risk by Regulatory Body</h3>
        <div className="space-y-3">
          {Object.entries(REGULATORY_BODY_CONFIG).map(([body, config]) => {
            const bodyDeadlines = deadlines.filter(d => d.regulatoryBody === body);
            const totalExposure = bodyDeadlines.reduce((sum, d) => sum + d.estimatedPenalty, 0);
            const avgRisk = bodyDeadlines.length > 0 
              ? bodyDeadlines.reduce((sum, d) => sum + d.penaltyRiskScore, 0) / bodyDeadlines.length 
              : 0;

            if (bodyDeadlines.length === 0) return null;

            return (
              <div key={body} className="flex items-center gap-4 p-3 bg-gray-50 rounded-lg">
                <span className={`px-2 py-1 rounded ${config.bgColor} ${config.color} text-sm`}>
                  {config.icon} {config.label}
                </span>
                <div className="flex-1">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-sm">{bodyDeadlines.length} filing(s)</span>
                    <span className="text-sm font-medium">${(totalExposure / 1000).toFixed(0)}K exposure</span>
                  </div>
                  <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                    <RiskProgressBar avgRisk={avgRisk} />
                  </div>
                </div>
                <span className="text-sm text-gray-500">Avg Risk: {avgRisk.toFixed(0)}</span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );

  // Render reminders tab
  const renderReminders = () => {
    const allReminders = deadlines.flatMap(d => 
      d.reminderSchedule.map(r => ({ ...r, deadlineName: d.name, dueDate: d.dueDate, deadlineId: d.id }))
    ).sort((a, b) => {
      const aDays = Math.ceil((a.dueDate.getTime() - Date.now()) / (24 * 60 * 60 * 1000)) - a.daysBeforeDue;
      const bDays = Math.ceil((b.dueDate.getTime() - Date.now()) / (24 * 60 * 60 * 1000)) - b.daysBeforeDue;
      return aDays - bDays;
    });

    return (
      <div className="space-y-4">
        <div className="grid grid-cols-3 gap-4">
          <div className="bg-green-50 rounded-lg border border-green-200 p-4">
            <div className="flex items-center gap-2 text-green-700">
              <CheckCircle className="w-5 h-5" />
              <span className="font-medium">Sent</span>
            </div>
            <div className="text-2xl font-bold text-green-800 mt-2">
              {allReminders.filter(r => r.sent).length}
            </div>
          </div>
          <div className="bg-yellow-50 rounded-lg border border-yellow-200 p-4">
            <div className="flex items-center gap-2 text-yellow-700">
              <Clock className="w-5 h-5" />
              <span className="font-medium">Pending</span>
            </div>
            <div className="text-2xl font-bold text-yellow-800 mt-2">
              {allReminders.filter(r => !r.sent).length}
            </div>
          </div>
          <div className="bg-blue-50 rounded-lg border border-blue-200 p-4">
            <div className="flex items-center gap-2 text-blue-700">
              <Bell className="w-5 h-5" />
              <span className="font-medium">This Week</span>
            </div>
            <div className="text-2xl font-bold text-blue-800 mt-2">
              {allReminders.filter(r => {
                const triggerDate = new Date(r.dueDate.getTime() - r.daysBeforeDue * 24 * 60 * 60 * 1000);
                const daysUntil = Math.ceil((triggerDate.getTime() - Date.now()) / (24 * 60 * 60 * 1000));
                return daysUntil >= 0 && daysUntil <= 7 && !r.sent;
              }).length}
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg border overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50 border-b">
              <tr>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Deadline</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Days Before Due</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Trigger Date</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Recipients</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Method</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {allReminders.map((reminder, idx) => {
                const triggerDate = new Date(reminder.dueDate.getTime() - reminder.daysBeforeDue * 24 * 60 * 60 * 1000);
                return (
                  <tr key={`${reminder.deadlineId}-${idx}`} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-sm">{reminder.deadlineName}</td>
                    <td className="px-4 py-3 text-sm">{reminder.daysBeforeDue} days</td>
                    <td className="px-4 py-3 text-sm">{triggerDate.toLocaleDateString()}</td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 bg-gray-100 rounded text-xs">
                        {reminder.recipientType.replace(/_/g, ' ')}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm">{reminder.notificationType}</td>
                    <td className="px-4 py-3">
                      {reminder.sent ? (
                        <span className="flex items-center gap-1 text-green-600 text-sm">
                          <CheckCircle className="w-4 h-4" />
                          Sent
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-gray-500 text-sm">
                          <Clock className="w-4 h-4" />
                          Pending
                        </span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    );
  };

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold flex items-center gap-2">
              <Shield className="w-6 h-6 text-blue-600" />
              Regulatory Deadline Intelligence
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              Compliance deadline tracking with penalty risk scoring
            </p>
          </div>
          <div className="flex items-center gap-3">
            {metrics.overdue > 0 && (
              <div className="flex items-center gap-2 px-3 py-1.5 bg-red-100 text-red-700 rounded-lg">
                <AlertTriangle className="w-4 h-4" />
                <span className="text-sm font-medium">{metrics.overdue} Overdue</span>
              </div>
            )}
            <button className="flex items-center gap-2 px-3 py-1.5 border rounded-lg hover:bg-gray-50">
              <Settings className="w-4 h-4" />
              Settings
            </button>
          </div>
        </div>

        {/* Stats bar */}
        <div className="grid grid-cols-5 gap-4 mt-4">
          <div className="bg-gray-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-gray-500">Total Deadlines</span>
              <FileText className="w-4 h-4 text-gray-400" />
            </div>
            <div className="text-xl font-bold">{metrics.total}</div>
          </div>
          <div className="bg-red-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-red-600">Overdue</span>
              <AlertCircle className="w-4 h-4 text-red-400" />
            </div>
            <div className="text-xl font-bold text-red-700">{metrics.overdue}</div>
          </div>
          <div className="bg-orange-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-orange-600">Due in 7 Days</span>
              <Clock className="w-4 h-4 text-orange-400" />
            </div>
            <div className="text-xl font-bold text-orange-700">{metrics.upcoming7Days}</div>
          </div>
          <div className="bg-purple-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-purple-600">High Risk</span>
              <AlertTriangle className="w-4 h-4 text-purple-400" />
            </div>
            <div className="text-xl font-bold text-purple-700">{metrics.highRisk}</div>
          </div>
          <div className="bg-green-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-green-600">Avg Progress</span>
              <TrendingUp className="w-4 h-4 text-green-400" />
            </div>
            <div className="text-xl font-bold text-green-700">{metrics.avgCompletion.toFixed(0)}%</div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <div className="flex gap-6">
          {[
            { id: 'deadlines' as const, label: 'Deadlines', icon: FileText },
            { id: 'calendar' as const, label: 'Calendar View', icon: Calendar },
            { id: 'risk' as const, label: 'Risk Analysis', icon: AlertTriangle },
            { id: 'reminders' as const, label: 'Reminders', icon: Bell }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-2 px-1 py-3 border-b-2 transition-colors ${
                activeTab === tab.id 
                  ? 'border-blue-500 text-blue-600' 
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              <tab.icon className="w-4 h-4" />
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {activeTab === 'deadlines' && renderDeadlines()}
        {activeTab === 'calendar' && renderCalendar()}
        {activeTab === 'risk' && renderRisk()}
        {activeTab === 'reminders' && renderReminders()}
      </div>
    </div>
  );
};

export default RegulatoryDeadlineIntelligence;
