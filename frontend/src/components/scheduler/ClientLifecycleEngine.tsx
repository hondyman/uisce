/**
 * ClientLifecycleEngine.tsx
 * 
 * Client Lifecycle Event Intelligence Engine:
 * - Life event detection (retirement, inheritance, job change, home purchase)
 * - AI-driven event prediction with certainty scoring
 * - Automatic workflow triggering based on lifecycle stage
 * - Proactive planning timeline visualization
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Calendar,
  User,
  Home,
  Briefcase,
  Gift,
  TrendingUp,
  Clock,
  AlertCircle,
  Check,
  ChevronRight,
  ChevronDown,
  Plus,
  Settings,
  Zap,
  Target,
  Bell,
  Brain,
  Sparkles,
  FileText
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface LifecycleEvent {
  id: string;
  clientId: string;
  clientName: string;
  eventType: LifecycleEventType;
  eventDate: Date;
  anticipatedDate?: Date;
  certaintyScore: number; // 0.0 to 1.0
  triggerSource: 'CLIENT_DECLARED' | 'AI_DETECTED' | 'CRM_IMPORTED' | 'ADVISOR_ENTERED';
  status: 'UPCOMING' | 'IN_PROGRESS' | 'COMPLETED' | 'CANCELLED';
  metadata: Record<string, unknown>;
  triggeredWorkflows: TriggeredWorkflow[];
}

type LifecycleEventType = 
  | 'RETIREMENT'
  | 'HOME_PURCHASE'
  | 'INHERITANCE'
  | 'JOB_CHANGE'
  | 'MARRIAGE'
  | 'DIVORCE'
  | 'CHILD_BIRTH'
  | 'COLLEGE_FUNDING'
  | 'BUSINESS_SALE'
  | 'MAJOR_HEALTH_EVENT';

interface TriggeredWorkflow {
  id: string;
  templateName: string;
  daysBeforeEvent: number;
  scheduledDate: Date;
  status: 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED';
  description: string;
}

interface LifecycleJobTemplate {
  id: string;
  eventType: LifecycleEventType;
  daysBeforeEvent: number;
  jobTemplate: {
    name: string;
    type: string;
    actions: string[];
  };
  priority: number;
  description: string;
}

interface ClientLifecycleEngineProps {
  tenantId: string;
  onEventSelect?: (event: LifecycleEvent) => void;
  onWorkflowTrigger?: (workflow: TriggeredWorkflow) => void;
}

// ============================================================================
// Constants
// ============================================================================

const EVENT_TYPE_CONFIG: Record<LifecycleEventType, { 
  icon: React.ElementType; 
  color: string; 
  label: string;
  bgColor: string;
}> = {
  RETIREMENT: { icon: Calendar, color: 'text-blue-600', bgColor: 'bg-blue-100', label: 'Retirement' },
  HOME_PURCHASE: { icon: Home, color: 'text-green-600', bgColor: 'bg-green-100', label: 'Home Purchase' },
  INHERITANCE: { icon: Gift, color: 'text-purple-600', bgColor: 'bg-purple-100', label: 'Inheritance' },
  JOB_CHANGE: { icon: Briefcase, color: 'text-orange-600', bgColor: 'bg-orange-100', label: 'Job Change' },
  MARRIAGE: { icon: User, color: 'text-pink-600', bgColor: 'bg-pink-100', label: 'Marriage' },
  DIVORCE: { icon: User, color: 'text-red-600', bgColor: 'bg-red-100', label: 'Divorce' },
  CHILD_BIRTH: { icon: User, color: 'text-cyan-600', bgColor: 'bg-cyan-100', label: 'Child Birth' },
  COLLEGE_FUNDING: { icon: FileText, color: 'text-indigo-600', bgColor: 'bg-indigo-100', label: 'College Funding' },
  BUSINESS_SALE: { icon: TrendingUp, color: 'text-emerald-600', bgColor: 'bg-emerald-100', label: 'Business Sale' },
  MAJOR_HEALTH_EVENT: { icon: AlertCircle, color: 'text-amber-600', bgColor: 'bg-amber-100', label: 'Health Event' }
};

// Progress bar component to avoid inline styles
const PROGRESS_WIDTH_CLASSES = [
  'w-0', 'w-[10%]', 'w-[20%]', 'w-[30%]', 'w-[40%]', 
  'w-[50%]', 'w-[60%]', 'w-[70%]', 'w-[80%]', 'w-[90%]', 'w-full'
];

const ProgressBar: React.FC<{ completed: number; total: number }> = ({ completed, total }) => {
  const percentage = total > 0 ? Math.round((completed / total) * 10) : 0;
  const widthClass = PROGRESS_WIDTH_CLASSES[Math.min(percentage, 10)];
  return <div className={`h-full bg-green-500 rounded-full transition-all ${widthClass}`} />;
};

const _DEFAULT_JOB_TEMPLATES: LifecycleJobTemplate[] = [
  // Retirement templates
  { id: 't1', eventType: 'RETIREMENT', daysBeforeEvent: -90, priority: 1, description: 'Tax optimization review', jobTemplate: { name: 'Pre-Retirement Tax Review', type: 'MEETING', actions: ['schedule_meeting', 'generate_tax_report'] } },
  { id: 't2', eventType: 'RETIREMENT', daysBeforeEvent: -60, priority: 2, description: 'Social Security election strategy', jobTemplate: { name: 'Social Security Analysis', type: 'REPORT', actions: ['generate_ss_report', 'schedule_review'] } },
  { id: 't3', eventType: 'RETIREMENT', daysBeforeEvent: -30, priority: 3, description: 'Portfolio rebalancing to income focus', jobTemplate: { name: 'Retirement Rebalance', type: 'REBALANCE', actions: ['analyze_portfolio', 'propose_trades', 'compliance_review'] } },
  { id: 't4', eventType: 'RETIREMENT', daysBeforeEvent: 0, priority: 4, description: 'Initiate retiree billing cadence', jobTemplate: { name: 'Retirement Day Setup', type: 'ADMIN', actions: ['update_billing', 'schedule_quarterly_reviews', 'send_congratulations'] } },
  
  // Inheritance templates
  { id: 't5', eventType: 'INHERITANCE', daysBeforeEvent: 0, priority: 1, description: 'Initial inheritance review meeting', jobTemplate: { name: 'Inheritance Consultation', type: 'MEETING', actions: ['schedule_meeting', 'prepare_estate_checklist'] } },
  { id: 't6', eventType: 'INHERITANCE', daysBeforeEvent: 7, priority: 2, description: 'Tax implications analysis', jobTemplate: { name: 'Inheritance Tax Analysis', type: 'REPORT', actions: ['analyze_tax_impact', 'identify_opportunities'] } },
  { id: 't7', eventType: 'INHERITANCE', daysBeforeEvent: 30, priority: 3, description: 'Integration into portfolio strategy', jobTemplate: { name: 'Portfolio Integration', type: 'REBALANCE', actions: ['update_financial_plan', 'propose_allocation'] } },
  
  // Home Purchase templates
  { id: 't8', eventType: 'HOME_PURCHASE', daysBeforeEvent: -60, priority: 1, description: 'Liquidity planning review', jobTemplate: { name: 'Home Purchase Liquidity Review', type: 'MEETING', actions: ['analyze_liquidity', 'identify_funding_sources'] } },
  { id: 't9', eventType: 'HOME_PURCHASE', daysBeforeEvent: -30, priority: 2, description: 'Down payment strategy', jobTemplate: { name: 'Down Payment Planning', type: 'REPORT', actions: ['calculate_options', 'tax_efficient_withdrawal'] } },
  { id: 't10', eventType: 'HOME_PURCHASE', daysBeforeEvent: 14, priority: 3, description: 'Post-purchase portfolio review', jobTemplate: { name: 'Post-Purchase Rebalance', type: 'REBALANCE', actions: ['update_allocation', 'adjust_risk_profile'] } },
];

// ============================================================================
// Mock Data Generation
// ============================================================================

const generateMockEvents = (): LifecycleEvent[] => {
  const now = new Date();
  
  return [
    {
      id: 'evt-1',
      clientId: 'client-001',
      clientName: 'Robert & Susan Mitchell',
      eventType: 'RETIREMENT',
      eventDate: new Date(now.getTime() + 75 * 24 * 60 * 60 * 1000), // 75 days from now
      anticipatedDate: new Date(now.getTime() + 75 * 24 * 60 * 60 * 1000),
      certaintyScore: 0.95,
      triggerSource: 'CLIENT_DECLARED',
      status: 'UPCOMING',
      metadata: { targetAge: 65, currentAge: 64.8, retirementIncome: 150000 },
      triggeredWorkflows: [
        { id: 'wf-1', templateName: 'Pre-Retirement Tax Review', daysBeforeEvent: -90, scheduledDate: new Date(now.getTime() - 15 * 24 * 60 * 60 * 1000), status: 'COMPLETED', description: 'Tax optimization review' },
        { id: 'wf-2', templateName: 'Social Security Analysis', daysBeforeEvent: -60, scheduledDate: new Date(now.getTime() + 15 * 24 * 60 * 60 * 1000), status: 'PENDING', description: 'Social Security election strategy' },
        { id: 'wf-3', templateName: 'Retirement Rebalance', daysBeforeEvent: -30, scheduledDate: new Date(now.getTime() + 45 * 24 * 60 * 60 * 1000), status: 'PENDING', description: 'Portfolio rebalancing to income focus' }
      ]
    },
    {
      id: 'evt-2',
      clientId: 'client-002',
      clientName: 'Jennifer Walsh',
      eventType: 'INHERITANCE',
      eventDate: new Date(now.getTime() + 14 * 24 * 60 * 60 * 1000),
      certaintyScore: 0.88,
      triggerSource: 'AI_DETECTED',
      status: 'UPCOMING',
      metadata: { estimatedValue: 2500000, assetTypes: ['Real Estate', 'Securities', 'Cash'] },
      triggeredWorkflows: [
        { id: 'wf-4', templateName: 'Inheritance Consultation', daysBeforeEvent: 0, scheduledDate: new Date(now.getTime() + 14 * 24 * 60 * 60 * 1000), status: 'PENDING', description: 'Initial inheritance review meeting' }
      ]
    },
    {
      id: 'evt-3',
      clientId: 'client-003',
      clientName: 'Michael & Emily Chen',
      eventType: 'HOME_PURCHASE',
      eventDate: new Date(now.getTime() + 45 * 24 * 60 * 60 * 1000),
      anticipatedDate: new Date(now.getTime() + 45 * 24 * 60 * 60 * 1000),
      certaintyScore: 0.72,
      triggerSource: 'CRM_IMPORTED',
      status: 'UPCOMING',
      metadata: { targetPrice: 1200000, downPaymentPercent: 20 },
      triggeredWorkflows: [
        { id: 'wf-5', templateName: 'Home Purchase Liquidity Review', daysBeforeEvent: -60, scheduledDate: new Date(now.getTime() - 15 * 24 * 60 * 60 * 1000), status: 'COMPLETED', description: 'Liquidity planning review' },
        { id: 'wf-6', templateName: 'Down Payment Planning', daysBeforeEvent: -30, scheduledDate: new Date(now.getTime() + 15 * 24 * 60 * 60 * 1000), status: 'RUNNING', description: 'Down payment strategy' }
      ]
    },
    {
      id: 'evt-4',
      clientId: 'client-004',
      clientName: 'David Thompson',
      eventType: 'BUSINESS_SALE',
      eventDate: new Date(now.getTime() + 180 * 24 * 60 * 60 * 1000),
      certaintyScore: 0.65,
      triggerSource: 'AI_DETECTED',
      status: 'UPCOMING',
      metadata: { businessValue: 5000000, industry: 'Technology Services' },
      triggeredWorkflows: []
    },
    {
      id: 'evt-5',
      clientId: 'client-005',
      clientName: 'Sarah & James Wilson',
      eventType: 'COLLEGE_FUNDING',
      eventDate: new Date(now.getTime() + 365 * 24 * 60 * 60 * 1000),
      anticipatedDate: new Date(now.getTime() + 365 * 24 * 60 * 60 * 1000),
      certaintyScore: 0.98,
      triggerSource: 'CLIENT_DECLARED',
      status: 'UPCOMING',
      metadata: { childName: 'Emma', targetSchool: 'Private University', estimatedCost: 280000 },
      triggeredWorkflows: []
    }
  ];
};

// ============================================================================
// Components
// ============================================================================

const CertaintyBadge: React.FC<{ score: number }> = ({ score }) => {
  const getColor = () => {
    if (score >= 0.9) return 'bg-green-100 text-green-800';
    if (score >= 0.7) return 'bg-yellow-100 text-yellow-800';
    return 'bg-orange-100 text-orange-800';
  };
  
  return (
    <span className={`text-xs px-2 py-0.5 rounded-full ${getColor()}`}>
      {(score * 100).toFixed(0)}% certain
    </span>
  );
};

const WorkflowTimeline: React.FC<{
  workflows: TriggeredWorkflow[];
  eventDate: Date;
}> = ({ workflows, eventDate }) => {
  const sortedWorkflows = useMemo(() => 
    [...workflows].sort((a, b) => a.daysBeforeEvent - b.daysBeforeEvent),
    [workflows]
  );

  const getStatusIcon = (status: TriggeredWorkflow['status']) => {
    switch (status) {
      case 'COMPLETED': return <Check size={14} className="text-green-500" />;
      case 'RUNNING': return <Zap size={14} className="text-blue-500 animate-pulse" />;
      case 'PENDING': return <Clock size={14} className="text-gray-400" />;
      case 'FAILED': return <AlertCircle size={14} className="text-red-500" />;
    }
  };

  const getStatusColor = (status: TriggeredWorkflow['status']) => {
    switch (status) {
      case 'COMPLETED': return 'border-green-500 bg-green-50';
      case 'RUNNING': return 'border-blue-500 bg-blue-50';
      case 'PENDING': return 'border-gray-300 bg-gray-50';
      case 'FAILED': return 'border-red-500 bg-red-50';
    }
  };

  return (
    <div className="relative pl-6 space-y-3">
      {/* Timeline line */}
      <div className="absolute left-2 top-2 bottom-2 w-0.5 bg-gray-200" />
      
      {sortedWorkflows.map((workflow, _index) => (
        <div key={workflow.id} className="relative">
          {/* Timeline dot */}
          <div className={`absolute -left-4 w-4 h-4 rounded-full border-2 flex items-center justify-center ${getStatusColor(workflow.status)}`}>
            {getStatusIcon(workflow.status)}
          </div>
          
          <div className={`ml-2 p-3 rounded border ${getStatusColor(workflow.status)}`}>
            <div className="flex items-center justify-between mb-1">
              <span className="font-medium text-sm">{workflow.templateName}</span>
              <span className="text-xs text-gray-500">
                {workflow.daysBeforeEvent === 0 
                  ? 'Event Day' 
                  : workflow.daysBeforeEvent < 0 
                    ? `${Math.abs(workflow.daysBeforeEvent)} days before`
                    : `${workflow.daysBeforeEvent} days after`
                }
              </span>
            </div>
            <p className="text-xs text-gray-600">{workflow.description}</p>
            <div className="text-xs text-gray-400 mt-1">
              Scheduled: {new Date(workflow.scheduledDate).toLocaleDateString()}
            </div>
          </div>
        </div>
      ))}
      
      {/* Event marker */}
      <div className="relative">
        <div className="absolute -left-4 w-4 h-4 rounded-full border-2 border-purple-500 bg-purple-100 flex items-center justify-center">
          <Target size={10} className="text-purple-600" />
        </div>
        <div className="ml-2 p-3 rounded border-2 border-purple-300 bg-purple-50">
          <span className="font-semibold text-purple-800">Event Date</span>
          <div className="text-xs text-purple-600">
            {eventDate.toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
          </div>
        </div>
      </div>
    </div>
  );
};

