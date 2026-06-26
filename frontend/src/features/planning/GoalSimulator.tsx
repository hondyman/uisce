import React, { useState, useEffect } from 'react';
import { Target, TrendingUp, Calendar, DollarSign, Sliders } from 'lucide-react';
import Slider from 'rc-slider';
import 'rc-slider/assets/index.css';

interface GoalSimulatorProps {
  goalId?: string;
  onSave?: (simulation: any) => void;
}

interface SimulationInputs {
  goalName: string;
  targetAmount: number;
  timeHorizonYears: number;
  currentSavings: number;
  monthlyContribution: number;
  expectedReturn: number;
  inflationRate: number;
}

interface SimulationResults {
  futureValue: number;
  totalContributions: number;
  investmentGrowth: number;
  monthlyRequired: number;
  shortfall: number;
  probabilityOfSuccess: number;
  projectedRetirementAge?: number;
}

export const GoalSimulator: React.FC<GoalSimulatorProps> = ({ goalId, onSave }) => {
  const [inputs, setInputs] = useState<SimulationInputs>({
    goalName: 'Retirement',
    targetAmount: 2000000,
    timeHorizonYears: 25,
    currentSavings: 100000,
    monthlyContribution: 2000,
    expectedReturn: 7.0,
    inflationRate: 3.0,
  });

  const [results, setResults] = useState<SimulationResults | null>(null);
  const [isCalculating, setIsCalculating] = useState(false);

  useEffect(() => {
    calculateProjection();
  }, [inputs]);

  const calculateProjection = async () => {
    setIsCalculating(true);

    // Simulate API call delay
    await new Promise(resolve => setTimeout(resolve, 300));

    // Future Value calculation with monthly contributions
    const monthlyRate = inputs.expectedReturn / 100 / 12;
    const months = inputs.timeHorizonYears * 12;

    // FV of current savings
    const fvCurrent = inputs.currentSavings * Math.pow(1 + monthlyRate, months);

    // FV of monthly contributions (annuity)
    const fvContributions = inputs.monthlyContribution * 
      ((Math.pow(1 + monthlyRate, months) - 1) / monthlyRate);

    const futureValue = fvCurrent + fvContributions;
    const totalContributions = inputs.currentSavings + (inputs.monthlyContribution * months);
    const investmentGrowth = futureValue - totalContributions;

    // Calculate monthly required to reach goal
    const remaining = inputs.targetAmount - fvCurrent;
    const monthlyRequired = remaining > 0
      ? (remaining * monthlyRate) / (Math.pow(1 + monthlyRate, months) - 1)
      : 0;

    const shortfall = Math.max(0, inputs.targetAmount - futureValue);

    // Monte Carlo simulation probability (simplified)
    const ratio = futureValue / inputs.targetAmount;
    const volatility = 0.15; // 15% standard deviation
    const probabilityOfSuccess = Math.min(100, Math.max(0, 
      50 + ((ratio - 1) / volatility) * 34
    ));

    setResults({
      futureValue,
      totalContributions,
      investmentGrowth,
      monthlyRequired,
      shortfall,
      probabilityOfSuccess,
    });

    setIsCalculating(false);
  };

  const updateInput = (key: keyof SimulationInputs, value: number) => {
    setInputs(prev => ({ ...prev, [key]: value }));
  };

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const formatPercent = (value: number) => {
    return `${value.toFixed(1)}%`;
  };

  const saveSimulation = async () => {
    try {
      const response = await fetch('/api/dashboard/simulations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scenarioName: `${inputs.goalName} Projection`,
          scenarioType: 'RETIREMENT',
          inputParameters: inputs,
          projectedOutcomes: results,
        }),
      });

      if (response.ok && onSave) {
        const simulation = await response.json();
        onSave(simulation);
      }
    } catch (error) {
      console.error('Failed to save simulation:', error);
    }
  };

  const isOnTrack = results && results.futureValue >= inputs.targetAmount;

  return (
    <div className="max-w-5xl mx-auto p-6 bg-white rounded-2xl shadow-xl">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="p-3 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-xl">
            <Target className="w-6 h-6 text-white" />
          </div>
          <div>
            <h2 className="text-2xl font-bold text-gray-900">Goal Simulator</h2>
            <p className="text-sm text-gray-600">Adjust sliders to see projected outcomes</p>
          </div>
        </div>

        <button
          onClick={saveSimulation}
          className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors text-sm font-medium"
        >
          Save Simulation
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Input Controls */}
        <div className="space-y-6">
          <div className="bg-gradient-to-br from-gray-50 to-blue-50 p-6 rounded-xl">
            <h3 className="font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <Sliders className="w-5 h-5" />
              Adjust Your Inputs
            </h3>

            <div className="space-y-6">
              <SliderInput
                label="Target Amount"
                value={inputs.targetAmount}
                onChange={(v) => updateInput('targetAmount', v)}
                min={100000}
                max={10000000}
                step={50000}
                format={formatCurrency}
              />

              <SliderInput
                label="Time Horizon (Years)"
                value={inputs.timeHorizonYears}
                onChange={(v) => updateInput('timeHorizonYears', v)}
                min={5}
                max={50}
                step={1}
                format={(v) => `${v} years`}
              />

              <SliderInput
                label="Current Savings"
                value={inputs.currentSavings}
                onChange={(v) => updateInput('currentSavings', v)}
                min={0}
                max={5000000}
                step={10000}
                format={formatCurrency}
              />

              <SliderInput
                label="Monthly Contribution"
                value={inputs.monthlyContribution}
                onChange={(v) => updateInput('monthlyContribution', v)}
                min={0}
                max={20000}
                step={100}
                format={formatCurrency}
              />

              <SliderInput
                label="Expected Return"
                value={inputs.expectedReturn}
                onChange={(v) => updateInput('expectedReturn', v)}
                min={2}
                max={15}
                step={0.5}
                format={formatPercent}
              />

              <SliderInput
                label="Inflation Rate"
                value={inputs.inflationRate}
                onChange={(v) => updateInput('inflationRate', v)}
                min={1}
                max={10}
                step={0.5}
                format={formatPercent}
              />
            </div>
          </div>
        </div>

        {/* Results */}
        <div className="space-y-4">
          <div className={`p-6 rounded-xl border-2 ${
            isOnTrack 
              ? 'bg-gradient-to-br from-green-50 to-emerald-50 border-green-300'
              : 'bg-gradient-to-br from-orange-50 to-red-50 border-orange-300'
          }`}>
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-gray-900">Projected Outcome</h3>
              {isOnTrack ? (
                <span className="px-3 py-1 bg-green-500 text-white text-sm font-medium rounded-full">
                  On Track ✓
                </span>
              ) : (
                <span className="px-3 py-1 bg-orange-500 text-white text-sm font-medium rounded-full">
                  Needs Adjustment
                </span>
              )}
            </div>

            {results && (
              <>
                <div className="mb-4">
                  <p className="text-sm text-gray-600 mb-1">Projected Value</p>
                  <p className="text-4xl font-bold text-gray-900">{formatCurrency(results.futureValue)}</p>
                  <p className="text-sm text-gray-600 mt-1">
                    vs. target of {formatCurrency(inputs.targetAmount)}
                  </p>
                </div>

                <div className="w-full bg-gray-200 rounded-full h-4 overflow-hidden mb-4">
                  <div
                    className={`h-full rounded-full transition-all duration-500 ${
                      isOnTrack
                        ? 'bg-gradient-to-r from-green-400 to-green-600'
                        : 'bg-gradient-to-r from-orange-400 to-orange-600'
                    }`}
                    style={{ width: `${Math.min(100, (results.futureValue / inputs.targetAmount) * 100)}%` }}
                  />
                </div>

                <div className="bg-white p-4 rounded-lg shadow-sm">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm text-gray-600">Success Probability</span>
                    <span className={`text-2xl font-bold ${
                      results.probabilityOfSuccess >= 80 ? 'text-green-600' :
                      results.probabilityOfSuccess >= 60 ? 'text-yellow-600' : 'text-orange-600'
                    }`}>
                      {formatPercent(results.probabilityOfSuccess)}
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className={`h-full rounded-full ${
                        results.probabilityOfSuccess >= 80 ? 'bg-green-500' :
                        results.probabilityOfSuccess >= 60 ? 'bg-yellow-500' : 'bg-orange-500'
                      }`}
                      style={{ width: `${results.probabilityOfSuccess}%` }}
                    />
                  </div>
                </div>
              </>
            )}
          </div>

          {results && (
            <div className="space-y-3">
              <ResultCard
                icon={<DollarSign />}
                label="Investment Growth"
                value={formatCurrency(results.investmentGrowth)}
                subtext={`From ${formatCurrency(results.totalContributions)} contributed`}
                color="blue"
              />

              {results.shortfall > 0 && (
                <ResultCard
                  icon={<TrendingUp />}
                  label="Monthly Required"
                  value={formatCurrency(results.monthlyRequired)}
                  subtext="to reach your goal"
                  color="purple"
                />
              )}

              {results.shortfall > 0 && (
                <ResultCard
                  icon={<Target />}
                  label="Current Shortfall"
                  value={formatCurrency(results.shortfall)}
                  subtext="below target"
                  color="orange"
                />
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

const SliderInput: React.FC<{
  label: string;
  value: number;
  onChange: (value: number) => void;
  min: number;
  max: number;
  step: number;
  format: (value: number) => string;
}> = ({ label, value, onChange, min, max, step, format }) => {
  return (
    <div>
      <div className="flex justify-between items-center mb-2">
        <label className="text-sm font-medium text-gray-700">{label}</label>
        <span className="text-sm font-semibold text-indigo-600">{format(value)}</span>
      </div>
      <Slider
        value={value}
        onChange={onChange}
        min={min}
        max={max}
        step={step}
        trackStyle={{ backgroundColor: '#6366f1', height: 6 }}
        handleStyle={{
          borderColor: '#6366f1',
          backgroundColor: '#fff',
          height: 20,
          width: 20,
          marginTop: -7,
        }}
        railStyle={{ backgroundColor: '#e5e7eb', height: 6 }}
      />
    </div>
  );
};

const ResultCard: React.FC<{
  icon: React.ReactNode;
  label: string;
  value: string;
  subtext: string;
  color: string;
}> = ({ icon, label, value, subtext, color }) => {
  const colorMap: Record<string, string> = {
    blue: 'from-blue-500 to-blue-600',
    purple: 'from-purple-500 to-purple-600',
    orange: 'from-orange-500 to-orange-600',
  };

  return (
    <div className="bg-white p-4 rounded-xl border border-gray-200 shadow-sm">
      <div className="flex items-start gap-3">
        <div className={`p-2 rounded-lg bg-gradient-to-br ${colorMap[color]} text-white`}>
          {icon}
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm text-gray-600">{label}</p>
          <p className="text-xl font-bold text-gray-900">{value}</p>
          <p className="text-xs text-gray-500">{subtext}</p>
        </div>
      </div>
    </div>
  );
};
