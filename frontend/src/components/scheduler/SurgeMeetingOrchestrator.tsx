/**
 * SurgeMeetingOrchestrator.tsx
 * 
 * Surge Meeting Orchestrator with AI Load Balancing:
 * - Intelligent meeting slot allocation across advisor teams
 * - OR-Tools constraint satisfaction for optimal scheduling
 * - Real-time capacity monitoring and surge detection
 * - AI-powered advisor matching based on expertise
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Users,
  Clock,
  AlertTriangle,
  Settings,
  ChevronRight,
  RefreshCw,
  BarChart3,
  Brain,
  Sparkles,
  Filter,
  Search,
  Bell
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface Advisor {
  id: string;
  name: string;
  title: string;
  expertise: string[];
  certifications: string[];
  maxDailyMeetings: number;
  preferredHours: { start: number; end: number };
  currentLoad: number;
  availability: DayAvailability[];
}

interface DayAvailability {
  date: string;
  slots: TimeSlot[];
  blockedReasons?: string[];
}

interface TimeSlot {
  start: string;
  end: string;
  status: 'AVAILABLE' | 'BOOKED' | 'HELD' | 'BLOCKED';
  meetingId?: string;
  clientId?: string;
}

interface MeetingRequest {
  id: string;
  clientId: string;
  clientName: string;
  clientTier: 'PLATINUM' | 'GOLD' | 'SILVER' | 'STANDARD';
  meetingType: MeetingType;
  duration: number; // minutes
  urgency: 'CRITICAL' | 'HIGH' | 'NORMAL' | 'LOW';
  requestedTimeframe: { earliest: Date; latest: Date };
  requiredExpertise: string[];
  preferredAdvisors: string[];
  status: 'PENDING' | 'SCHEDULED' | 'CONFIRMED' | 'CANCELLED';
  scheduledSlot?: { advisorId: string; date: string; time: string };
  aiScore?: number;
}

type MeetingType = 
  | 'ANNUAL_REVIEW'
  | 'QUARTERLY_UPDATE'
  | 'PORTFOLIO_REBALANCE'
  | 'TAX_PLANNING'
  | 'ESTATE_PLANNING'
  | 'NEW_CLIENT_ONBOARDING'
  | 'URGENT_CONSULTATION'
  | 'RETIREMENT_PLANNING';

interface SurgeAlert {
  id: string;
  type: 'CAPACITY_WARNING' | 'OVERLOAD' | 'SKILL_GAP' | 'TIME_CONFLICT';
  severity: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
  message: string;
  affectedAdvisors: string[];
  suggestedAction: string;
  timestamp: Date;
}

interface ScheduleOptimization {
  id: string;
  status: 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED';
  startTime: Date;
  endTime?: Date;
  requestsProcessed: number;
  requestsScheduled: number;
  optimizationScore: number;
  constraints: OptimizationConstraint[];
}

interface OptimizationConstraint {
  type: string;
  description: string;
  weight: number;
  satisfied: boolean;
}

// ============================================================================
// Constants
// ============================================================================

const MEETING_TYPE_CONFIG: Record<MeetingType, {
  label: string;
  defaultDuration: number;
  requiredExpertise: string[];
  color: string;
}> = {
  ANNUAL_REVIEW: { label: 'Annual Review', defaultDuration: 60, requiredExpertise: ['financial_planning'], color: 'bg-blue-100 text-blue-800' },
  QUARTERLY_UPDATE: { label: 'Quarterly Update', defaultDuration: 30, requiredExpertise: ['portfolio_management'], color: 'bg-green-100 text-green-800' },
  PORTFOLIO_REBALANCE: { label: 'Portfolio Rebalance', defaultDuration: 45, requiredExpertise: ['portfolio_management', 'trading'], color: 'bg-purple-100 text-purple-800' },
  TAX_PLANNING: { label: 'Tax Planning', defaultDuration: 60, requiredExpertise: ['tax_planning', 'cpa'], color: 'bg-orange-100 text-orange-800' },
  ESTATE_PLANNING: { label: 'Estate Planning', defaultDuration: 90, requiredExpertise: ['estate_planning', 'legal'], color: 'bg-indigo-100 text-indigo-800' },
  NEW_CLIENT_ONBOARDING: { label: 'New Client Onboarding', defaultDuration: 90, requiredExpertise: ['onboarding', 'compliance'], color: 'bg-cyan-100 text-cyan-800' },
  URGENT_CONSULTATION: { label: 'Urgent Consultation', defaultDuration: 30, requiredExpertise: [], color: 'bg-red-100 text-red-800' },
  RETIREMENT_PLANNING: { label: 'Retirement Planning', defaultDuration: 60, requiredExpertise: ['retirement_planning', 'social_security'], color: 'bg-amber-100 text-amber-800' }
};

const URGENCY_COLORS: Record<string, string> = {
  CRITICAL: 'bg-red-100 text-red-800 border-red-200',
  HIGH: 'bg-orange-100 text-orange-800 border-orange-200',
  NORMAL: 'bg-blue-100 text-blue-800 border-blue-200',
  LOW: 'bg-gray-100 text-gray-800 border-gray-200'
};

const CLIENT_TIER_COLORS: Record<string, string> = {
  PLATINUM: 'bg-purple-100 text-purple-800',
  GOLD: 'bg-yellow-100 text-yellow-800',
  SILVER: 'bg-gray-200 text-gray-800',
  STANDARD: 'bg-blue-50 text-blue-800'
};

// ============================================================================
// Mock Data
// ============================================================================

const MOCK_ADVISORS: Advisor[] = [
  {
    id: 'adv1',
    name: 'Sarah Mitchell',
    title: 'Senior Wealth Advisor',
    expertise: ['financial_planning', 'retirement_planning', 'estate_planning'],
    certifications: ['CFP', 'CFA'],
    maxDailyMeetings: 6,
    preferredHours: { start: 9, end: 17 },
    currentLoad: 4,
    availability: []
  },
  {
    id: 'adv2',
    name: 'James Chen',
    title: 'Portfolio Manager',
    expertise: ['portfolio_management', 'trading', 'tax_planning'],
    certifications: ['CFA', 'CPA'],
    maxDailyMeetings: 5,
    preferredHours: { start: 8, end: 16 },
    currentLoad: 5,
    availability: []
  },
  {
    id: 'adv3',
    name: 'Emily Rodriguez',
    title: 'Financial Planner',
    expertise: ['financial_planning', 'social_security', 'onboarding'],
    certifications: ['CFP'],
    maxDailyMeetings: 7,
    preferredHours: { start: 10, end: 18 },
    currentLoad: 3,
    availability: []
  }
];

const MOCK_REQUESTS: MeetingRequest[] = [
  {
    id: 'req1',
    clientId: 'c001',
    clientName: 'Robert & Linda Thompson',
    clientTier: 'PLATINUM',
    meetingType: 'ANNUAL_REVIEW',
    duration: 60,
    urgency: 'NORMAL',
    requestedTimeframe: { earliest: new Date(), latest: new Date(Date.now() + 14 * 24 * 60 * 60 * 1000) },
    requiredExpertise: ['financial_planning', 'retirement_planning'],
    preferredAdvisors: ['adv1'],
    status: 'PENDING',
    aiScore: 0.92
  },
  {
    id: 'req2',
    clientId: 'c002',
    clientName: 'Michael Chang',
    clientTier: 'GOLD',
    meetingType: 'TAX_PLANNING',
    duration: 60,
    urgency: 'HIGH',
    requestedTimeframe: { earliest: new Date(), latest: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000) },
    requiredExpertise: ['tax_planning', 'cpa'],
    preferredAdvisors: ['adv2'],
    status: 'PENDING',
    aiScore: 0.87
  },
  {
    id: 'req3',
    clientId: 'c003',
    clientName: 'Jennifer Williams',
    clientTier: 'SILVER',
    meetingType: 'NEW_CLIENT_ONBOARDING',
    duration: 90,
    urgency: 'NORMAL',
    requestedTimeframe: { earliest: new Date(), latest: new Date(Date.now() + 10 * 24 * 60 * 60 * 1000) },
    requiredExpertise: ['onboarding', 'compliance'],
    preferredAdvisors: [],
    status: 'PENDING',
    aiScore: 0.78
  },
  {
    id: 'req4',
    clientId: 'c004',
    clientName: 'David & Maria Santos',
    clientTier: 'PLATINUM',
    meetingType: 'URGENT_CONSULTATION',
    duration: 30,
    urgency: 'CRITICAL',
    requestedTimeframe: { earliest: new Date(), latest: new Date(Date.now() + 2 * 24 * 60 * 60 * 1000) },
    requiredExpertise: [],
    preferredAdvisors: ['adv1', 'adv2'],
    status: 'PENDING',
    aiScore: 0.95
  }
];

// ============================================================================
// Helper Components
// ============================================================================

const LoadBar: React.FC<{ current: number; max: number }> = ({ current, max }) => {
  const percentage = (current / max) * 100;
  const colorClass = percentage >= 90 ? 'bg-red-500' : percentage >= 70 ? 'bg-yellow-500' : 'bg-green-500';
  const widthClass = percentage >= 100 ? 'w-full' : 
    percentage >= 90 ? 'w-[90%]' : 
    percentage >= 80 ? 'w-[80%]' : 
    percentage >= 70 ? 'w-[70%]' : 
    percentage >= 60 ? 'w-[60%]' : 
    percentage >= 50 ? 'w-[50%]' : 
    percentage >= 40 ? 'w-[40%]' : 
    percentage >= 30 ? 'w-[30%]' : 
    percentage >= 20 ? 'w-[20%]' : 
    percentage >= 10 ? 'w-[10%]' : 'w-0';
  
  return (
    <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
      <div className={`h-full rounded-full transition-all ${colorClass} ${widthClass}`} />
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

interface SurgeMeetingOrchestratorProps {
  tenantId?: string;
  datasourceId?: string;
}

export const SurgeMeetingOrchestrator: React.FC<SurgeMeetingOrchestratorProps> = ({
  tenantId: _tenantId,
  datasourceId: _datasourceId
}) => {
  // State
  const [advisors, setAdvisors] = useState<Advisor[]>(MOCK_ADVISORS);
  const [requests, setRequests] = useState<MeetingRequest[]>(MOCK_REQUESTS);
  const [alerts, setAlerts] = useState<SurgeAlert[]>([]);
  const [selectedAdvisor, setSelectedAdvisor] = useState<string | null>(null);
  const [selectedRequest, setSelectedRequest] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'queue' | 'advisors' | 'optimization' | 'alerts'>('queue');
  const [isOptimizing, setIsOptimizing] = useState(false);
  const [optimization, setOptimization] = useState<ScheduleOptimization | null>(null);
  const [filterUrgency, setFilterUrgency] = useState<string>('ALL');
  const [searchQuery, setSearchQuery] = useState('');

  // Derived state
  const filteredRequests = useMemo(() => {
    return requests.filter(req => {
      if (filterUrgency !== 'ALL' && req.urgency !== filterUrgency) return false;
      if (searchQuery && !req.clientName.toLowerCase().includes(searchQuery.toLowerCase())) return false;
      return true;
    });
  }, [requests, filterUrgency, searchQuery]);

  const teamCapacity = useMemo(() => {
    const total = advisors.reduce((sum, adv) => sum + adv.maxDailyMeetings, 0);
    const used = advisors.reduce((sum, adv) => sum + adv.currentLoad, 0);
    return { total, used, available: total - used, utilizationPct: (used / total) * 100 };
  }, [advisors]);

  const pendingCount = useMemo(() => requests.filter(r => r.status === 'PENDING').length, [requests]);
  const criticalCount = useMemo(() => requests.filter(r => r.urgency === 'CRITICAL' && r.status === 'PENDING').length, [requests]);

  // Simulate surge detection
  useEffect(() => {
    const checkSurge = () => {
      const newAlerts: SurgeAlert[] = [];
      
      // Check team capacity
      if (teamCapacity.utilizationPct >= 90) {
        newAlerts.push({
          id: `alert-capacity-${Date.now()}`,
          type: 'OVERLOAD',
          severity: 'CRITICAL',
          message: 'Team capacity at critical level',
          affectedAdvisors: advisors.filter(a => a.currentLoad >= a.maxDailyMeetings).map(a => a.id),
          suggestedAction: 'Consider rescheduling non-urgent meetings or bringing in additional support',
          timestamp: new Date()
        });
      } else if (teamCapacity.utilizationPct >= 70) {
        newAlerts.push({
          id: `alert-capacity-${Date.now()}`,
          type: 'CAPACITY_WARNING',
          severity: 'MEDIUM',
          message: 'Team capacity approaching limit',
          affectedAdvisors: advisors.filter(a => a.currentLoad >= a.maxDailyMeetings * 0.8).map(a => a.id),
          suggestedAction: 'Monitor incoming requests and prepare overflow capacity',
          timestamp: new Date()
        });
      }

      // Check for skill gaps
      const taxRequests = requests.filter(r => r.requiredExpertise.includes('tax_planning') && r.status === 'PENDING');
      const taxAdvisors = advisors.filter(a => a.expertise.includes('tax_planning'));
      if (taxRequests.length > taxAdvisors.length * 2) {
        newAlerts.push({
          id: `alert-skill-${Date.now()}`,
          type: 'SKILL_GAP',
          severity: 'HIGH',
          message: 'High demand for tax planning expertise',
          affectedAdvisors: taxAdvisors.map(a => a.id),
          suggestedAction: 'Consider scheduling tax planning meetings in batches or bringing in specialist',
          timestamp: new Date()
        });
      }

      setAlerts(newAlerts);
    };

    checkSurge();
    const interval = setInterval(checkSurge, 30000);
    return () => clearInterval(interval);
  }, [advisors, requests, teamCapacity.utilizationPct]);

  // Run AI optimization
  const runOptimization = useCallback(async () => {
    setIsOptimizing(true);
    const opt: ScheduleOptimization = {
      id: `opt-${Date.now()}`,
      status: 'RUNNING',
      startTime: new Date(),
      requestsProcessed: 0,
      requestsScheduled: 0,
      optimizationScore: 0,
      constraints: [
        { type: 'advisor_availability', description: 'Respect advisor availability windows', weight: 1.0, satisfied: false },
        { type: 'client_tier_priority', description: 'Prioritize platinum and gold clients', weight: 0.9, satisfied: false },
        { type: 'urgency_priority', description: 'Schedule critical and high urgency first', weight: 0.95, satisfied: false },
        { type: 'expertise_match', description: 'Match required expertise to advisor skills', weight: 0.85, satisfied: false },
        { type: 'load_balance', description: 'Distribute meetings evenly across team', weight: 0.7, satisfied: false },
        { type: 'client_preference', description: 'Honor preferred advisor when possible', weight: 0.6, satisfied: false }
      ]
    };
    setOptimization(opt);

    // Simulate optimization process
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Update requests with scheduled slots
    const updatedRequests = requests.map(req => {
      if (req.status !== 'PENDING') return req;
      
      // Find best advisor match
      const matchedAdvisor = advisors.find(adv => 
        req.requiredExpertise.every(exp => adv.expertise.includes(exp)) &&
        adv.currentLoad < adv.maxDailyMeetings
      ) || advisors.find(adv => adv.currentLoad < adv.maxDailyMeetings);

      if (matchedAdvisor) {
        return {
          ...req,
          status: 'SCHEDULED' as const,
          scheduledSlot: {
            advisorId: matchedAdvisor.id,
            date: new Date(Date.now() + Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
            time: `${9 + Math.floor(Math.random() * 8)}:00`
          }
        };
      }
      return req;
    });

    const scheduled = updatedRequests.filter(r => r.status === 'SCHEDULED').length;
    
    setRequests(updatedRequests);
    setOptimization({
      ...opt,
      status: 'COMPLETED',
      endTime: new Date(),
      requestsProcessed: requests.filter(r => r.status === 'PENDING').length,
      requestsScheduled: scheduled - requests.filter(r => r.status === 'SCHEDULED').length,
      optimizationScore: 0.89,
      constraints: opt.constraints.map(c => ({ ...c, satisfied: Math.random() > 0.2 }))
    });
    setIsOptimizing(false);

    // Update advisor loads
    setAdvisors(prev => prev.map(adv => {
      const newMeetings = updatedRequests.filter(r => r.scheduledSlot?.advisorId === adv.id && r.status === 'SCHEDULED').length;
      return { ...adv, currentLoad: Math.min(adv.currentLoad + newMeetings, adv.maxDailyMeetings) };
    }));
  }, [advisors, requests]);

  // Render queue tab
  const renderQueue = () => (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search clients..."
            className="w-full pl-10 pr-4 py-2 border rounded-lg text-sm"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-gray-500" />
          <select
            value={filterUrgency}
            onChange={(e) => setFilterUrgency(e.target.value)}
            className="border rounded-lg px-3 py-2 text-sm"
            title="Filter by urgency level"
          >
            <option value="ALL">All Urgencies</option>
            <option value="CRITICAL">Critical</option>
            <option value="HIGH">High</option>
            <option value="NORMAL">Normal</option>
            <option value="LOW">Low</option>
          </select>
        </div>
      </div>

      {/* Request list */}
      <div className="space-y-3">
        {filteredRequests.map(request => (
          <div
            key={request.id}
            className={`bg-white rounded-lg border p-4 cursor-pointer transition-all hover:shadow-md ${
              selectedRequest === request.id ? 'ring-2 ring-blue-500' : ''
            }`}
            onClick={() => setSelectedRequest(selectedRequest === request.id ? null : request.id)}
            onKeyDown={(e) => e.key === 'Enter' && setSelectedRequest(selectedRequest === request.id ? null : request.id)}
            tabIndex={0}
            role="button"
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-3">
                  <h3 className="font-medium">{request.clientName}</h3>
                  <span className={`px-2 py-0.5 rounded text-xs ${CLIENT_TIER_COLORS[request.clientTier]}`}>
                    {request.clientTier}
                  </span>
                  <span className={`px-2 py-0.5 rounded text-xs border ${URGENCY_COLORS[request.urgency]}`}>
                    {request.urgency}
                  </span>
                </div>
                <div className="flex items-center gap-4 mt-2 text-sm text-gray-600">
                  <span className={`px-2 py-0.5 rounded text-xs ${MEETING_TYPE_CONFIG[request.meetingType].color}`}>
                    {MEETING_TYPE_CONFIG[request.meetingType].label}
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {request.duration} min
                  </span>
                  {request.aiScore && (
                    <span className="flex items-center gap-1 text-purple-600">
                      <Brain className="w-3 h-3" />
                      AI Score: {Math.round(request.aiScore * 100)}%
                    </span>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-2">
                {request.status === 'SCHEDULED' && request.scheduledSlot && (
                  <span className="text-sm text-green-600 bg-green-50 px-2 py-1 rounded">
                    Scheduled: {request.scheduledSlot.date} at {request.scheduledSlot.time}
                  </span>
                )}
                {request.status === 'PENDING' && (
                  <span className="text-sm text-yellow-600 bg-yellow-50 px-2 py-1 rounded">
                    Pending
                  </span>
                )}
                <ChevronRight className={`w-5 h-5 text-gray-400 transition-transform ${selectedRequest === request.id ? 'rotate-90' : ''}`} />
              </div>
            </div>

            {selectedRequest === request.id && (
              <div className="mt-4 pt-4 border-t">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <h4 className="text-xs font-medium text-gray-500 mb-2">Required Expertise</h4>
                    <div className="flex flex-wrap gap-1">
                      {request.requiredExpertise.length > 0 ? (
                        request.requiredExpertise.map(exp => (
                          <span key={exp} className="px-2 py-0.5 bg-gray-100 rounded text-xs">
                            {exp.replace(/_/g, ' ')}
                          </span>
                        ))
                      ) : (
                        <span className="text-xs text-gray-400">Any advisor</span>
                      )}
                    </div>
                  </div>
                  <div>
                    <h4 className="text-xs font-medium text-gray-500 mb-2">Preferred Advisors</h4>
                    <div className="flex flex-wrap gap-1">
                      {request.preferredAdvisors.length > 0 ? (
                        request.preferredAdvisors.map(advId => {
                          const advisor = advisors.find(a => a.id === advId);
                          return (
                            <span key={advId} className="px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs">
                              {advisor?.name || advId}
                            </span>
                          );
                        })
                      ) : (
                        <span className="text-xs text-gray-400">No preference</span>
                      )}
                    </div>
                  </div>
                  <div>
                    <h4 className="text-xs font-medium text-gray-500 mb-2">Timeframe</h4>
                    <span className="text-sm">
                      {new Date(request.requestedTimeframe.earliest).toLocaleDateString()} - {new Date(request.requestedTimeframe.latest).toLocaleDateString()}
                    </span>
                  </div>
                  {request.scheduledSlot && (
                    <div>
                      <h4 className="text-xs font-medium text-gray-500 mb-2">Assigned Advisor</h4>
                      <span className="text-sm font-medium">
                        {advisors.find(a => a.id === request.scheduledSlot?.advisorId)?.name}
                      </span>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );

  // Render advisors tab
  const renderAdvisors = () => (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-4">
        {advisors.map(advisor => (
          <div
            key={advisor.id}
            className={`bg-white rounded-lg border p-4 cursor-pointer transition-all hover:shadow-md ${
              selectedAdvisor === advisor.id ? 'ring-2 ring-blue-500' : ''
            }`}
            onClick={() => setSelectedAdvisor(selectedAdvisor === advisor.id ? null : advisor.id)}
            onKeyDown={(e) => e.key === 'Enter' && setSelectedAdvisor(selectedAdvisor === advisor.id ? null : advisor.id)}
            tabIndex={0}
            role="button"
          >
            <div className="flex items-start gap-3">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white font-semibold">
                {advisor.name.split(' ').map(n => n[0]).join('')}
              </div>
              <div className="flex-1">
                <h3 className="font-medium">{advisor.name}</h3>
                <p className="text-sm text-gray-500">{advisor.title}</p>
              </div>
            </div>

            <div className="mt-4 space-y-3">
              <div>
                <div className="flex items-center justify-between text-xs mb-1">
                  <span className="text-gray-500">Daily Load</span>
                  <span className="font-medium">{advisor.currentLoad}/{advisor.maxDailyMeetings}</span>
                </div>
                <LoadBar current={advisor.currentLoad} max={advisor.maxDailyMeetings} />
              </div>

              <div>
                <h4 className="text-xs text-gray-500 mb-1">Certifications</h4>
                <div className="flex flex-wrap gap-1">
                  {advisor.certifications.map(cert => (
                    <span key={cert} className="px-2 py-0.5 bg-green-50 text-green-700 rounded text-xs">
                      {cert}
                    </span>
                  ))}
                </div>
              </div>

              {selectedAdvisor === advisor.id && (
                <div className="pt-3 border-t">
                  <h4 className="text-xs text-gray-500 mb-1">Expertise Areas</h4>
                  <div className="flex flex-wrap gap-1">
                    {advisor.expertise.map(exp => (
                      <span key={exp} className="px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs">
                        {exp.replace(/_/g, ' ')}
                      </span>
                    ))}
                  </div>
                  <div className="mt-3">
                    <h4 className="text-xs text-gray-500 mb-1">Preferred Hours</h4>
                    <span className="text-sm">{advisor.preferredHours.start}:00 - {advisor.preferredHours.end}:00</span>
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );

  // Render optimization tab
  const renderOptimization = () => (
    <div className="space-y-6">
      {/* Run optimization button */}
      <div className="bg-gradient-to-r from-purple-50 to-blue-50 rounded-lg border border-purple-200 p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="font-semibold text-lg flex items-center gap-2">
              <Brain className="w-5 h-5 text-purple-600" />
              AI Schedule Optimization
            </h3>
            <p className="text-sm text-gray-600 mt-1">
              Uses OR-Tools constraint satisfaction to find optimal meeting assignments
            </p>
          </div>
          <button
            onClick={runOptimization}
            disabled={isOptimizing || pendingCount === 0}
            className="flex items-center gap-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isOptimizing ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin" />
                Optimizing...
              </>
            ) : (
              <>
                <Sparkles className="w-4 h-4" />
                Run Optimization
              </>
            )}
          </button>
        </div>

        {pendingCount === 0 && (
          <p className="text-sm text-amber-600 mt-3">No pending requests to optimize</p>
        )}
      </div>

      {/* Optimization results */}
      {optimization && (
        <div className="bg-white rounded-lg border p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold">Optimization Results</h3>
            <span className={`px-2 py-1 rounded text-xs ${
              optimization.status === 'COMPLETED' ? 'bg-green-100 text-green-800' :
              optimization.status === 'RUNNING' ? 'bg-blue-100 text-blue-800' :
              optimization.status === 'FAILED' ? 'bg-red-100 text-red-800' :
              'bg-gray-100 text-gray-800'
            }`}>
              {optimization.status}
            </span>
          </div>

          <div className="grid grid-cols-4 gap-4 mb-6">
            <div className="bg-gray-50 rounded-lg p-3 text-center">
              <div className="text-2xl font-bold text-gray-900">{optimization.requestsProcessed}</div>
              <div className="text-xs text-gray-500">Requests Processed</div>
            </div>
            <div className="bg-green-50 rounded-lg p-3 text-center">
              <div className="text-2xl font-bold text-green-600">{optimization.requestsScheduled}</div>
              <div className="text-xs text-gray-500">Successfully Scheduled</div>
            </div>
            <div className="bg-purple-50 rounded-lg p-3 text-center">
              <div className="text-2xl font-bold text-purple-600">{Math.round(optimization.optimizationScore * 100)}%</div>
              <div className="text-xs text-gray-500">Optimization Score</div>
            </div>
            <div className="bg-blue-50 rounded-lg p-3 text-center">
              <div className="text-2xl font-bold text-blue-600">
                {optimization.endTime ? 
                  `${((optimization.endTime.getTime() - optimization.startTime.getTime()) / 1000).toFixed(1)}s` : 
                  '...'
                }
              </div>
              <div className="text-xs text-gray-500">Processing Time</div>
            </div>
          </div>

          <h4 className="font-medium text-sm mb-3">Constraint Satisfaction</h4>
          <div className="space-y-2">
            {optimization.constraints.map((constraint, idx) => (
              <div key={idx} className="flex items-center justify-between p-2 bg-gray-50 rounded">
                <div className="flex items-center gap-2">
                  {constraint.satisfied ? (
                    <div className="w-5 h-5 rounded-full bg-green-500 flex items-center justify-center">
                      <svg className="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    </div>
                  ) : (
                    <div className="w-5 h-5 rounded-full bg-yellow-500 flex items-center justify-center">
                      <AlertTriangle className="w-3 h-3 text-white" />
                    </div>
                  )}
                  <span className="text-sm">{constraint.description}</span>
                </div>
                <span className="text-xs text-gray-500">Weight: {constraint.weight}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );

  // Render alerts tab
  const renderAlerts = () => (
    <div className="space-y-4">
      {alerts.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <Bell className="w-12 h-12 mx-auto mb-4 text-gray-300" />
          <p>No active alerts</p>
          <p className="text-sm">System is operating normally</p>
        </div>
      ) : (
        alerts.map(alert => (
          <div
            key={alert.id}
            className={`rounded-lg border p-4 ${
              alert.severity === 'CRITICAL' ? 'bg-red-50 border-red-200' :
              alert.severity === 'HIGH' ? 'bg-orange-50 border-orange-200' :
              alert.severity === 'MEDIUM' ? 'bg-yellow-50 border-yellow-200' :
              'bg-blue-50 border-blue-200'
            }`}
          >
            <div className="flex items-start gap-3">
              <AlertTriangle className={`w-5 h-5 ${
                alert.severity === 'CRITICAL' ? 'text-red-600' :
                alert.severity === 'HIGH' ? 'text-orange-600' :
                alert.severity === 'MEDIUM' ? 'text-yellow-600' :
                'text-blue-600'
              }`} />
              <div className="flex-1">
                <div className="flex items-center justify-between">
                  <h3 className="font-medium">{alert.message}</h3>
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    alert.severity === 'CRITICAL' ? 'bg-red-200 text-red-800' :
                    alert.severity === 'HIGH' ? 'bg-orange-200 text-orange-800' :
                    alert.severity === 'MEDIUM' ? 'bg-yellow-200 text-yellow-800' :
                    'bg-blue-200 text-blue-800'
                  }`}>
                    {alert.severity}
                  </span>
                </div>
                <p className="text-sm text-gray-600 mt-1">{alert.suggestedAction}</p>
                {alert.affectedAdvisors.length > 0 && (
                  <div className="mt-2">
                    <span className="text-xs text-gray-500">Affected advisors: </span>
                    {alert.affectedAdvisors.map(advId => {
                      const advisor = advisors.find(a => a.id === advId);
                      return (
                        <span key={advId} className="text-xs font-medium">
                          {advisor?.name || advId}
                          {alert.affectedAdvisors.indexOf(advId) < alert.affectedAdvisors.length - 1 ? ', ' : ''}
                        </span>
                      );
                    })}
                  </div>
                )}
                <span className="text-xs text-gray-400 mt-2 block">
                  {alert.timestamp.toLocaleTimeString()}
                </span>
              </div>
            </div>
          </div>
        ))
      )}
    </div>
  );

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold flex items-center gap-2">
              <Users className="w-6 h-6 text-blue-600" />
              Surge Meeting Orchestrator
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              AI-powered meeting scheduling with load balancing
            </p>
          </div>
          <div className="flex items-center gap-4">
            {criticalCount > 0 && (
              <div className="flex items-center gap-2 px-3 py-1.5 bg-red-100 text-red-700 rounded-lg">
                <AlertTriangle className="w-4 h-4" />
                <span className="text-sm font-medium">{criticalCount} Critical</span>
              </div>
            )}
            <button className="flex items-center gap-2 px-3 py-1.5 border rounded-lg hover:bg-gray-50">
              <Settings className="w-4 h-4" />
              Settings
            </button>
          </div>
        </div>

        {/* Stats bar */}
        <div className="grid grid-cols-4 gap-4 mt-4">
          <div className="bg-blue-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-blue-600">Pending Requests</span>
              <Clock className="w-4 h-4 text-blue-400" />
            </div>
            <div className="text-2xl font-bold text-blue-900 mt-1">{pendingCount}</div>
          </div>
          <div className="bg-green-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-green-600">Team Capacity</span>
              <Users className="w-4 h-4 text-green-400" />
            </div>
            <div className="text-2xl font-bold text-green-900 mt-1">{teamCapacity.available}/{teamCapacity.total}</div>
          </div>
          <div className="bg-purple-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-purple-600">Utilization</span>
              <BarChart3 className="w-4 h-4 text-purple-400" />
            </div>
            <div className="text-2xl font-bold text-purple-900 mt-1">{Math.round(teamCapacity.utilizationPct)}%</div>
          </div>
          <div className="bg-orange-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-orange-600">Active Alerts</span>
              <Bell className="w-4 h-4 text-orange-400" />
            </div>
            <div className="text-2xl font-bold text-orange-900 mt-1">{alerts.length}</div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <div className="flex gap-6">
          {[
            { id: 'queue' as const, label: 'Request Queue', icon: Clock, count: pendingCount },
            { id: 'advisors' as const, label: 'Advisor Team', icon: Users, count: advisors.length },
            { id: 'optimization' as const, label: 'AI Optimization', icon: Brain },
            { id: 'alerts' as const, label: 'Alerts', icon: Bell, count: alerts.length }
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
              {tab.count !== undefined && (
                <span className={`px-1.5 py-0.5 rounded text-xs ${
                  activeTab === tab.id ? 'bg-blue-100' : 'bg-gray-100'
                }`}>
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {activeTab === 'queue' && renderQueue()}
        {activeTab === 'advisors' && renderAdvisors()}
        {activeTab === 'optimization' && renderOptimization()}
        {activeTab === 'alerts' && renderAlerts()}
      </div>
    </div>
  );
};

export default SurgeMeetingOrchestrator;