const EventCard: React.FC<{
  event: LifecycleEvent;
  isExpanded: boolean;
  onToggle: () => void;
  onSelect: () => void;
}> = ({ event, isExpanded, onToggle, onSelect }) => {
  const config = EVENT_TYPE_CONFIG[event.eventType];
  const Icon = config.icon;
  const daysUntil = Math.ceil((new Date(event.eventDate).getTime() - Date.now()) / (1000 * 60 * 60 * 24));
  
  return (
    <div className="bg-white rounded-lg border shadow-sm overflow-hidden">
      <div 
        className="p-4 cursor-pointer hover:bg-gray-50 transition-colors"
        onClick={onToggle}
      >
        <div className="flex items-start gap-3">
          <div className={`p-2 rounded-lg ${config.bgColor}`}>
            <Icon size={20} className={config.color} />
          </div>
          
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="font-semibold text-gray-900">{event.clientName}</h3>
              <CertaintyBadge score={event.certaintyScore} />
            </div>
            <div className="flex items-center gap-2 text-sm">
              <span className={`font-medium ${config.color}`}>{config.label}</span>
              <span className="text-gray-400">•</span>
              <span className="text-gray-600">
                {daysUntil > 0 ? `${daysUntil} days away` : daysUntil === 0 ? 'Today' : `${Math.abs(daysUntil)} days ago`}
              </span>
            </div>
            <div className="flex items-center gap-2 mt-2 text-xs text-gray-500">
              <span className="flex items-center gap-1">
                {event.triggerSource === 'AI_DETECTED' && <Brain size={12} className="text-purple-500" />}
                {event.triggerSource === 'CLIENT_DECLARED' && <User size={12} />}
                {event.triggerSource === 'CRM_IMPORTED' && <FileText size={12} />}
                {event.triggerSource === 'ADVISOR_ENTERED' && <User size={12} />}
                {event.triggerSource.replace('_', ' ').toLowerCase()}
              </span>
              <span>•</span>
              <span>{event.triggeredWorkflows.length} workflows</span>
              <span>•</span>
              <span>{event.triggeredWorkflows.filter(w => w.status === 'COMPLETED').length} completed</span>
            </div>
          </div>
          
          <div className="flex items-center gap-2">
            {daysUntil <= 30 && daysUntil > 0 && (
              <span className="text-xs px-2 py-1 bg-amber-100 text-amber-800 rounded-full flex items-center gap-1">
                <Bell size={12} />
                Approaching
              </span>
            )}
            {isExpanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
          </div>
        </div>
      </div>
      
      {isExpanded && (
        <div className="border-t bg-gray-50 p-4">
          <div className="grid grid-cols-2 gap-4 mb-4">
            <div>
              <h4 className="text-xs font-medium text-gray-500 mb-2">Event Details</h4>
              <div className="bg-white rounded border p-3 space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Event Date:</span>
                  <span className="font-medium">{new Date(event.eventDate).toLocaleDateString()}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Status:</span>
                  <span className="font-medium">{event.status}</span>
                </div>
                {Object.entries(event.metadata).slice(0, 3).map(([key, value]) => (
                  <div key={key} className="flex justify-between text-sm">
                    <span className="text-gray-500">{key.replace(/([A-Z])/g, ' $1').trim()}:</span>
                    <span className="font-medium">
                      {typeof value === 'number' && key.toLowerCase().includes('value') 
                        ? `$${value.toLocaleString()}`
                        : String(value)
                      }
                    </span>
                  </div>
                ))}
              </div>
            </div>
            
            <div>
              <h4 className="text-xs font-medium text-gray-500 mb-2">Workflow Progress</h4>
              <div className="bg-white rounded border p-3">
                <div className="flex items-center gap-2 mb-2">
                  <div className="flex-1 h-2 bg-gray-200 rounded-full overflow-hidden">
                    <ProgressBar 
                      completed={event.triggeredWorkflows.filter(w => w.status === 'COMPLETED').length}
                      total={event.triggeredWorkflows.length}
                    />
                  </div>
                  <span className="text-sm font-medium">
                    {event.triggeredWorkflows.filter(w => w.status === 'COMPLETED').length}/{event.triggeredWorkflows.length}
                  </span>
                </div>
                <div className="flex gap-2 text-xs">
                  <span className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full bg-green-500" />
                    Completed
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full bg-blue-500" />
                    Running
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full bg-gray-300" />
                    Pending
                  </span>
                </div>
              </div>
            </div>
          </div>
          
          <div>
            <h4 className="text-xs font-medium text-gray-500 mb-2">Workflow Timeline</h4>
            <WorkflowTimeline 
              workflows={event.triggeredWorkflows} 
              eventDate={new Date(event.eventDate)} 
            />
          </div>
          
          <div className="flex justify-end gap-2 mt-4 pt-4 border-t">
            <button
              onClick={onSelect}
              className="px-3 py-1.5 text-sm border rounded hover:bg-gray-100 transition-colors"
            >
              View Client
            </button>
            <button className="px-3 py-1.5 text-sm border rounded hover:bg-gray-100 transition-colors flex items-center gap-1">
              <Plus size={14} />
              Add Workflow
            </button>
            <button className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors flex items-center gap-1">
              <Settings size={14} />
              Configure
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

const AIInsightsPanel: React.FC<{ events: LifecycleEvent[] }> = ({ events }) => {
  const aiDetectedCount = events.filter(e => e.triggerSource === 'AI_DETECTED').length;
  const upcomingUrgent = events.filter(e => {
    const daysUntil = Math.ceil((new Date(e.eventDate).getTime() - Date.now()) / (1000 * 60 * 60 * 24));
    return daysUntil <= 30 && daysUntil > 0;
  }).length;
  
  return (
    <div className="bg-gradient-to-r from-purple-50 to-indigo-50 rounded-lg border border-purple-200 p-4">
      <div className="flex items-center gap-2 mb-3">
        <Sparkles size={18} className="text-purple-600" />
        <h3 className="font-semibold text-purple-900">AI Lifecycle Insights</h3>
      </div>
      
      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white/60 rounded p-3">
          <div className="text-2xl font-bold text-purple-700">{aiDetectedCount}</div>
          <div className="text-xs text-purple-600">AI-Detected Events</div>
        </div>
        <div className="bg-white/60 rounded p-3">
          <div className="text-2xl font-bold text-amber-700">{upcomingUrgent}</div>
          <div className="text-xs text-amber-600">Urgent (≤30 days)</div>
        </div>
        <div className="bg-white/60 rounded p-3">
          <div className="text-2xl font-bold text-green-700">
            {events.reduce((sum, e) => sum + e.triggeredWorkflows.filter(w => w.status === 'COMPLETED').length, 0)}
          </div>
          <div className="text-xs text-green-600">Workflows Completed</div>
        </div>
      </div>
      
      {aiDetectedCount > 0 && (
        <div className="mt-3 text-sm text-purple-700">
          <Brain size={14} className="inline mr-1" />
          AI detected {aiDetectedCount} potential life events from CRM data, emails, and account activity patterns.
        </div>
      )}
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

export const ClientLifecycleEngine: React.FC<ClientLifecycleEngineProps> = ({
  tenantId: _tenantId,
  onEventSelect,
  onWorkflowTrigger: _onWorkflowTrigger
}) => {
  const [events, setEvents] = useState<LifecycleEvent[]>([]);
  const [expandedEvent, setExpandedEvent] = useState<string | null>(null);
  const [filterType, setFilterType] = useState<LifecycleEventType | 'ALL'>('ALL');
  const [filterStatus, setFilterStatus] = useState<'ALL' | 'UPCOMING' | 'IN_PROGRESS'>('UPCOMING');
  const [_showAddModal, setShowAddModal] = useState(false);

  useEffect(() => {
    setEvents(generateMockEvents());
  }, []);

  const filteredEvents = useMemo(() => {
    return events.filter(event => {
      if (filterType !== 'ALL' && event.eventType !== filterType) return false;
      if (filterStatus !== 'ALL' && event.status !== filterStatus) return false;
      return true;
    });
  }, [events, filterType, filterStatus]);

  const handleToggleExpand = useCallback((eventId: string) => {
    setExpandedEvent(prev => prev === eventId ? null : eventId);
  }, []);

  return (
    <div className="flex flex-col h-full bg-gray-50">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b bg-white">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-indigo-100 rounded-lg">
            <Calendar size={20} className="text-indigo-600" />
          </div>
          <div>
            <h2 className="font-semibold text-gray-900">Client Lifecycle Intelligence</h2>
            <p className="text-xs text-gray-500">Proactive life-stage event management</p>
          </div>
        </div>
        
        <button
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
        >
          <Plus size={16} />
          Add Event
        </button>
      </div>
      
      {/* AI Insights */}
      <div className="p-4">
        <AIInsightsPanel events={events} />
      </div>
      
      {/* Filters */}
      <div className="px-4 pb-4 flex items-center gap-4">
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-500">Event Type:</span>
          <select
            value={filterType}
            onChange={(e) => setFilterType(e.target.value as LifecycleEventType | 'ALL')}
            className="px-3 py-1.5 border rounded text-sm"
            title="Filter by event type"
            aria-label="Filter by event type"
          >
            <option value="ALL">All Types</option>
            {Object.entries(EVENT_TYPE_CONFIG).map(([type, config]) => (
              <option key={type} value={type}>{config.label}</option>
            ))}
          </select>
        </div>
        
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-500">Status:</span>
          <select
            value={filterStatus}
            onChange={(e) => setFilterStatus(e.target.value as typeof filterStatus)}
            className="px-3 py-1.5 border rounded text-sm"
            title="Filter by status"
            aria-label="Filter by status"
          >
            <option value="ALL">All Statuses</option>
            <option value="UPCOMING">Upcoming</option>
            <option value="IN_PROGRESS">In Progress</option>
          </select>
        </div>
        
        <div className="flex-1" />
        
        <span className="text-sm text-gray-500">
          {filteredEvents.length} event{filteredEvents.length !== 1 ? 's' : ''}
        </span>
      </div>
      
      {/* Event List */}
      <div className="flex-1 overflow-auto px-4 pb-4 space-y-3">
        {filteredEvents.length > 0 ? (
          filteredEvents.map(event => (
            <EventCard
              key={event.id}
              event={event}
              isExpanded={expandedEvent === event.id}
              onToggle={() => handleToggleExpand(event.id)}
              onSelect={() => onEventSelect?.(event)}
            />
          ))
        ) : (
          <div className="text-center py-12">
            <Calendar size={48} className="mx-auto text-gray-300 mb-4" />
            <h3 className="font-medium text-gray-600 mb-2">No lifecycle events found</h3>
            <p className="text-sm text-gray-500">Add client life events to trigger automated workflows</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ClientLifecycleEngine;
