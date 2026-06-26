import React, { useState } from 'react';
import { Wand2, Sparkles, AlertCircle, Check, Loader, Edit3, Zap, MessageSquare, FileText, ArrowRight } from 'lucide-react';
import { BusinessProcess, BPStep } from './useBPBuilderAPI';

interface NaturalLanguageBuilderProps {
  onProcessGenerated: (process: BusinessProcess) => void;
  onCancel: () => void;
  tenant: { id: string };
  datasource: { id: string };
}

const EXAMPLE_PROMPTS = [
  {
    title: "Expense Approval",
    text: "Create an expense approval process. Under $1000 goes to manager for approval. Over $1000 requires CFO approval. Send email notifications at each step and after final approval.",
    icon: "💰"
  },
  {
    title: "Employee Onboarding",
    text: "Create an employee onboarding workflow. First collect employee information, then validate it. Send welcome email, create accounts in parallel (IT system, payroll, benefits). Finally assign a buddy and notify HR when complete.",
    icon: "👋"
  },
  {
    title: "Purchase Order",
    text: "Create a purchase order workflow. Validate the order details first. If amount is under $5000, manager approves. If over $5000, both manager and finance director must approve in parallel. After approval, send to procurement and notify requestor.",
    icon: "🛒"
  },
  {
    title: "Document Review",
    text: "Create a document review process. Submit document for review. Route to legal team for review. If changes needed, send back to submitter. If approved, get final signature from department head and archive the document.",
    icon: "📄"
  }
];

