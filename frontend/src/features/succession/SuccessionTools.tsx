import React, { useState } from 'react';
import { 
  TrendingUp, Users, DollarSign, Target, 
  Award, AlertTriangle, CheckCircle
} from 'lucide-react';

// Types
interface PracticeMetrics {
  advisor_id: string;
  total_aum: number;
  client_count: number;
  trailing_12mo_revenue: number;
  estimated_valuation: number;
  valuation_multiple: number;
  succession_readiness_score: number;
  key_person_dependency_score: number;
}

interface SuccessionToolsProps {
  advisorId: string;
}

export const SuccessionTools: React.FC<SuccessionToolsProps> = ({ advisorId }) => {
  const [metrics, setMetrics] = useState<PracticeMetrics>({
    advisor_id: advisorId,
    total_aum: 25000000,
    client_count: 50,
    trailing_12mo_revenue: 250000,
    estimated_valuation: 625000,
    valuation_multiple: 2.5,
    succession_readiness_score: 65,
    key_person_dependency_score: 75,
  });

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Practice Succession & Valuation</h1>
          <p className="text-gray-500 mt-1">Continuity Planning & Business Transition Tools</p>
        </div>
        <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
          Generate Succession Plan
        </button>
      </div>

      {/* Valuation Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <ValuationCard 
          title="Practice Valuation" 
          value={`$${(metrics.estimated_valuation / 1000).toFixed(0)}k`}
          subtitle={`${metrics.valuation_multiple}x Revenue`}
          icon={<DollarSign className="text-green-600" />}
        />
        <ValuationCard 
          title="Total AUM" 
          value={`$${(metrics.total_aum / 1000000).toFixed(1)}M`} 
          subtitle={`${metrics.client_count} Clients`}
          icon={<TrendingUp className="text-blue-600" />}
        />
        <ValuationCard 
          title="Annual Revenue" 
          value={`$${(metrics.trailing_12mo_revenue / 1000).toFixed(0)}k`} 
          subtitle="Trailing 12 Months"
          icon={<Target className="text-purple-600" />}
        />
        <ValuationCard 
          title="Readiness Score" 
          value={`${metrics.succession_readiness_score}/100`} 
          subtitle={metrics.succession_readiness_score >= 70 ? "Ready" : "Needs Work"}
          icon={metrics.succession_readiness_score >= 70 ? 
            <CheckCircle className="text-green-600" /> : 
            <AlertTriangle className="text-orange-600" />
          }
        />
      </div>

      {/* Readiness Breakdown */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <h2 className="font-semibold text-lg mb-4">Succession Readiness</h2>
          <div className="space-y-4">
            <ReadinessBar label="Client Documentation" score={85} />
            <ReadinessBar label="CRM Hygiene" score={85} />
            <ReadinessBar label="Service Process Docs" score={70} />
            <ReadinessBar label="Financial Transparency" score={90} />
            <ReadinessBar label="Client Concentration" score={58} />
          </div>
        </div>

        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <h2 className="font-semibold text-lg mb-4">Recommended Actions</h2>
          <div className="space-y-3">
            <ActionItem 
              priority="HIGH" 
              action="Document client service procedures" 
              impact="Critical for smooth transition"
            />
            <ActionItem 
              priority="MEDIUM" 
              action="Reduce client concentration risk" 
              impact="Top 10 clients = 42% of AUM"
            />
            <ActionItem 
              priority="LOW" 
              action="Update investment philosophy document" 
              impact="Enhances continuity"
            />
          </div>
        </div>
      </div>

      {/* Successor Recommendations */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200">
        <div className="p-6 border-b border-gray-200">
          <h2 className="font-semibold text-lg">Recommended Successors</h2>
        </div>
        <div className="divide-y divide-gray-200">
          <SuccessorCard 
            name="Sarah Johnson" 
            compatibilityScore={0.87} 
            strengths={["Client demographic match", "Similar service style", "Capacity available"]}
          />
          <SuccessorCard 
            name="Michael Chen" 
            compatibilityScore={0.82} 
            strengths={["Strong specialization overlap", "High client satisfaction", "Geographic proximity"]}
          />
        </div>
      </div>
    </div>
  );
};

const ValuationCard: React.FC<{ title: string; value: string; subtitle: string; icon: React.ReactNode }> = ({ 
  title, value, subtitle, icon 
}) => (
  <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-200">
    <div className="flex justify-between items-start mb-2">
      <div className="p-2 bg-gray-50 rounded-lg">{icon}</div>
    </div>
    <h3 className="text-gray-500 text-sm font-medium">{title}</h3>
    <div className="text-2xl font-bold text-gray-900 mt-1">{value}</div>
    <div className="text-xs text-gray-400 mt-1">{subtitle}</div>
  </div>
);

const ReadinessBar: React.FC<{ label: string; score: number }> = ({ label, score }) => {
  const getColor = () => {
    if (score >= 80) return 'bg-green-500';
    if (score >= 60) return 'bg-yellow-500';
    return 'bg-red-500';
  };

  return (
    <div>
      <div className="flex justify-between text-sm mb-1">
        <span className="text-gray-700">{label}</span>
        <span className="font-medium text-gray-900">{score}%</span>
      </div>
      <div className="w-full bg-gray-200 h-2 rounded-full overflow-hidden">
        <div className={`${getColor()} h-full`} style={{ width: `${score}%` }}></div>
      </div>
    </div>
  );
};

const ActionItem: React.FC<{ priority: string; action: string; impact: string }> = ({ priority, action, impact }) => {
  const colors = {
    HIGH: 'bg-red-100 text-red-700',
    MEDIUM: 'bg-yellow-100 text-yellow-700',
    LOW: 'bg-blue-100 text-blue-700',
  };

  return (
    <div className="flex gap-3">
      <span className={`px-2 py-1 rounded text-xs font-medium self-start ${colors[priority as keyof typeof colors]}`}>
        {priority}
      </span>
      <div className="flex-1">
        <div className="font-medium text-gray-900 text-sm">{action}</div>
        <div className="text-xs text-gray-500 mt-1">{impact}</div>
      </div>
    </div>
  );
};

const SuccessorCard: React.FC<{ name: string; compatibilityScore: number; strengths: string[] }> = ({ 
  name, compatibilityScore, strengths 
}) => (
  <div className="p-6 hover:bg-gray-50 transition-colors">
    <div className="flex justify-between items-start mb-3">
      <div className="flex items-center gap-3">
        <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center">
          <Users className="text-blue-600" size={24} />
        </div>
        <div>
          <h3 className="font-semibold text-gray-900">{name}</h3>
          <div className="flex items-center gap-2 mt-1">
            <Award size={14} className="text-yellow-500" />
            <span className="text-sm text-gray-600">Compatibility: {(compatibilityScore * 100).toFixed(0)}%</span>
          </div>
        </div>
      </div>
      <button className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm">
        View Details
      </button>
    </div>
    <div className="flex flex-wrap gap-2">
      {strengths.map((strength, idx) => (
        <span key={idx} className="px-2 py-1 bg-green-50 text-green-700 text-xs rounded">
          {strength}
        </span>
      ))}
    </div>
  </div>
);

export default SuccessionTools;
