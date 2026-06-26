import React, { useState, useEffect } from 'react';
import { 
  Phone, Mail, Calendar, ArrowRight, CheckCircle, XCircle, 
  AlertTriangle, TrendingUp, DollarSign, Clock, Filter, 
  ChevronDown, MoreHorizontal, Play, BarChart2
} from 'lucide-react';
import { fetchAPI } from '../../../api';

// Types
interface ActionTemplate {
  email_subject?: string;
  email_body?: string;
  call_script?: string;
  meeting_agenda?: string;
  [key: string]: any;
}

interface NextBestAction {
  action_id: string;
  client_id: string;
  client_name: string;
  action_type: string;
  action_name: string;
  confidence: number;
  urgency_score: number;
  expected_value: number;
  success_probability: number;
  trigger_signal: string;
  reasoning: string;
  recommended_channel: string;
  estimated_duration_minutes: number;
  template_content: ActionTemplate;
}

interface NBADashboardProps {
  advisorId?: string;
}

export const AdvisorNBADashboard: React.FC<NBADashboardProps> = ({ advisorId }) => {
  const [actions, setActions] = useState<NextBestAction[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedAction, setSelectedAction] = useState<NextBestAction | null>(null);
  const [filter, setFilter] = useState('ALL'); // ALL, HIGH_URGENCY, HIGH_VALUE

  useEffect(() => {
    loadRecommendations();
  }, [advisorId]);

  const loadRecommendations = async () => {
    setLoading(true);
    try {
      const data = await fetchAPI<NextBestAction[]>('/nba/recommendations');
      setActions(data);
    } catch (error) {
      console.error('Failed to load NBA recommendations:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleExecuteAction = (action: NextBestAction) => {
    setSelectedAction(action);
  };

  const handleDismissAction = async (actionId: string) => {
    try {
      await fetchAPI('/nba/dismiss', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action_id: actionId })
      });
      setActions(actions.filter(a => a.action_id !== actionId));
    } catch (error) {
      console.error('Failed to dismiss action:', error);
    }
  };

  const getFilteredActions = () => {
    if (filter === 'HIGH_URGENCY') return actions.filter(a => a.urgency_score > 0.8);
    if (filter === 'HIGH_VALUE') return actions.filter(a => a.expected_value > 1000);
    return actions;
  };

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Next Best Actions</h1>
          <p className="text-gray-500 mt-1">AI-driven recommendations for your book of business</p>
        </div>
        <div className="flex gap-3">
          <FilterButton 
            label="All Actions" 
            active={filter === 'ALL'} 
            onClick={() => setFilter('ALL')} 
          />
          <FilterButton 
            label="High Urgency" 
            active={filter === 'HIGH_URGENCY'} 
            onClick={() => setFilter('HIGH_URGENCY')} 
            icon={<AlertTriangle size={16} />}
          />
          <FilterButton 
            label="High Value" 
            active={filter === 'HIGH_VALUE'} 
            onClick={() => setFilter('HIGH_VALUE')} 
            icon={<DollarSign size={16} />}
          />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <StatCard 
          title="Potential Revenue" 
          value={`$${actions.reduce((sum, a) => sum + a.expected_value, 0).toLocaleString()}`} 
          icon={<TrendingUp className="text-green-600" />}
          trend="+12% vs last week"
        />
        <StatCard 
          title="Pending Actions" 
          value={actions.length.toString()} 
          icon={<Clock className="text-blue-600" />}
          trend="5 urgent"
        />
        <StatCard 
          title="Success Rate" 
          value="68%" 
          icon={<CheckCircle className="text-purple-600" />}
          trend="+2.4%"
        />
      </div>

      <div className="space-y-4">
        {loading ? (
          <div className="text-center py-12 text-gray-500">Loading recommendations...</div>
        ) : getFilteredActions().map(action => (
          <ActionCard 
            key={action.action_id} 
            action={action} 
            onExecute={() => handleExecuteAction(action)}
            onDismiss={() => handleDismissAction(action.action_id)}
          />
        ))}
      </div>

      {selectedAction && (
        <ActionExecutionModal 
          action={selectedAction} 
          onClose={() => setSelectedAction(null)}
          onComplete={() => {
            setActions(actions.filter(a => a.action_id !== selectedAction.action_id));
            setSelectedAction(null);
          }}
        />
      )}
    </div>
  );
};

const ActionCard: React.FC<{ 
  action: NextBestAction; 
  onExecute: () => void; 
  onDismiss: () => void; 
}> = ({ action, onExecute, onDismiss }) => {
  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow">
      <div className="flex justify-between items-start">
        <div className="flex gap-4">
          <div className={`p-3 rounded-lg ${
            action.urgency_score > 0.8 ? 'bg-red-50 text-red-600' : 'bg-blue-50 text-blue-600'
          }`}>
            {action.recommended_channel === 'PHONE' ? <Phone size={24} /> : <Mail size={24} />}
          </div>
          <div>
            <div className="flex items-center gap-2 mb-1">
              <h3 className="font-semibold text-lg text-gray-900">{action.action_name}</h3>
              <UrgencyBadge score={action.urgency_score} />
            </div>
            <p className="text-gray-600 mb-2">Client: <span className="font-medium text-gray-900">{action.client_name}</span></p>
            <p className="text-sm text-gray-500 bg-gray-50 p-2 rounded inline-block">
              Trigger: {action.trigger_signal.replace(/_/g, ' ')}
            </p>
          </div>
        </div>
        
        <div className="text-right">
          <div className="text-2xl font-bold text-gray-900">${action.expected_value.toLocaleString()}</div>
          <div className="text-sm text-gray-500">Est. Value</div>
          <div className="mt-1 text-xs font-medium text-green-600 bg-green-50 px-2 py-1 rounded-full inline-block">
            {Math.round(action.success_probability * 100)}% Success Prob.
          </div>
        </div>
      </div>

      <div className="mt-4 pt-4 border-t border-gray-100 flex justify-between items-center">
        <div className="text-sm text-gray-600">
          <span className="font-medium">Why:</span> {action.reasoning}
        </div>
        <div className="flex gap-3">
          <button 
            onClick={onDismiss}
            className="px-4 py-2 text-gray-600 hover:bg-gray-50 rounded-lg text-sm font-medium transition-colors"
          >
            Dismiss
          </button>
          <button 
            onClick={onExecute}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors flex items-center gap-2"
          >
            <Play size={16} /> Execute Action
          </button>
        </div>
      </div>
    </div>
  );
};