export const NaturalLanguageBuilder: React.FC<NaturalLanguageBuilderProps> = ({
  onProcessGenerated,
  onCancel,
  tenant,
  datasource
}) => {
  const [input, setInput] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [generatedProcess, setGeneratedProcess] = useState<BusinessProcess | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [aiInsights, setAiInsights] = useState<string[]>([]);

  const handleGenerate = async () => {
    if (!input.trim()) {
      setError('Please describe your workflow');
      return;
    }

    setIsProcessing(true);
    setError(null);
    setAiInsights([]);

    try {
      const response = await fetch(`/api/business-processes/generate-from-nl?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenant.id,
          'X-Tenant-Datasource-ID': datasource.id,
        },
        body: JSON.stringify({
          description: input,
          tenant_id: tenant.id,
          tenant_instance_id: datasource.id
        })
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to generate process');
      }

      const data = await response.json();
      
      if (data.success && data.data) {
        setGeneratedProcess(data.data.process);
        setAiInsights(data.data.insights || []);
        setShowPreview(true);
      } else {
        throw new Error(data.error || 'Failed to generate process');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate process');
    } finally {
      setIsProcessing(false);
    }
  };

  const handleAccept = () => {
    if (generatedProcess) {
      onProcessGenerated(generatedProcess);
    }
  };

  const handleUseExample = (exampleText: string) => {
    setInput(exampleText);
    setShowPreview(false);
    setGeneratedProcess(null);
    setError(null);
  };

  if (showPreview && generatedProcess) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-purple-50 via-blue-50 to-indigo-50 p-6">
        <div className="max-w-6xl mx-auto">
          {/* Header */}
          <div className="bg-white rounded-xl shadow-lg p-6 mb-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-3 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-lg">
                  <Sparkles className="text-white" size={24} />
                </div>
                <div>
                  <h2 className="text-2xl font-bold text-gray-900">AI-Generated Process Preview</h2>
                  <p className="text-gray-600">Review and customize before saving</p>
                </div>
              </div>
              <div className="flex gap-3">
                <button
                  onClick={() => setShowPreview(false)}
                  className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 font-semibold"
                >
                  Back to Edit
                </button>
                <button
                  onClick={handleAccept}
                  className="px-6 py-2 bg-gradient-to-r from-purple-600 to-indigo-600 text-white rounded-lg hover:from-purple-700 hover:to-indigo-700 font-semibold flex items-center gap-2"
                >
                  <Check size={18} />
                  Accept & Continue
                </button>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-6">
            {/* Process Details */}
            <div className="col-span-2 space-y-6">
              <div className="bg-white rounded-xl shadow-lg p-6">
                <h3 className="text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                  <FileText size={20} />
                  Process Overview
                </h3>
                <div className="space-y-3">
                  <div>
                    <label className="text-sm font-semibold text-gray-600">Process Name</label>
                    <p className="text-lg text-gray-900 mt-1">{generatedProcess.processName}</p>
                  </div>
                  <div>
                    <label className="text-sm font-semibold text-gray-600">Entity Type</label>
                    <p className="text-gray-900 mt-1">{generatedProcess.entity}</p>
                  </div>
                  <div>
                    <label className="text-sm font-semibold text-gray-600">Description</label>
                    <p className="text-gray-700 mt-1">{generatedProcess.description}</p>
                  </div>
                </div>
              </div>

              {/* Steps Preview */}
              <div className="bg-white rounded-xl shadow-lg p-6">
                <h3 className="text-lg font-bold text-gray-900 mb-4">
                  Workflow Steps ({generatedProcess.steps.length})
                </h3>
                <div className="space-y-3">
                  {generatedProcess.steps.map((step, idx) => (
                    <div key={step.id} className="border border-gray-200 rounded-lg p-4">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-2">
                            <span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs font-semibold">
                              Step {step.stepOrder}
                            </span>
                            <span className="px-2 py-1 bg-gray-100 text-gray-700 rounded text-xs font-semibold">
                              {step.stepType}
                            </span>
                            {step.executionMode === 'parallel' && (
                              <span className="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs font-semibold">
                                Parallel
                              </span>
                            )}
                          </div>
                          <h4 className="font-semibold text-gray-900">{step.stepName}</h4>
                          {step.description && (
                            <p className="text-sm text-gray-600 mt-1">{step.description}</p>
                          )}
                          <div className="flex items-center gap-4 mt-2 text-sm text-gray-600">
                            {step.durationHours && (
                              <span>⏱️ {step.durationHours}h</span>
                            )}
                            {step.assigneeRole && (
                              <span>👤 {step.assigneeRole}</span>
                            )}
                            {step.conditionLogic && (
                              <span>🔀 Conditional</span>
                            )}
                            {step.approvalChain && (
                              <span>✅ {step.approvalChain.approvalMode} approval</span>
                            )}
                          </div>
                        </div>
                      </div>
                      {idx < generatedProcess.steps.length - 1 && (
                        <div className="flex justify-center mt-2">
                          <ArrowRight className="text-gray-400 rotate-90" size={20} />
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* AI Insights Sidebar */}
            <div className="space-y-6">
              <div className="bg-gradient-to-br from-purple-100 to-indigo-100 rounded-xl shadow-lg p-6">
                <h3 className="text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                  <Sparkles size={20} className="text-purple-600" />
                  AI Insights
                </h3>
                <div className="space-y-3">
                  {aiInsights.map((insight, idx) => (
                    <div key={idx} className="bg-white rounded-lg p-3 text-sm text-gray-700">
                      {insight}
                    </div>
                  ))}
                  {aiInsights.length === 0 && (
                    <p className="text-sm text-gray-600">No additional insights for this process.</p>
                  )}
                </div>
              </div>

              <div className="bg-white rounded-xl shadow-lg p-6">
                <h3 className="text-lg font-bold text-gray-900 mb-4">Quick Stats</h3>
                <div className="space-y-3">
                  <div className="flex justify-between items-center">
                    <span className="text-gray-600">Total Steps</span>
                    <span className="font-semibold text-gray-900">{generatedProcess.steps.length}</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-gray-600">Est. Duration</span>
                    <span className="font-semibold text-gray-900">
                      {generatedProcess.steps.reduce((acc, s) => acc + s.durationHours, 0)}h
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-gray-600">Approval Steps</span>
                    <span className="font-semibold text-gray-900">
                      {generatedProcess.steps.filter(s => s.stepType === 'approve').length}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-gray-600">Parallel Steps</span>
                    <span className="font-semibold text-gray-900">
                      {generatedProcess.steps.filter(s => s.executionMode === 'parallel').length}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-50 via-blue-50 to-indigo-50 p-6">
      <div className="max-w-5xl mx-auto">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-3 mb-4">
            <div className="p-4 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-2xl">
              <Wand2 className="text-white" size={32} />
            </div>
            <h1 className="text-4xl font-bold bg-gradient-to-r from-purple-600 to-indigo-600 bg-clip-text text-transparent">
              Natural Language Process Builder
            </h1>
          </div>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto">
            Describe your workflow in plain English, and AI will create a complete business process for you
          </p>
        </div>

        {/* Main Input Card */}
        <div className="bg-white rounded-2xl shadow-xl p-8 mb-6">
          <div className="mb-6">
            <label className="flex items-center gap-2 text-lg font-semibold text-gray-900 mb-3">
              <MessageSquare size={20} />
              Describe Your Workflow
            </label>
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="Example: Create an expense approval process. Under $1000 goes to manager. Over $1000 requires CFO approval. Send email notifications at each step..."
              className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-purple-500 focus:border-transparent resize-none text-gray-900 placeholder-gray-400"
              rows={6}
              disabled={isProcessing}
            />
            <div className="flex items-center justify-between mt-3">
              <p className="text-sm text-gray-500">
                💡 Tip: Mention approvals, conditions, notifications, and parallel steps
              </p>
              <span className="text-sm text-gray-400">
                {input.length} characters
              </span>
            </div>
          </div>

          {error && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3">
              <AlertCircle className="text-red-600 flex-shrink-0 mt-0.5" size={20} />
              <div>
                <h4 className="font-semibold text-red-900">Error</h4>
                <p className="text-sm text-red-700">{error}</p>
              </div>
            </div>
          )}

          <div className="flex gap-3">
            <button
              onClick={handleGenerate}
              disabled={isProcessing || !input.trim()}
              className="flex-1 px-6 py-3 bg-gradient-to-r from-purple-600 to-indigo-600 text-white rounded-xl hover:from-purple-700 hover:to-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed font-semibold text-lg flex items-center justify-center gap-3 transition-all"
            >
              {isProcessing ? (
                <>
                  <Loader className="animate-spin" size={24} />
                  Generating with AI...
                </>
              ) : (
                <>
                  <Sparkles size={24} />
                  Generate Process
                </>
              )}
            </button>
            <button
              onClick={onCancel}
              className="px-6 py-3 border-2 border-gray-300 rounded-xl text-gray-700 hover:bg-gray-50 font-semibold transition-all"
            >
              Cancel
            </button>
          </div>
        </div>

        {/* Example Prompts */}
        <div className="bg-white rounded-2xl shadow-xl p-8">
          <div className="flex items-center gap-2 mb-6">
            <Zap className="text-purple-600" size={24} />
            <h3 className="text-xl font-bold text-gray-900">Example Workflows</h3>
          </div>
          <div className="grid grid-cols-2 gap-4">
            {EXAMPLE_PROMPTS.map((example, idx) => (
              <button
                key={idx}
                onClick={() => handleUseExample(example.text)}
                disabled={isProcessing}
                className="text-left p-4 border-2 border-gray-200 rounded-xl hover:border-purple-400 hover:bg-purple-50 transition-all group disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <div className="flex items-start gap-3">
                  <span className="text-3xl">{example.icon}</span>
                  <div className="flex-1">
                    <h4 className="font-semibold text-gray-900 group-hover:text-purple-600 mb-2">
                      {example.title}
                    </h4>
                    <p className="text-sm text-gray-600 line-clamp-3">
                      {example.text}
                    </p>
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>

        {/* Features */}
        <div className="mt-8 grid grid-cols-3 gap-6">
          <div className="bg-white rounded-xl shadow-lg p-6 text-center">
            <div className="w-12 h-12 bg-purple-100 rounded-full flex items-center justify-center mx-auto mb-3">
              <Sparkles className="text-purple-600" size={24} />
            </div>
            <h4 className="font-semibold text-gray-900 mb-2">AI-Powered</h4>
            <p className="text-sm text-gray-600">
              Advanced AI understands workflow nuances and generates optimized processes
            </p>
          </div>
          <div className="bg-white rounded-xl shadow-lg p-6 text-center">
            <div className="w-12 h-12 bg-indigo-100 rounded-full flex items-center justify-center mx-auto mb-3">
              <Edit3 className="text-indigo-600" size={24} />
            </div>
            <h4 className="font-semibold text-gray-900 mb-2">Fully Editable</h4>
            <p className="text-sm text-gray-600">
              Preview and customize every detail before saving to your workspace
            </p>
          </div>
          <div className="bg-white rounded-xl shadow-lg p-6 text-center">
            <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-3">
              <Zap className="text-blue-600" size={24} />
            </div>
            <h4 className="font-semibold text-gray-900 mb-2">10x Faster</h4>
            <p className="text-sm text-gray-600">
              Create complex workflows in seconds instead of hours of manual configuration
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};
