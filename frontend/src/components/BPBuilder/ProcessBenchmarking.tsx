/**
 * Process Performance Benchmarking - Industry Standards & Peer Comparison
 * 
 * Features:
 * - Industry benchmark comparison (Fortune 500 standards)
 * - Performance scoring (0-100) with detailed breakdown
 * - Peer group analysis and ranking
 * - Best practice recommendations based on industry leaders
 * - Competitive positioning analysis
 * - Gap analysis with actionable insights
 */

import React, { useState, useEffect } from 'react';
import {
  TrendingUp,
  Target,
  Star,
  AlertCircle,
  CheckCircle2,
  ArrowUp,
  ArrowDown,
  Minus,
  Trophy,
  Zap,
  BookOpen,
  Globe,
  Building2,
  Lightbulb,
  Shield,
} from 'lucide-react';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface BenchmarkScore {
  overall_score: number;
  dimension_scores: {
    efficiency: number;
    quality: number;
    speed: number;
    automation: number;
    compliance: number;
  };
  percentile: number;
  grade: 'A+' | 'A' | 'B+' | 'B' | 'C+' | 'C' | 'D' | 'F';
}

interface IndustryBenchmark {
  industry: string;
  process_type: string;
  metrics: {
    avg_duration_minutes: number;
    success_rate: number;
    automation_rate: number;
    error_rate: number;
    cycle_time_minutes: number;
  };
  top_quartile: {
    avg_duration_minutes: number;
    success_rate: number;
    automation_rate: number;
  };
  median: {
    avg_duration_minutes: number;
    success_rate: number;
    automation_rate: number;
  };
  sample_size: number;
}

interface PeerComparison {
  peer_group: string;
  your_rank: number;
  total_peers: number;
  percentile: number;
  comparison_metrics: {
    metric_name: string;
    your_value: number;
    peer_avg: number;
    peer_best: number;
    variance: number;
  }[];
}

interface BestPractice {
  id: string;
  category: string;
  title: string;
  description: string;
  impact: 'high' | 'medium' | 'low';
  effort: 'low' | 'medium' | 'high';
  industry_adoption: number;
  expected_improvement: number;
  implementation_steps: string[];
  case_studies: CaseStudy[];
}

interface CaseStudy {
  company: string;
  industry: string;
  improvement: string;
  timeframe: string;
}

interface GapAnalysis {
  dimension: string;
  current_score: number;
  target_score: number;
  gap: number;
  priority: 'critical' | 'high' | 'medium' | 'low';
  recommendations: string[];
}