const ActionExecutionModal: React.FC<{
  action: NextBestAction;
  onClose: () => void;
  onComplete: () => void;
}> = ({ action, onClose, onComplete }) => {
  const [step, setStep] = useState(1);
  const [notes, setNotes] = useState('');

  const handleComplete = async () => {
    try {
      await fetchAPI('/nba/complete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action_id: action.action_id,
          outcome: 'COMPLETED',
          notes: notes,
          executed_at: new Date().toISOString()
        })
      });
      onComplete();
    } catch (error) {
      console.error('Failed to complete action:', error);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-xl shadow-xl max-w-2xl w-full mx-4 overflow-hidden">
        <div className="p-6 border-b border-gray-200 flex justify-between items-center">
          <h2 className="text-xl font-bold text-gray-900">Execute Action</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600" aria-label="Close" title="Close">
            <XCircle size={24} />
          </button>
        </div>

        <div className="p-6">
          <div className="mb-6">
            <div className="flex items-center gap-4 mb-4">
              <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
                step >= 1 ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-500'
              }`}>1</div>
              <div className="h-1 flex-1 bg-gray-200">
                <div className={`h-full bg-blue-600 transition-all ${step >= 2 ? 'w-full' : 'w-0'}`} />
              </div>
              <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
                step >= 2 ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-500'
              }`}>2</div>
            </div>
          </div>

          {step === 1 ? (
            <div className="space-y-4">
              <h3 className="font-semibold text-lg">Review & Prepare</h3>
              <div className="bg-gray-50 p-4 rounded-lg border border-gray-200">
                <h4 className="text-sm font-medium text-gray-500 uppercase mb-2">Suggested Script / Content</h4>
                <p className="text-gray-800 whitespace-pre-wrap">
                  {action.template_content.call_script || action.template_content.email_body || "No template content available."}
                </p>
              </div>
              <div className="flex justify-end mt-6">
                <button 
                  onClick={() => setStep(2)}
                  className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium"
                >
                  Proceed to Execution
                </button>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <h3 className="font-semibold text-lg">Log Outcome</h3>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Execution Notes</label>
                <textarea 
                  className="w-full border border-gray-300 rounded-lg p-3 h-32 focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Describe the client's reaction and any follow-up items..."
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                />
              </div>
              <div className="flex justify-end gap-3 mt-6">
                <button 
                  onClick={() => setStep(1)}
                  className="px-4 py-2 text-gray-600 hover:bg-gray-50 rounded-lg font-medium"
                >
                  Back
                </button>
                <button 
                  onClick={handleComplete}
                  className="px-6 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 font-medium"
                >
                  Complete Action
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// Helper Components
const FilterButton: React.FC<{ 
  label: string; 
  active: boolean; 
  onClick: () => void; 
  icon?: React.ReactNode; 
}> = ({ label, active, onClick, icon }) => (
  <button 
    onClick={onClick}
    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center gap-2 ${
      active 
        ? 'bg-blue-600 text-white' 
        : 'bg-white text-gray-600 border border-gray-200 hover:bg-gray-50'
    }`}
  >
    {icon}
    {label}
  </button>
);

const StatCard: React.FC<{ 
  title: string; 
  value: string; 
  icon: React.ReactNode; 
  trend: string; 
}> = ({ title, value, icon, trend }) => (
  <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-200">
    <div className="flex justify-between items-start mb-4">
      <div className="p-2 bg-gray-50 rounded-lg">{icon}</div>
      <span className="text-xs font-medium text-green-600 bg-green-50 px-2 py-1 rounded-full">{trend}</span>
    </div>
    <h3 className="text-gray-500 text-sm font-medium">{title}</h3>
    <div className="text-2xl font-bold text-gray-900 mt-1">{value}</div>
  </div>
);

const UrgencyBadge: React.FC<{ score: number }> = ({ score }) => {
  if (score > 0.8) {
    return <span className="text-xs font-bold text-red-600 bg-red-100 px-2 py-0.5 rounded uppercase">High Urgency</span>;
  }
  if (score > 0.5) {
    return <span className="text-xs font-bold text-yellow-600 bg-yellow-100 px-2 py-0.5 rounded uppercase">Medium Urgency</span>;
  }
  return <span className="text-xs font-bold text-blue-600 bg-blue-100 px-2 py-0.5 rounded uppercase">Low Urgency</span>;
};

export default AdvisorNBADashboard;
