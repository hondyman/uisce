/**
 * MeetingPrepAutomation.tsx
 * 
 * AI-Powered Meeting Preparation Automation:
 * - Automated meeting packet generation
 * - Client-specific document assembly
 * - Pre-meeting agenda and talking points
 * - Portfolio performance summaries and insights
 */

import React, { useState, useMemo } from 'react';
import {
  FileText,
  Users,
  Calendar,
  Clock,
  Sparkles,
  Download,
  Send,
  CheckCircle,
  AlertCircle,
  RefreshCw,
  Eye,
  Settings,
  Filter,
  Search,
  BarChart3,
  TrendingUp,
  Briefcase
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface Meeting {
  id: string;
  clientId: string;
  clientName: string;
  clientTier: 'PLATINUM' | 'GOLD' | 'SILVER' | 'STANDARD';
  advisorId: string;
  advisorName: string;
  meetingType: MeetingType;
  scheduledDate: Date;
  duration: number;
  status: MeetingStatus;
  prepStatus: PrepStatus;
  packetId?: string;
}

type MeetingType = 'ANNUAL_REVIEW' | 'QUARTERLY_UPDATE' | 'TAX_PLANNING' | 'RETIREMENT_PLANNING' | 'ESTATE_PLANNING' | 'NEW_CLIENT' | 'AD_HOC';

type MeetingStatus = 'SCHEDULED' | 'CONFIRMED' | 'COMPLETED' | 'CANCELLED' | 'RESCHEDULED';

type PrepStatus = 'NOT_STARTED' | 'IN_PROGRESS' | 'READY' | 'DELIVERED';

interface MeetingPacket {
  id: string;
  meetingId: string;
  clientId: string;
  generatedAt: Date;
  status: 'GENERATING' | 'READY' | 'DELIVERED' | 'ERROR';
  sections: PacketSection[];
  aiInsights: AIInsight[];
  deliveryMethod?: 'EMAIL' | 'PORTAL' | 'PRINT';
  deliveredAt?: Date;
}

interface PacketSection {
  id: string;
  type: SectionType;
  title: string;
  status: 'PENDING' | 'GENERATING' | 'READY' | 'ERROR';
  pageCount?: number;
  lastUpdated?: Date;
}

type SectionType = 
  | 'COVER_PAGE'
  | 'AGENDA'
  | 'PORTFOLIO_SUMMARY'
  | 'PERFORMANCE_REPORT'
  | 'HOLDINGS_DETAIL'
  | 'ASSET_ALLOCATION'
  | 'TAX_SUMMARY'
  | 'PLANNING_SCENARIOS'
  | 'MARKET_COMMENTARY'
  | 'RECOMMENDATIONS'
  | 'ACTION_ITEMS';

interface AIInsight {
  id: string;
  category: 'PORTFOLIO' | 'PLANNING' | 'TAX' | 'RISK' | 'OPPORTUNITY';
  title: string;
  description: string;
  priority: 'HIGH' | 'MEDIUM' | 'LOW';
  relatedData?: string;
}

interface PacketTemplate {
  id: string;
  name: string;
  meetingType: MeetingType;
  sections: SectionType[];
  isDefault: boolean;
}

// ============================================================================
// Constants
// ============================================================================

const MEETING_TYPE_CONFIG: Record<MeetingType, { label: string; color: string; bgColor: string }> = {
  ANNUAL_REVIEW: { label: 'Annual Review', color: 'text-blue-700', bgColor: 'bg-blue-100' },
  QUARTERLY_UPDATE: { label: 'Quarterly Update', color: 'text-green-700', bgColor: 'bg-green-100' },
  TAX_PLANNING: { label: 'Tax Planning', color: 'text-orange-700', bgColor: 'bg-orange-100' },
  RETIREMENT_PLANNING: { label: 'Retirement Planning', color: 'text-purple-700', bgColor: 'bg-purple-100' },
  ESTATE_PLANNING: { label: 'Estate Planning', color: 'text-indigo-700', bgColor: 'bg-indigo-100' },
  NEW_CLIENT: { label: 'New Client', color: 'text-cyan-700', bgColor: 'bg-cyan-100' },
  AD_HOC: { label: 'Ad Hoc', color: 'text-gray-700', bgColor: 'bg-gray-100' }
};

const PREP_STATUS_CONFIG: Record<PrepStatus, { label: string; color: string; bgColor: string }> = {
  NOT_STARTED: { label: 'Not Started', color: 'text-gray-700', bgColor: 'bg-gray-100' },
  IN_PROGRESS: { label: 'In Progress', color: 'text-yellow-700', bgColor: 'bg-yellow-100' },
  READY: { label: 'Ready', color: 'text-green-700', bgColor: 'bg-green-100' },
  DELIVERED: { label: 'Delivered', color: 'text-blue-700', bgColor: 'bg-blue-100' }
};

const CLIENT_TIER_COLORS: Record<string, string> = {
  PLATINUM: 'bg-purple-100 text-purple-800',
  GOLD: 'bg-yellow-100 text-yellow-800',
  SILVER: 'bg-gray-200 text-gray-800',
  STANDARD: 'bg-blue-50 text-blue-800'
};

const SECTION_CONFIG: Record<SectionType, { label: string; icon: React.FC<{ className?: string }> }> = {
  COVER_PAGE: { label: 'Cover Page', icon: FileText },
  AGENDA: { label: 'Meeting Agenda', icon: Calendar },
  PORTFOLIO_SUMMARY: { label: 'Portfolio Summary', icon: Briefcase },
  PERFORMANCE_REPORT: { label: 'Performance Report', icon: TrendingUp },
  HOLDINGS_DETAIL: { label: 'Holdings Detail', icon: FileText },
  ASSET_ALLOCATION: { label: 'Asset Allocation', icon: BarChart3 },
  TAX_SUMMARY: { label: 'Tax Summary', icon: FileText },
  PLANNING_SCENARIOS: { label: 'Planning Scenarios', icon: Sparkles },
  MARKET_COMMENTARY: { label: 'Market Commentary', icon: TrendingUp },
  RECOMMENDATIONS: { label: 'Recommendations', icon: CheckCircle },
  ACTION_ITEMS: { label: 'Action Items', icon: CheckCircle }
};

// ============================================================================
// Mock Data
// ============================================================================

const generateMockMeetings = (): Meeting[] => {
  const clients = [
    { id: 'c1', name: 'Robert & Linda Thompson', tier: 'PLATINUM' as const },
    { id: 'c2', name: 'Michael Chang', tier: 'GOLD' as const },
    { id: 'c3', name: 'Jennifer Williams', tier: 'SILVER' as const },
    { id: 'c4', name: 'David & Maria Santos', tier: 'PLATINUM' as const },
    { id: 'c5', name: 'Sarah Johnson', tier: 'GOLD' as const }
  ];
  
  const advisors = [
    { id: 'a1', name: 'Sarah Mitchell' },
    { id: 'a2', name: 'James Chen' },
    { id: 'a3', name: 'Emily Rodriguez' }
  ];

  const meetingTypes: MeetingType[] = ['ANNUAL_REVIEW', 'QUARTERLY_UPDATE', 'TAX_PLANNING', 'RETIREMENT_PLANNING'];
  const prepStatuses: PrepStatus[] = ['NOT_STARTED', 'IN_PROGRESS', 'READY', 'DELIVERED'];

  return Array.from({ length: 12 }, (_, i) => {
    const client = clients[i % clients.length];
    const advisor = advisors[i % advisors.length];
    const scheduledDate = new Date(Date.now() + (i - 3) * 24 * 60 * 60 * 1000);
    
    return {
      id: `mtg-${i + 1}`,
      clientId: client.id,
      clientName: client.name,
      clientTier: client.tier,
      advisorId: advisor.id,
      advisorName: advisor.name,
      meetingType: meetingTypes[i % meetingTypes.length],
      scheduledDate,
      duration: 60,
      status: scheduledDate > new Date() ? 'SCHEDULED' : 'COMPLETED',
      prepStatus: prepStatuses[Math.min(i % 4, 3)],
      packetId: i < 6 ? `pkt-${i + 1}` : undefined
    };
  });
};

const generateMockPacket = (meeting: Meeting): MeetingPacket => {
  const sections: PacketSection[] = [
    { id: 's1', type: 'COVER_PAGE', title: 'Cover Page', status: 'READY', pageCount: 1 },
    { id: 's2', type: 'AGENDA', title: 'Meeting Agenda', status: 'READY', pageCount: 1 },
    { id: 's3', type: 'PORTFOLIO_SUMMARY', title: 'Portfolio Summary', status: 'READY', pageCount: 2 },
    { id: 's4', type: 'PERFORMANCE_REPORT', title: 'Performance Report', status: 'READY', pageCount: 4 },
    { id: 's5', type: 'ASSET_ALLOCATION', title: 'Asset Allocation', status: 'READY', pageCount: 2 },
    { id: 's6', type: 'RECOMMENDATIONS', title: 'Recommendations', status: 'READY', pageCount: 2 }
  ];

  const insights: AIInsight[] = [
    { id: 'i1', category: 'PORTFOLIO', title: 'Portfolio Concentration Risk', description: 'Technology sector allocation (32%) exceeds target (25%). Consider rebalancing.', priority: 'HIGH' },
    { id: 'i2', category: 'TAX', title: 'Tax-Loss Harvesting Opportunity', description: '3 positions have unrealized losses totaling $12,500 available for harvesting.', priority: 'MEDIUM' },
    { id: 'i3', category: 'OPPORTUNITY', title: 'Roth Conversion Window', description: 'Lower income year creates favorable Roth conversion opportunity.', priority: 'MEDIUM' }
  ];

  return {
    id: meeting.packetId || `pkt-${meeting.id}`,
    meetingId: meeting.id,
    clientId: meeting.clientId,
    generatedAt: new Date(),
    status: meeting.prepStatus === 'DELIVERED' ? 'DELIVERED' : 'READY',
    sections,
    aiInsights: insights,
    deliveryMethod: meeting.prepStatus === 'DELIVERED' ? 'EMAIL' : undefined,
    deliveredAt: meeting.prepStatus === 'DELIVERED' ? new Date(Date.now() - 2 * 60 * 60 * 1000) : undefined
  };
};

const DEFAULT_TEMPLATES: PacketTemplate[] = [
  {
    id: 't1',
    name: 'Annual Review Package',
    meetingType: 'ANNUAL_REVIEW',
    sections: ['COVER_PAGE', 'AGENDA', 'PORTFOLIO_SUMMARY', 'PERFORMANCE_REPORT', 'ASSET_ALLOCATION', 'TAX_SUMMARY', 'RECOMMENDATIONS', 'ACTION_ITEMS'],
    isDefault: true
  },
  {
    id: 't2',
    name: 'Quarterly Update',
    meetingType: 'QUARTERLY_UPDATE',
    sections: ['COVER_PAGE', 'AGENDA', 'PORTFOLIO_SUMMARY', 'PERFORMANCE_REPORT', 'MARKET_COMMENTARY'],
    isDefault: true
  },
  {
    id: 't3',
    name: 'Tax Planning Session',
    meetingType: 'TAX_PLANNING',
    sections: ['COVER_PAGE', 'AGENDA', 'TAX_SUMMARY', 'PLANNING_SCENARIOS', 'RECOMMENDATIONS'],
    isDefault: true
  }
];

// ============================================================================
// Main Component
// ============================================================================

interface MeetingPrepAutomationProps {
  tenantId?: string;
  datasourceId?: string;
}

export const MeetingPrepAutomation: React.FC<MeetingPrepAutomationProps> = ({
  tenantId: _tenantId,
  datasourceId: _datasourceId
}) => {
  // State
  const [meetings] = useState<Meeting[]>(generateMockMeetings);
  const [selectedMeeting, setSelectedMeeting] = useState<Meeting | null>(null);
  const [activePacket, setActivePacket] = useState<MeetingPacket | null>(null);
  const [activeTab, setActiveTab] = useState<'upcoming' | 'packets' | 'templates' | 'insights'>('upcoming');
  const [filterPrepStatus, setFilterPrepStatus] = useState<string>('ALL');
  const [searchQuery, setSearchQuery] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);

  // Derived state
  const upcomingMeetings = useMemo(() => {
    return meetings.filter(m => {
      const isUpcoming = m.scheduledDate >= new Date();
      if (!isUpcoming) return false;
      if (filterPrepStatus !== 'ALL' && m.prepStatus !== filterPrepStatus) return false;
      if (searchQuery && !m.clientName.toLowerCase().includes(searchQuery.toLowerCase())) return false;
      return true;
    }).sort((a, b) => a.scheduledDate.getTime() - b.scheduledDate.getTime());
  }, [meetings, filterPrepStatus, searchQuery]);

  const metrics = useMemo(() => ({
    totalUpcoming: meetings.filter(m => m.scheduledDate >= new Date()).length,
    needsPrep: meetings.filter(m => m.scheduledDate >= new Date() && m.prepStatus === 'NOT_STARTED').length,
    inProgress: meetings.filter(m => m.prepStatus === 'IN_PROGRESS').length,
    ready: meetings.filter(m => m.prepStatus === 'READY').length,
    delivered: meetings.filter(m => m.prepStatus === 'DELIVERED').length
  }), [meetings]);

  // Generate packet
  const handleGeneratePacket = async (meeting: Meeting) => {
    setIsGenerating(true);
    setSelectedMeeting(meeting);
    
    // Simulate generation
    await new Promise(resolve => setTimeout(resolve, 2000));
    
    const packet = generateMockPacket(meeting);
    setActivePacket(packet);
    setIsGenerating(false);
  };

  // View packet
  const handleViewPacket = (meeting: Meeting) => {
    setSelectedMeeting(meeting);
    setActivePacket(generateMockPacket(meeting));
  };

  // Render upcoming meetings tab
  const renderUpcoming = () => (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search by client name..."
            className="w-full pl-10 pr-4 py-2 border rounded-lg text-sm"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-gray-500" />
          <select
            value={filterPrepStatus}
            onChange={(e) => setFilterPrepStatus(e.target.value)}
            className="border rounded-lg px-3 py-2 text-sm"
            title="Filter by prep status"
          >
            <option value="ALL">All Status</option>
            {Object.entries(PREP_STATUS_CONFIG).map(([key, config]) => (
              <option key={key} value={key}>{config.label}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Meeting list */}
      <div className="space-y-3">
        {upcomingMeetings.map(meeting => {
          const typeConfig = MEETING_TYPE_CONFIG[meeting.meetingType];
          const prepConfig = PREP_STATUS_CONFIG[meeting.prepStatus];
          const daysUntil = Math.ceil((meeting.scheduledDate.getTime() - Date.now()) / (24 * 60 * 60 * 1000));
          
          return (
            <div
              key={meeting.id}
              className={`bg-white rounded-lg border p-4 ${
                daysUntil <= 1 && meeting.prepStatus === 'NOT_STARTED' ? 'border-red-200 bg-red-50' : ''
              }`}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <h3 className="font-medium">{meeting.clientName}</h3>
                    <span className={`px-2 py-0.5 rounded text-xs ${CLIENT_TIER_COLORS[meeting.clientTier]}`}>
                      {meeting.clientTier}
                    </span>
                    <span className={`px-2 py-0.5 rounded text-xs ${typeConfig.bgColor} ${typeConfig.color}`}>
                      {typeConfig.label}
                    </span>
                  </div>
                  <div className="flex items-center gap-4 mt-2 text-sm text-gray-600">
                    <span className="flex items-center gap-1">
                      <Calendar className="w-4 h-4" />
                      {meeting.scheduledDate.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}
                    </span>
                    <span className="flex items-center gap-1">
                      <Clock className="w-4 h-4" />
                      {meeting.scheduledDate.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })}
                    </span>
                    <span className="flex items-center gap-1">
                      <Users className="w-4 h-4" />
                      {meeting.advisorName}
                    </span>
                    <span className={`flex items-center gap-1 ${daysUntil <= 1 ? 'text-red-600 font-medium' : ''}`}>
                      {daysUntil === 0 ? 'Today' : daysUntil === 1 ? 'Tomorrow' : `${daysUntil} days`}
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className={`px-2 py-1 rounded text-xs ${prepConfig.bgColor} ${prepConfig.color}`}>
                    {prepConfig.label}
                  </span>
                  {meeting.prepStatus === 'NOT_STARTED' && (
                    <button
                      onClick={() => handleGeneratePacket(meeting)}
                      disabled={isGenerating}
                      className="flex items-center gap-2 px-3 py-1.5 bg-purple-600 text-white rounded-lg text-sm hover:bg-purple-700 disabled:opacity-50"
                    >
                      <Sparkles className="w-4 h-4" />
                      Generate Packet
                    </button>
                  )}
                  {(meeting.prepStatus === 'READY' || meeting.prepStatus === 'DELIVERED') && (
                    <button
                      onClick={() => handleViewPacket(meeting)}
                      className="flex items-center gap-2 px-3 py-1.5 border rounded-lg text-sm hover:bg-gray-50"
                    >
                      <Eye className="w-4 h-4" />
                      View
                    </button>
                  )}
                  {meeting.prepStatus === 'IN_PROGRESS' && (
                    <div className="flex items-center gap-2 text-yellow-600">
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      <span className="text-sm">Generating...</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );

  // Render packet viewer
  const renderPacketViewer = () => {
    if (!selectedMeeting || !activePacket) {
      return (
        <div className="text-center py-12 text-gray-500">
          <FileText className="w-12 h-12 mx-auto mb-4 text-gray-300" />
          <p>Select a meeting to view its packet</p>
        </div>
      );
    }

    return (
      <div className="space-y-6">
        {/* Packet header */}
        <div className="bg-white rounded-lg border p-6">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold">{selectedMeeting.clientName} - Meeting Packet</h2>
              <p className="text-sm text-gray-500 mt-1">
                {MEETING_TYPE_CONFIG[selectedMeeting.meetingType].label} • {selectedMeeting.scheduledDate.toLocaleDateString()}
              </p>
            </div>
            <div className="flex items-center gap-3">
              <button className="flex items-center gap-2 px-3 py-1.5 border rounded-lg text-sm hover:bg-gray-50">
                <Download className="w-4 h-4" />
                Download PDF
              </button>
              <button className="flex items-center gap-2 px-3 py-1.5 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700">
                <Send className="w-4 h-4" />
                Send to Client
              </button>
            </div>
          </div>

          {/* Sections */}
          <div className="mt-6">
            <h3 className="text-sm font-medium text-gray-700 mb-3">Packet Sections</h3>
            <div className="grid grid-cols-2 gap-3">
              {activePacket.sections.map(section => {
                const sectionConfig = SECTION_CONFIG[section.type];
                return (
                  <div key={section.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                    <div className="flex items-center gap-3">
                      <sectionConfig.icon className="w-5 h-5 text-gray-400" />
                      <span className="text-sm font-medium">{section.title}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      {section.pageCount && (
                        <span className="text-xs text-gray-500">{section.pageCount} pg</span>
                      )}
                      {section.status === 'READY' ? (
                        <CheckCircle className="w-4 h-4 text-green-500" />
                      ) : section.status === 'GENERATING' ? (
                        <RefreshCw className="w-4 h-4 text-blue-500 animate-spin" />
                      ) : section.status === 'ERROR' ? (
                        <AlertCircle className="w-4 h-4 text-red-500" />
                      ) : (
                        <Clock className="w-4 h-4 text-gray-400" />
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* AI Insights */}
        <div className="bg-white rounded-lg border p-6">
          <h3 className="text-sm font-medium text-gray-700 mb-3 flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-purple-500" />
            AI-Generated Insights
          </h3>
          <div className="space-y-3">
            {activePacket.aiInsights.map(insight => (
              <div 
                key={insight.id} 
                className={`p-4 rounded-lg border ${
                  insight.priority === 'HIGH' ? 'bg-red-50 border-red-200' :
                  insight.priority === 'MEDIUM' ? 'bg-yellow-50 border-yellow-200' :
                  'bg-blue-50 border-blue-200'
                }`}
              >
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <span className={`px-2 py-0.5 rounded text-xs ${
                        insight.category === 'PORTFOLIO' ? 'bg-blue-100 text-blue-800' :
                        insight.category === 'TAX' ? 'bg-orange-100 text-orange-800' :
                        insight.category === 'RISK' ? 'bg-red-100 text-red-800' :
                        'bg-green-100 text-green-800'
                      }`}>
                        {insight.category}
                      </span>
                      <h4 className="font-medium text-sm">{insight.title}</h4>
                    </div>
                    <p className="text-sm text-gray-600 mt-1">{insight.description}</p>
                  </div>
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    insight.priority === 'HIGH' ? 'bg-red-200 text-red-800' :
                    insight.priority === 'MEDIUM' ? 'bg-yellow-200 text-yellow-800' :
                    'bg-gray-200 text-gray-800'
                  }`}>
                    {insight.priority}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  };

  // Render templates tab
  const renderTemplates = () => (
    <div className="space-y-4">
      {DEFAULT_TEMPLATES.map(template => {
        const typeConfig = MEETING_TYPE_CONFIG[template.meetingType];
        return (
          <div key={template.id} className="bg-white rounded-lg border p-4">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-3">
                <h3 className="font-medium">{template.name}</h3>
                <span className={`px-2 py-0.5 rounded text-xs ${typeConfig.bgColor} ${typeConfig.color}`}>
                  {typeConfig.label}
                </span>
                {template.isDefault && (
                  <span className="px-2 py-0.5 bg-green-100 text-green-800 rounded text-xs">Default</span>
                )}
              </div>
              <button className="text-sm text-blue-600 hover:text-blue-700">Edit Template</button>
            </div>
            <div className="flex flex-wrap gap-2">
              {template.sections.map(section => {
                const sectionConfig = SECTION_CONFIG[section];
                return (
                  <span key={section} className="flex items-center gap-1 px-2 py-1 bg-gray-100 rounded text-xs">
                    <sectionConfig.icon className="w-3 h-3" />
                    {sectionConfig.label}
                  </span>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold flex items-center gap-2">
              <Sparkles className="w-6 h-6 text-purple-600" />
              Meeting Prep Automation
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              AI-powered meeting packet generation and insights
            </p>
          </div>
          <button className="flex items-center gap-2 px-3 py-1.5 border rounded-lg hover:bg-gray-50">
            <Settings className="w-4 h-4" />
            Settings
          </button>
        </div>

        {/* Stats bar */}
        <div className="grid grid-cols-5 gap-4 mt-4">
          <div className="bg-gray-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-gray-500">Upcoming Meetings</span>
              <Calendar className="w-4 h-4 text-gray-400" />
            </div>
            <div className="text-xl font-bold">{metrics.totalUpcoming}</div>
          </div>
          <div className="bg-red-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-red-600">Needs Prep</span>
              <AlertCircle className="w-4 h-4 text-red-400" />
            </div>
            <div className="text-xl font-bold text-red-700">{metrics.needsPrep}</div>
          </div>
          <div className="bg-yellow-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-yellow-600">In Progress</span>
              <RefreshCw className="w-4 h-4 text-yellow-400" />
            </div>
            <div className="text-xl font-bold text-yellow-700">{metrics.inProgress}</div>
          </div>
          <div className="bg-green-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-green-600">Ready</span>
              <CheckCircle className="w-4 h-4 text-green-400" />
            </div>
            <div className="text-xl font-bold text-green-700">{metrics.ready}</div>
          </div>
          <div className="bg-blue-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-blue-600">Delivered</span>
              <Send className="w-4 h-4 text-blue-400" />
            </div>
            <div className="text-xl font-bold text-blue-700">{metrics.delivered}</div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <div className="flex gap-6">
          {[
            { id: 'upcoming' as const, label: 'Upcoming Meetings', icon: Calendar },
            { id: 'packets' as const, label: 'Packet Viewer', icon: FileText },
            { id: 'templates' as const, label: 'Templates', icon: Settings }
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
        {activeTab === 'upcoming' && renderUpcoming()}
        {activeTab === 'packets' && renderPacketViewer()}
        {activeTab === 'templates' && renderTemplates()}
      </div>
    </div>
  );
};

export default MeetingPrepAutomation;