interface ProcessBenchmarkingProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
  processType?: string;
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const ProcessBenchmarking: React.FC<ProcessBenchmarkingProps> = ({
  tenant,
  datasource,
  processType,
}) => {
  const [benchmarkScore, setBenchmarkScore] = useState<BenchmarkScore | null>(null);
  const [industryBenchmark, setIndustryBenchmark] = useState<IndustryBenchmark | null>(null);
  const [peerComparison, setPeerComparison] = useState<PeerComparison | null>(null);
  const [bestPractices, setBestPractices] = useState<BestPractice[]>([]);
  const [gapAnalysis, setGapAnalysis] = useState<GapAnalysis[]>([]);
  const [selectedIndustry, setSelectedIndustry] = useState<string>('financial_services');
  const [selectedProcess, setSelectedProcess] = useState<string>(processType || 'investment_approval');
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState<'overview' | 'peer' | 'practices' | 'gaps'>('overview');

  // Available industries for benchmarking
  const industries = [
    { value: 'financial_services', label: 'Financial Services' },
    { value: 'wealth_management', label: 'Wealth Management' },
    { value: 'asset_management', label: 'Asset Management' },
    { value: 'banking', label: 'Banking' },
    { value: 'insurance', label: 'Insurance' },
  ];

  // Available process types
  const processTypes = [
    { value: 'investment_approval', label: 'Investment Approval' },
    { value: 'client_onboarding', label: 'Client Onboarding' },
    { value: 'portfolio_rebalancing', label: 'Portfolio Rebalancing' },
    { value: 'compliance_review', label: 'Compliance Review' },
    { value: 'risk_assessment', label: 'Risk Assessment' },
  ];

  // Fetch benchmark data
  const fetchBenchmarkData = async () => {
    try {
      setLoading(true);
      const [scoreRes, industryRes, peerRes, practicesRes, gapRes] = await Promise.all([
        fetch(`/api/process-benchmarking/score?tenant_id=${tenant.id}&process_type=${selectedProcess}`),
        fetch(`/api/process-benchmarking/industry?industry=${selectedIndustry}&process_type=${selectedProcess}`),
        fetch(`/api/process-benchmarking/peers?tenant_id=${tenant.id}&industry=${selectedIndustry}`),
        fetch(`/api/process-benchmarking/best-practices?industry=${selectedIndustry}&process_type=${selectedProcess}`),
        fetch(`/api/process-benchmarking/gap-analysis?tenant_id=${tenant.id}&process_type=${selectedProcess}`),
      ]);

      const [score, industry, peer, practices, gaps] = await Promise.all([
        scoreRes.json(),
        industryRes.json(),
        peerRes.json(),
        practicesRes.json(),
        gapRes.json(),
      ]);

      setBenchmarkScore(score);
      setIndustryBenchmark(industry);
      setPeerComparison(peer);
      setBestPractices(practices || []);
      setGapAnalysis(gaps || []);
    } catch (error) {
      console.error('Failed to fetch benchmark data:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBenchmarkData();
  }, [tenant.id, selectedIndustry, selectedProcess]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="flex flex-col items-center gap-4">
          <Trophy className="w-12 h-12 animate-pulse text-yellow-500" />
          <p className="text-gray-600">Loading benchmark data...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 p-6">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 flex items-center gap-3">
              <Trophy className="w-8 h-8 text-yellow-600" />
              Process Performance Benchmarking
            </h1>
            <p className="text-gray-600 mt-2">
              Industry standards, peer comparison, and best practice recommendations
            </p>
          </div>
        </div>

        {/* Filters */}
        <div className="flex gap-4 mt-6">
          <select
            value={selectedIndustry}
            onChange={(e) => setSelectedIndustry(e.target.value)}
            className="px-4 py-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500 bg-white"            aria-label="Select industry for benchmarking"          >
            {industries.map((ind) => (
              <option key={ind.value} value={ind.value}>
                {ind.label}
              </option>
            ))}
          </select>
          <select
            value={selectedProcess}
            onChange={(e) => setSelectedProcess(e.target.value)}
            className="px-4 py-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500 bg-white"            aria-label="Select process type for benchmarking"          >
            {processTypes.map((proc) => (
              <option key={proc.value} value={proc.value}>
                {proc.label}
              </option>
            ))}
          </select>
        </div>

        {/* View Mode Tabs */}
        <div className="flex gap-2 mt-6">
          {(['overview', 'peer', 'practices', 'gaps'] as const).map((mode) => (
            <button
              key={mode}
              onClick={() => setViewMode(mode)}
              className={`px-6 py-3 rounded-lg font-medium transition-all ${
                viewMode === mode
                  ? 'bg-white text-blue-600 shadow-lg border-2 border-blue-200'
                  : 'bg-white/50 text-gray-600 hover:bg-white hover:shadow'
              }`}
            >
              {mode === 'overview' && 'Overview'}
              {mode === 'peer' && 'Peer Comparison'}
              {mode === 'practices' && 'Best Practices'}
              {mode === 'gaps' && 'Gap Analysis'}
            </button>
          ))}
        </div>
      </div>

      {/* Overview View */}
      {viewMode === 'overview' && benchmarkScore && (
        <>
          <OverallScoreCard score={benchmarkScore} />
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
            <DimensionScoresCard scores={benchmarkScore.dimension_scores} />
            {industryBenchmark && <IndustryComparisonCard benchmark={industryBenchmark} />}
          </div>
        </>
      )}

      {/* Peer Comparison View */}
      {viewMode === 'peer' && peerComparison && (
        <PeerComparisonSection comparison={peerComparison} />
      )}

      {/* Best Practices View */}
      {viewMode === 'practices' && (
        <BestPracticesSection practices={bestPractices} />
      )}

      {/* Gap Analysis View */}
      {viewMode === 'gaps' && (
        <GapAnalysisSection gaps={gapAnalysis} />
      )}
    </div>
  );
};

// ============================================================================
// OVERALL SCORE CARD
// ============================================================================

const OverallScoreCard: React.FC<{ score: BenchmarkScore }> = ({ score }) => {
  const getGradeColor = (grade: string) => {
    if (grade.startsWith('A')) return 'from-green-500 to-emerald-600';
    if (grade.startsWith('B')) return 'from-blue-500 to-blue-600';
    if (grade.startsWith('C')) return 'from-yellow-500 to-yellow-600';
    return 'from-red-500 to-red-600';
  };

  const getGradeIcon = (grade: string) => {
    if (grade.startsWith('A')) return <Trophy className="w-16 h-16" />;
    if (grade.startsWith('B')) return <Star className="w-16 h-16" />;
    if (grade.startsWith('C')) return <Target className="w-16 h-16" />;
    return <AlertCircle className="w-16 h-16" />;
  };

  return (
    <div className="bg-white rounded-2xl shadow-xl p-8">
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Overall Performance Score</h2>
          <p className="text-gray-600 mb-6">
            Based on efficiency, quality, speed, automation, and compliance metrics
          </p>
          <div className="flex items-center gap-8">
            <div>
              <div className="text-6xl font-bold text-gray-900">{score.overall_score}</div>
              <div className="text-sm text-gray-500 mt-1">out of 100</div>
            </div>
            <div className={`px-8 py-4 rounded-2xl bg-gradient-to-br ${getGradeColor(score.grade)} text-white`}>
              <div className="text-4xl font-bold text-center">{score.grade}</div>
              <div className="text-sm text-center mt-1 opacity-90">Grade</div>
            </div>
            <div className="flex flex-col items-center">
              <div className="text-3xl font-bold text-blue-600">{score.percentile}th</div>
              <div className="text-sm text-gray-600">Percentile</div>
            </div>
          </div>
        </div>
        <div className={`p-6 rounded-2xl bg-gradient-to-br ${getGradeColor(score.grade)} text-white`}>
          {getGradeIcon(score.grade)}
        </div>
      </div>

      {/* Score Gauge */}
      <div className="mt-8">
        <div className="flex justify-between text-xs text-gray-600 mb-2">
          <span>Poor</span>
          <span>Below Average</span>
          <span>Average</span>
          <span>Good</span>
          <span>Excellent</span>
        </div>
        <div className="h-4 bg-gradient-to-r from-red-500 via-yellow-500 via-blue-500 to-green-500 rounded-full relative">
          <div
            className="absolute top-1/2 transform -translate-y-1/2 -translate-x-1/2 w-6 h-6 bg-white border-4 border-gray-900 rounded-full shadow-lg"
            style={{ left: `${score.overall_score}%` }}
          />
        </div>
        <div className="flex justify-between text-xs text-gray-600 mt-1">
          <span>0</span>
          <span>25</span>
          <span>50</span>
          <span>75</span>
          <span>100</span>
        </div>
      </div>
    </div>
  );
};

// ============================================================================
// DIMENSION SCORES CARD
// ============================================================================

const DimensionScoresCard: React.FC<{ scores: BenchmarkScore['dimension_scores'] }> = ({ scores }) => {
  const dimensions = [
    { key: 'efficiency', label: 'Efficiency', icon: <Zap className="w-5 h-5" />, color: 'blue' },
    { key: 'quality', label: 'Quality', icon: <Shield className="w-5 h-5" />, color: 'green' },
    { key: 'speed', label: 'Speed', icon: <TrendingUp className="w-5 h-5" />, color: 'purple' },
    { key: 'automation', label: 'Automation', icon: <Target className="w-5 h-5" />, color: 'orange' },
    { key: 'compliance', label: 'Compliance', icon: <CheckCircle2 className="w-5 h-5" />, color: 'red' },
  ];

  const getColorClass = (color: string, score: number) => {
    const intensity = score > 80 ? '600' : score > 60 ? '500' : score > 40 ? '400' : '300';
    return `text-${color}-${intensity}`;
  };

  const getBgColorClass = (color: string) => {
    return `bg-${color}-100`;
  };

  return (
    <div className="bg-white rounded-2xl shadow-xl p-6">
      <h3 className="text-xl font-bold text-gray-900 mb-6">Performance Dimensions</h3>
      <div className="space-y-5">
        {dimensions.map((dim) => {
          const score = scores[dim.key as keyof typeof scores];
          return (
            <div key={dim.key}>
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg ${getBgColorClass(dim.color)}`}>
                    {dim.icon}
                  </div>
                  <span className="font-medium text-gray-900">{dim.label}</span>
                </div>
                <span className={`text-2xl font-bold ${getColorClass(dim.color, score)}`}>
                  {score}
                </span>
              </div>
              <div className="h-3 bg-gray-100 rounded-full overflow-hidden">
                <div
                  className={`h-full bg-gradient-to-r from-${dim.color}-400 to-${dim.color}-600 rounded-full transition-all`}
                  style={{ width: `${score}%` }}
                />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

// ============================================================================
// INDUSTRY COMPARISON CARD
// ============================================================================

const IndustryComparisonCard: React.FC<{ benchmark: IndustryBenchmark }> = ({ benchmark }) => {
  return (
    <div className="bg-white rounded-2xl shadow-xl p-6">
      <h3 className="text-xl font-bold text-gray-900 mb-2">Industry Benchmark</h3>
      <p className="text-sm text-gray-600 mb-6">
        {benchmark.industry} • {benchmark.process_type} • {benchmark.sample_size} companies
      </p>
      <div className="space-y-4">
        <BenchmarkMetric
          label="Success Rate"
          your={95}
          median={benchmark.metrics.success_rate * 100}
          topQuartile={benchmark.top_quartile.success_rate * 100}
          unit="%"
          higherIsBetter
        />
        <BenchmarkMetric
          label="Avg Duration"
          your={45}
          median={benchmark.metrics.avg_duration_minutes}
          topQuartile={benchmark.top_quartile.avg_duration_minutes}
          unit=" min"
          higherIsBetter={false}
        />
        <BenchmarkMetric
          label="Automation Rate"
          your={78}
          median={benchmark.metrics.automation_rate * 100}
          topQuartile={benchmark.top_quartile.automation_rate * 100}
          unit="%"
          higherIsBetter
        />
      </div>
    </div>
  );
};

const BenchmarkMetric: React.FC<{
  label: string;
  your: number;
  median: number;
  topQuartile: number;
  unit: string;
  higherIsBetter: boolean;
}> = ({ label, your, median, topQuartile, unit, higherIsBetter }) => {
  const getComparison = () => {
    const vsMedian = higherIsBetter ? your - median : median - your;
    const vsTop = higherIsBetter ? your - topQuartile : topQuartile - your;
    
    if (vsTop >= 0) return { icon: <Trophy className="w-4 h-4" />, color: 'text-yellow-600', label: 'Top Quartile' };
    if (vsMedian >= 0) return { icon: <ArrowUp className="w-4 h-4" />, color: 'text-green-600', label: 'Above Median' };
    return { icon: <ArrowDown className="w-4 h-4" />, color: 'text-red-600', label: 'Below Median' };
  };

  const comparison = getComparison();

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium text-gray-700">{label}</span>
        <div className={`flex items-center gap-1 text-xs font-medium ${comparison.color}`}>
          {comparison.icon}
          {comparison.label}
        </div>
      </div>
      <div className="relative h-8 bg-gray-100 rounded-lg">
        {/* Median line */}
        <div
          className="absolute top-0 bottom-0 w-0.5 bg-gray-400"
          style={{ left: `${(median / Math.max(your, median, topQuartile, 100)) * 100}%` }}
        >
          <div className="absolute -top-6 left-1/2 transform -translate-x-1/2 text-xs text-gray-500 whitespace-nowrap">
            Median: {median.toFixed(0)}{unit}
          </div>
        </div>
        {/* Top quartile line */}
        <div
          className="absolute top-0 bottom-0 w-0.5 bg-yellow-500"
          style={{ left: `${(topQuartile / Math.max(your, median, topQuartile, 100)) * 100}%` }}
        >
          <div className="absolute -bottom-6 left-1/2 transform -translate-x-1/2 text-xs text-yellow-600 whitespace-nowrap">
            Top 25%: {topQuartile.toFixed(0)}{unit}
          </div>
        </div>
        {/* Your value */}
        <div
          className={`absolute top-0 bottom-0 rounded-lg ${comparison.color === 'text-yellow-600' ? 'bg-yellow-500' : comparison.color === 'text-green-600' ? 'bg-green-500' : 'bg-red-500'}`}
          style={{ width: `${(your / Math.max(your, median, topQuartile, 100)) * 100}%` }}
        >
          <span className="absolute right-2 top-1/2 transform -translate-y-1/2 text-xs font-bold text-white">
            {your.toFixed(0)}{unit}
          </span>
        </div>
      </div>
    </div>
  );
};

// ============================================================================
// PEER COMPARISON SECTION
// ============================================================================

const PeerComparisonSection: React.FC<{ comparison: PeerComparison }> = ({ comparison }) => {
  return (
    <div className="space-y-6">
      <div className="bg-white rounded-2xl shadow-xl p-8">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">Peer Group Ranking</h2>
            <p className="text-gray-600 mt-1">{comparison.peer_group}</p>
          </div>
          <div className="text-center">
            <div className="text-5xl font-bold text-blue-600">#{comparison.your_rank}</div>
            <div className="text-sm text-gray-600 mt-1">out of {comparison.total_peers} peers</div>
          </div>
        </div>
        
        <div className="bg-gradient-to-r from-blue-50 to-purple-50 rounded-xl p-6">
          <div className="flex items-center justify-between">
            <span className="text-gray-700">Your Percentile Ranking</span>
            <span className="text-3xl font-bold text-purple-600">{comparison.percentile}th</span>
          </div>
          <div className="mt-4 h-4 bg-white rounded-full overflow-hidden">
            <div
              className="h-full bg-gradient-to-r from-blue-500 to-purple-600 rounded-full"
              style={{ width: `${comparison.percentile}%` }}
            />
          </div>
        </div>
      </div>

      <div className="bg-white rounded-2xl shadow-xl p-6">
        <h3 className="text-xl font-bold text-gray-900 mb-6">Detailed Metric Comparison</h3>
        <div className="space-y-6">
          {comparison.comparison_metrics.map((metric, index) => (
            <div key={index} className="border-2 border-gray-100 rounded-xl p-4">
              <div className="flex items-center justify-between mb-4">
                <h4 className="font-bold text-gray-900">{metric.metric_name}</h4>
                <div className={`flex items-center gap-2 px-3 py-1 rounded-full text-sm font-bold ${
                  metric.variance > 0
                    ? 'bg-green-100 text-green-700'
                    : metric.variance < 0
                    ? 'bg-red-100 text-red-700'
                    : 'bg-gray-100 text-gray-700'
                }`}>
                  {metric.variance > 0 ? <ArrowUp className="w-4 h-4" /> : metric.variance < 0 ? <ArrowDown className="w-4 h-4" /> : <Minus className="w-4 h-4" />}
                  {Math.abs(metric.variance).toFixed(1)}%
                </div>
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div className="text-center">
                  <div className="text-2xl font-bold text-blue-600">{metric.your_value.toFixed(1)}</div>
                  <div className="text-xs text-gray-600 mt-1">Your Value</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-gray-700">{metric.peer_avg.toFixed(1)}</div>
                  <div className="text-xs text-gray-600 mt-1">Peer Average</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">{metric.peer_best.toFixed(1)}</div>
                  <div className="text-xs text-gray-600 mt-1">Peer Best</div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

// ============================================================================
// BEST PRACTICES SECTION
// ============================================================================

const BestPracticesSection: React.FC<{ practices: BestPractice[] }> = ({ practices }) => {
  const [selectedPractice, setSelectedPractice] = useState<BestPractice | null>(null);

  const getImpactColor = (impact: string) => {
    return impact === 'high' ? 'bg-red-100 text-red-700 border-red-300' : 
           impact === 'medium' ? 'bg-yellow-100 text-yellow-700 border-yellow-300' :
           'bg-blue-100 text-blue-700 border-blue-300';
  };

  const getEffortColor = (effort: string) => {
    return effort === 'high' ? 'bg-orange-100 text-orange-700 border-orange-300' :
           effort === 'medium' ? 'bg-yellow-100 text-yellow-700 border-yellow-300' :
           'bg-green-100 text-green-700 border-green-300';
  };

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-2xl shadow-xl p-6">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-gray-900 flex items-center gap-3">
              <Lightbulb className="w-7 h-7 text-yellow-600" />
              Industry Best Practices
            </h2>
            <p className="text-gray-600 mt-1">
              Proven strategies from top-performing organizations
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {practices.map((practice) => (
            <div
              key={practice.id}
              onClick={() => setSelectedPractice(practice)}
              className="border-2 border-gray-200 rounded-xl p-5 hover:border-blue-300 hover:shadow-lg transition-all cursor-pointer"
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <BookOpen className="w-5 h-5 text-blue-600" />
                    <span className="text-xs font-medium text-gray-600">{practice.category}</span>
                  </div>
                  <h3 className="font-bold text-gray-900 text-lg">{practice.title}</h3>
                </div>
              </div>

              <p className="text-gray-700 text-sm mb-4">{practice.description}</p>

              <div className="flex items-center gap-2 mb-4">
                <span className={`px-3 py-1 rounded-full text-xs font-bold border-2 ${getImpactColor(practice.impact)}`}>
                  {practice.impact.toUpperCase()} IMPACT
                </span>
                <span className={`px-3 py-1 rounded-full text-xs font-bold border-2 ${getEffortColor(practice.effort)}`}>
                  {practice.effort.toUpperCase()} EFFORT
                </span>
              </div>

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <Globe className="w-4 h-4" />
                  {(practice.industry_adoption * 100).toFixed(0)}% adoption
                </div>
                <div className="text-green-600 font-bold text-sm">
                  +{(practice.expected_improvement * 100).toFixed(0)}% improvement
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Best Practice Detail Modal */}
      {selectedPractice && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-3xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b-2 border-gray-200">
              <div className="flex items-center justify-between">
                <h3 className="text-2xl font-bold text-gray-900">{selectedPractice.title}</h3>
                <button
                  onClick={() => setSelectedPractice(null)}
                  className="p-2 hover:bg-gray-100 rounded-lg transition-all"
                >
                  ×
                </button>
              </div>
            </div>

            <div className="p-6 space-y-6">
              <div>
                <h4 className="font-bold text-gray-900 mb-2">Description</h4>
                <p className="text-gray-700">{selectedPractice.description}</p>
              </div>

              <div>
                <h4 className="font-bold text-gray-900 mb-3">Implementation Steps</h4>
                <ol className="space-y-2">
                  {selectedPractice.implementation_steps.map((step, index) => (
                    <li key={index} className="flex gap-3">
                      <span className="flex-shrink-0 w-6 h-6 bg-blue-600 text-white rounded-full flex items-center justify-center text-sm font-bold">
                        {index + 1}
                      </span>
                      <span className="text-gray-700">{step}</span>
                    </li>
                  ))}
                </ol>
              </div>

              {selectedPractice.case_studies.length > 0 && (
                <div>
                  <h4 className="font-bold text-gray-900 mb-3">Case Studies</h4>
                  <div className="space-y-3">
                    {selectedPractice.case_studies.map((study, index) => (
                      <div key={index} className="bg-blue-50 border-2 border-blue-200 rounded-xl p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <Building2 className="w-5 h-5 text-blue-600" />
                          <span className="font-bold text-gray-900">{study.company}</span>
                          <span className="text-xs text-gray-600">• {study.industry}</span>
                        </div>
                        <p className="text-gray-700 text-sm mb-1">{study.improvement}</p>
                        <p className="text-xs text-gray-600">Timeframe: {study.timeframe}</p>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <div className="p-6 border-t-2 border-gray-200 flex justify-end gap-3">
              <button
                onClick={() => setSelectedPractice(null)}
                className="px-6 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium hover:bg-gray-200 transition-all"
              >
                Close
              </button>
              <button className="px-6 py-3 bg-green-600 text-white rounded-lg font-medium hover:bg-green-700 transition-all">
                Add to Roadmap
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// ============================================================================
// GAP ANALYSIS SECTION
// ============================================================================

const GapAnalysisSection: React.FC<{ gaps: GapAnalysis[] }> = ({ gaps }) => {
  const getPriorityColor = (priority: string) => {
    return priority === 'critical' ? 'bg-red-100 text-red-700 border-red-300' :
           priority === 'high' ? 'bg-orange-100 text-orange-700 border-orange-300' :
           priority === 'medium' ? 'bg-yellow-100 text-yellow-700 border-yellow-300' :
           'bg-blue-100 text-blue-700 border-blue-300';
  };

  return (
    <div className="bg-white rounded-2xl shadow-xl p-6">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900 flex items-center gap-3">
          <Target className="w-7 h-7 text-purple-600" />
          Performance Gap Analysis
        </h2>
        <p className="text-gray-600 mt-1">
          Identify and close gaps between current and target performance
        </p>
      </div>

      <div className="space-y-6">
        {gaps.map((gap, index) => (
          <div key={index} className="border-2 border-gray-200 rounded-xl p-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-xl font-bold text-gray-900">{gap.dimension}</h3>
                <span className={`inline-block mt-2 px-3 py-1 rounded-full text-xs font-bold border-2 ${getPriorityColor(gap.priority)}`}>
                  {gap.priority.toUpperCase()} PRIORITY
                </span>
              </div>
              <div className="text-right">
                <div className="text-4xl font-bold text-red-600">{gap.gap}</div>
                <div className="text-sm text-gray-600">point gap</div>
              </div>
            </div>

            <div className="mb-4">
              <div className="flex justify-between text-sm text-gray-600 mb-2">
                <span>Current: {gap.current_score}</span>
                <span>Target: {gap.target_score}</span>
              </div>
              <div className="relative h-4 bg-gray-100 rounded-full overflow-hidden">
                <div
                  className="h-full bg-gradient-to-r from-red-400 to-orange-400 rounded-full"
                  style={{ width: `${gap.current_score}%` }}
                />
                <div
                  className="absolute top-0 bottom-0 w-1 bg-green-600"
                  style={{ left: `${gap.target_score}%` }}
                />
              </div>
            </div>

            <div>
              <h4 className="font-bold text-gray-900 mb-2">Recommendations</h4>
              <ul className="space-y-2">
                {gap.recommendations.map((rec, idx) => (
                  <li key={idx} className="flex gap-2 text-gray-700">
                    <CheckCircle2 className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
                    <span>{rec}</span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
