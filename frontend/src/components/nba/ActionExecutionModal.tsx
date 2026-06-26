/**
 * ActionExecutionModal.tsx
 *
 * One-Click Action Execution Modal for NBA System
 *
 * Features:
 * - Pre-filled email/call scripts based on action templates
 * - Client context integration (portfolio, history, preferences)
 * - Real-time personalization with AI-generated content
 * - Multi-channel execution (Email, Phone, Video, In-Person, Task)
 * - Outcome tracking for ML feedback loop
 * - Compliance guardrails and audit logging
 *
 * Target: 80% reduction in action execution time via automation
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  X,
  Send,
  Phone,
  Mail,
  Video,
  User,
  Calendar,
  Clock,
  CheckCircle,
  AlertCircle,
  Loader2,
  Copy,
  Sparkles,
  MessageSquare,
  RefreshCw,
  Save,
  Shield,
  History,
  Target,
  DollarSign,
  ChevronDown,
  ChevronUp,
  Wand2,
  Mic,
  Pause,
  ExternalLink,
  Bell,
} from 'lucide-react';
import type {
  NextBestAction,
  ActionChannel,
  CompleteActionRequest,
  ActionOutcome,
} from '../../types/nba';

// ====================
// Types & Interfaces
// ====================

interface ActionExecutionModalProps {
  action: NextBestAction;
  onComplete: (outcome: CompleteActionRequest) => void;
  onClose: () => void;
}

interface EmailTemplate {
  subject: string;
  body: string;
  callToAction: string;
}

interface CallScript {
  opening: string;
  keyPoints: string[];
  objectionHandlers: Record<string, string>;
  closing: string;
}

interface TaskDetails {
  title: string;
  description: string;
  dueDate: Date;
  priority: 'HIGH' | 'MEDIUM' | 'LOW';
}

interface ExecutionState {
  channel: ActionChannel;
  status: 'DRAFT' | 'REVIEWING' | 'EXECUTING' | 'COMPLETED' | 'SCHEDULED';
  emailTemplate: EmailTemplate;
  callScript: CallScript;
  taskDetails: TaskDetails;
  notes: string;
  selectedTime: Date | null;
  aiSuggestions: string[];
  complianceChecks: ComplianceCheck[];
  revenueGenerated: number;
  clientResponded: boolean;
  advisorRating: number;
}

interface ComplianceCheck {
  rule: string;
  status: 'PASS' | 'WARN' | 'FAIL';
  message: string;
}

// ====================
// Channel Configuration
// ====================

const CHANNEL_CONFIG: Record<ActionChannel, {
  icon: React.ElementType;
  label: string;
  color: string;
  bgColor: string;
}> = {
  EMAIL: { icon: Mail, label: 'Email', color: 'text-blue-600', bgColor: 'bg-blue-50' },
  PHONE: { icon: Phone, label: 'Phone Call', color: 'text-green-600', bgColor: 'bg-green-50' },
  VIDEO_CALL: { icon: Video, label: 'Video Call', color: 'text-purple-600', bgColor: 'bg-purple-50' },
  IN_PERSON: { icon: User, label: 'In-Person', color: 'text-amber-600', bgColor: 'bg-amber-50' },
  AUTOMATED_MESSAGE: { icon: Bell, label: 'Automated', color: 'text-slate-600', bgColor: 'bg-slate-100' },
  PORTAL_NOTIFICATION: { icon: MessageSquare, label: 'Portal', color: 'text-cyan-600', bgColor: 'bg-cyan-50' },
};

// ====================
// Mock Data Generators
// ====================

function generateEmailTemplate(action: NextBestAction): EmailTemplate {
  return {
    subject: `${action.actionName} - ${action.clientName}`,
    body: `Dear ${action.clientName.split(' ')[0]},

I hope this message finds you well. Based on our analysis of your portfolio and recent market developments, I wanted to reach out regarding an opportunity that aligns with your investment goals.

${action.reasoning}

Given the current market conditions and our understanding of your financial objectives, this action could have a meaningful impact on your wealth strategy.

I would welcome the opportunity to discuss this further at your earliest convenience. Please let me know what time works best for a brief call or meeting.

Best regards,
[Your Name]
Wealth Management Advisor`,
    callToAction: 'Schedule a 15-minute call to discuss',
  };
}

function generateCallScript(action: NextBestAction): CallScript {
  return {
    opening: `Good [morning/afternoon] ${action.clientName.split(' ')[0]}, this is [Your Name] from [Firm]. How are you today? I'm calling because I've identified an opportunity that I believe would be valuable for your portfolio.`,
    keyPoints: [
      `Action: ${action.actionName}`,
      `Reasoning: ${action.reasoning}`,
      `Expected value: $${action.expectedValue.toLocaleString()}`,
      `Timeline: ${action.estimatedDurationMinutes} minutes to complete`,
      `Confidence level: ${Math.round(action.confidence * 100)}%`,
    ],
    objectionHandlers: {
      'Need to think about it': 'I completely understand. This is an important decision. Would it be helpful if I sent you a summary email with the key points we discussed?',
      'Market concerns': 'That\'s a valid concern. Our analysis accounts for current market conditions, and this recommendation actually helps mitigate some of those risks.',
      'Too busy right now': 'I appreciate how busy you are. Would it be helpful to schedule a brief 10-minute call next week when you have more time?',
      'Want to consult spouse': 'Absolutely, involving your spouse in financial decisions is important. Would you like to schedule a call when you can both be available?',
    },
    closing: `Thank you for your time today, ${action.clientName.split(' ')[0]}. I'll send you a follow-up email summarizing what we discussed. Is there anything else I can help you with?`,
  };
}

function generateTaskDetails(action: NextBestAction): TaskDetails {
  return {
    title: action.actionName,
    description: `Complete action for client ${action.clientName}: ${action.reasoning}`,
    dueDate: new Date(Date.now() + 24 * 60 * 60 * 1000),
    priority: action.priority === 'CRITICAL' || action.priority === 'HIGH' ? 'HIGH' : 'MEDIUM',
  };
}

function generateComplianceChecks(action: NextBestAction): ComplianceCheck[] {
  return [
    {
      rule: 'Suitability Check',
      status: 'PASS',
      message: 'Action is suitable for client\'s risk profile',
    },
    {
      rule: 'Disclosure Requirements',
      status: 'PASS',
      message: 'All required disclosures included',
    },
    {
      rule: 'Communication Policy',
      status: action.recommendedChannel === 'EMAIL' || action.recommendedChannel === 'PHONE' ? 'PASS' : 'WARN',
      message: action.recommendedChannel === 'EMAIL' || action.recommendedChannel === 'PHONE'
        ? 'Using standard communication channel'
        : 'Consider verifying client channel preferences',
    },
    {
      rule: 'Contact Frequency',
      status: 'PASS',
      message: 'Within acceptable contact frequency limits',
    },
  ];
}

function generateAISuggestions(action: NextBestAction): string[] {
  const suggestions = [
    `Signal detected: ${action.triggerSignal} with ${Math.round(action.triggerSignalStrength * 100)}% strength`,
    'Personalize the opening based on your last interaction',
    'Include a specific time slot suggestion to increase response rate',
  ];

  if (action.clientTier === 'VIP' || action.clientTier === 'HIGH_NET_WORTH') {
    suggestions.push('For high-value clients, consider offering a complimentary portfolio review');
  }

  if (action.priority === 'CRITICAL' || action.priority === 'HIGH') {
    suggestions.push('Emphasize the time-sensitive nature of this opportunity');
  }

  return suggestions;
}

// ====================
// Helper Components
// ====================

interface ChannelTabProps {
  channel: ActionChannel;
  isActive: boolean;
  onClick: () => void;
}

function ChannelTab({ channel, isActive, onClick }: ChannelTabProps) {
  const config = CHANNEL_CONFIG[channel];
  const Icon = config.icon;

  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
        isActive
          ? `${config.bgColor} ${config.color}`
          : 'text-slate-600 hover:bg-slate-100'
      }`}
      title={`Switch to ${config.label}`}
    >
      <Icon className="w-4 h-4" />
      {config.label}
    </button>
  );
}

interface ComplianceStatusProps {
  checks: ComplianceCheck[];
}

function ComplianceStatus({ checks }: ComplianceStatusProps) {
  const allPassed = checks.every(c => c.status === 'PASS');
  const hasWarnings = checks.some(c => c.status === 'WARN');
  const hasFails = checks.some(c => c.status === 'FAIL');

  return (
    <div className={`p-3 rounded-lg ${
      hasFails ? 'bg-red-50 border border-red-200' :
      hasWarnings ? 'bg-amber-50 border border-amber-200' :
      'bg-green-50 border border-green-200'
    }`}>
      <div className="flex items-center gap-2 mb-2">
        <Shield className={`w-4 h-4 ${
          hasFails ? 'text-red-600' :
          hasWarnings ? 'text-amber-600' :
          'text-green-600'
        }`} />
        <span className={`text-sm font-medium ${
          hasFails ? 'text-red-700' :
          hasWarnings ? 'text-amber-700' :
          'text-green-700'
        }`}>
          {allPassed ? 'All Compliance Checks Passed' :
           hasWarnings ? 'Compliance Warnings' :
           'Compliance Issues Detected'}
        </span>
      </div>
      <div className="space-y-1">
        {checks.map((check, index) => (
          <div key={index} className="flex items-start gap-2 text-xs">
            {check.status === 'PASS' ? (
              <CheckCircle className="w-3 h-3 text-green-500 mt-0.5" />
            ) : check.status === 'WARN' ? (
              <AlertCircle className="w-3 h-3 text-amber-500 mt-0.5" />
            ) : (
              <AlertCircle className="w-3 h-3 text-red-500 mt-0.5" />
            )}
            <span className="text-slate-600">{check.message}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ====================
// Main Component
// ====================

export function ActionExecutionModal({
  action,
  onComplete,
  onClose,
}: ActionExecutionModalProps) {
  const [state, setState] = useState<ExecutionState>({
    channel: action.recommendedChannel,
    status: 'DRAFT',
    emailTemplate: generateEmailTemplate(action),
    callScript: generateCallScript(action),
    taskDetails: generateTaskDetails(action),
    notes: '',
    selectedTime: null,
    aiSuggestions: generateAISuggestions(action),
    complianceChecks: generateComplianceChecks(action),
    revenueGenerated: 0,
    clientResponded: true,
    advisorRating: 5,
  });

  const [isGenerating, setIsGenerating] = useState(false);
  const [showCallScript, setShowCallScript] = useState(false);
  const [copied, setCopied] = useState(false);
  const [showScheduler, setShowScheduler] = useState(false);
  const [isRecording, setIsRecording] = useState(false);
  const [executionTime, setExecutionTime] = useState(0);

  // Track execution time
  useEffect(() => {
    const startTime = Date.now();
    const timer = setInterval(() => {
      setExecutionTime(Math.floor((Date.now() - startTime) / 1000));
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  const handleRegenerateContent = useCallback(async () => {
    setIsGenerating(true);
    // Simulate AI regeneration
    await new Promise(resolve => setTimeout(resolve, 1500));
    setState(prev => ({
      ...prev,
      emailTemplate: generateEmailTemplate(action),
      callScript: generateCallScript(action),
      aiSuggestions: generateAISuggestions(action),
    }));
    setIsGenerating(false);
  }, [action]);

  const handleCopyContent = useCallback((content: string) => {
    navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, []);

  const handleExecute = useCallback((outcome: ActionOutcome) => {
    setState(prev => ({ ...prev, status: 'EXECUTING' }));

    // Simulate execution
    setTimeout(() => {
      setState(prev => ({ ...prev, status: 'COMPLETED' }));

      const completeRequest: CompleteActionRequest = {
        actionId: action.actionId,
        outcome: outcome,
        notes: state.notes,
        revenueGenerated: state.revenueGenerated || undefined,
        clientResponded: state.clientResponded,
        advisorRating: state.advisorRating,
      };

      onComplete(completeRequest);
    }, 1000);
  }, [action.actionId, state.notes, state.revenueGenerated, state.clientResponded, state.advisorRating, onComplete]);

  const renderEmailEditor = () => (
    <div className="space-y-4">
      {/* AI Suggestions */}
      <div className="bg-gradient-to-r from-purple-50 to-indigo-50 rounded-lg p-4 border border-purple-100">
        <div className="flex items-center gap-2 mb-2">
          <Sparkles className="w-4 h-4 text-purple-600" />
          <span className="text-sm font-medium text-purple-700">AI Suggestions</span>
        </div>
        <ul className="space-y-1">
          {state.aiSuggestions.map((suggestion, index) => (
            <li key={index} className="text-xs text-purple-600 flex items-start gap-2">
              <span className="text-purple-400">•</span>
              {suggestion}
            </li>
          ))}
        </ul>
      </div>

      {/* Subject Line */}
      <div>
        <label htmlFor="email-subject" className="block text-sm font-medium text-slate-700 mb-1">Subject Line</label>
        <input
          id="email-subject"
          type="text"
          value={state.emailTemplate.subject}
          onChange={(e) => setState(prev => ({
            ...prev,
            emailTemplate: { ...prev.emailTemplate, subject: e.target.value }
          }))}
          placeholder="Enter email subject..."
          className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
      </div>

      {/* Email Body */}
      <div>
        <div className="flex items-center justify-between mb-1">
          <label htmlFor="email-body" className="text-sm font-medium text-slate-700">Email Body</label>
          <div className="flex items-center gap-2">
            <button
              onClick={() => handleCopyContent(state.emailTemplate.body)}
              className="text-xs text-slate-500 hover:text-slate-700 flex items-center gap-1"
              title="Copy email content"
            >
              {copied ? <CheckCircle className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
              {copied ? 'Copied!' : 'Copy'}
            </button>
            <button
              onClick={handleRegenerateContent}
              disabled={isGenerating}
              className="text-xs text-indigo-600 hover:text-indigo-700 flex items-center gap-1"
              title="Regenerate content with AI"
            >
              {isGenerating ? <Loader2 className="w-3 h-3 animate-spin" /> : <Wand2 className="w-3 h-3" />}
              Regenerate
            </button>
          </div>
        </div>
        <textarea
          id="email-body"
          value={state.emailTemplate.body}
          onChange={(e) => setState(prev => ({
            ...prev,
            emailTemplate: { ...prev.emailTemplate, body: e.target.value }
          }))}
          rows={12}
          placeholder="Enter email body..."
          className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
      </div>

      {/* Call to Action */}
      <div>
        <label htmlFor="email-cta" className="block text-sm font-medium text-slate-700 mb-1">Call to Action</label>
        <input
          id="email-cta"
          type="text"
          value={state.emailTemplate.callToAction}
          onChange={(e) => setState(prev => ({
            ...prev,
            emailTemplate: { ...prev.emailTemplate, callToAction: e.target.value }
          }))}
          placeholder="Enter call to action..."
          className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
      </div>
    </div>
  );

  const renderCallEditor = () => (
    <div className="space-y-4">
      {/* Call Recording Controls */}
      <div className="bg-green-50 rounded-lg p-4 border border-green-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Phone className="w-4 h-4 text-green-600" />
            <span className="text-sm font-medium text-green-700">Call Assistant</span>
          </div>
          <button
            onClick={() => setIsRecording(!isRecording)}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm ${
              isRecording
                ? 'bg-red-100 text-red-600'
                : 'bg-green-100 text-green-600'
            }`}
            title={isRecording ? 'Stop recording' : 'Start recording'}
          >
            {isRecording ? (
              <>
                <Pause className="w-4 h-4" />
                Stop Recording
              </>
            ) : (
              <>
                <Mic className="w-4 h-4" />
                Start Recording
              </>
            )}
          </button>
        </div>
        {isRecording && (
          <div className="mt-2 flex items-center gap-2 text-xs text-green-600">
            <span className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />
            Recording in progress... {Math.floor(executionTime / 60)}:{String(executionTime % 60).padStart(2, '0')}
          </div>
        )}
      </div>

      {/* Call Script Toggle */}
      <button
        onClick={() => setShowCallScript(!showCallScript)}
        className="w-full flex items-center justify-between p-3 bg-slate-50 rounded-lg hover:bg-slate-100 transition-colors"
        title={showCallScript ? 'Hide call script' : 'Show call script'}
      >
        <div className="flex items-center gap-2">
          <MessageSquare className="w-4 h-4 text-slate-600" />
          <span className="text-sm font-medium text-slate-700">Call Script</span>
        </div>
        {showCallScript ? (
          <ChevronUp className="w-4 h-4 text-slate-400" />
        ) : (
          <ChevronDown className="w-4 h-4 text-slate-400" />
        )}
      </button>

      {showCallScript && (
        <div className="space-y-4 border border-slate-200 rounded-lg p-4">
          {/* Opening */}
          <div>
            <label htmlFor="call-opening" className="block text-xs font-medium text-slate-500 mb-1 uppercase tracking-wider">
              Opening
            </label>
            <textarea
              id="call-opening"
              value={state.callScript.opening}
              onChange={(e) => setState(prev => ({
                ...prev,
                callScript: { ...prev.callScript, opening: e.target.value }
              }))}
              rows={3}
              placeholder="Enter opening script..."
              className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          {/* Key Points */}
          <div>
            <span className="block text-xs font-medium text-slate-500 mb-1 uppercase tracking-wider">
              Key Points
            </span>
            <ul className="space-y-2">
              {state.callScript.keyPoints.map((point, index) => (
                <li key={index} className="flex items-start gap-2">
                  <span className="w-5 h-5 rounded-full bg-indigo-100 text-indigo-600 text-xs flex items-center justify-center flex-shrink-0 mt-0.5">
                    {index + 1}
                  </span>
                  <input
                    type="text"
                    value={point}
                    onChange={(e) => {
                      const newPoints = [...state.callScript.keyPoints];
                      newPoints[index] = e.target.value;
                      setState(prev => ({
                        ...prev,
                        callScript: { ...prev.callScript, keyPoints: newPoints }
                      }));
                    }}
                    placeholder={`Key point ${index + 1}...`}
                    title={`Edit key point ${index + 1}`}
                    className="flex-1 px-3 py-1.5 border border-slate-200 rounded text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                </li>
              ))}
            </ul>
          </div>

          {/* Objection Handlers */}
          <div>
            <span className="block text-xs font-medium text-slate-500 mb-2 uppercase tracking-wider">
              Objection Handlers
            </span>
            <div className="space-y-2">
              {Object.entries(state.callScript.objectionHandlers).map(([objection, response]) => (
                <div key={objection} className="bg-amber-50 rounded-lg p-3">
                  <div className="text-xs font-medium text-amber-700 mb-1">
                    If client says: &quot;{objection}&quot;
                  </div>
                  <div className="text-xs text-amber-600">{response}</div>
                </div>
              ))}
            </div>
          </div>

          {/* Closing */}
          <div>
            <label htmlFor="call-closing" className="block text-xs font-medium text-slate-500 mb-1 uppercase tracking-wider">
              Closing
            </label>
            <textarea
              id="call-closing"
              value={state.callScript.closing}
              onChange={(e) => setState(prev => ({
                ...prev,
                callScript: { ...prev.callScript, closing: e.target.value }
              }))}
              rows={2}
              placeholder="Enter closing script..."
              className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
        </div>
      )}

      {/* Call Notes */}
      <div>
        <label htmlFor="call-notes" className="block text-sm font-medium text-slate-700 mb-1">Call Notes</label>
        <textarea
          id="call-notes"
          value={state.notes}
          onChange={(e) => setState(prev => ({ ...prev, notes: e.target.value }))}
          rows={4}
          placeholder="Add notes during or after the call..."
          className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
      </div>
    </div>
  );

  const renderTaskEditor = () => (
    <div className="space-y-4">
      {/* Task Title */}
      <div>
        <label htmlFor="task-title" className="block text-sm font-medium text-slate-700 mb-1">Task Title</label>
        <input
          id="task-title"
          type="text"
          value={state.taskDetails.title}
          onChange={(e) => setState(prev => ({
            ...prev,
            taskDetails: { ...prev.taskDetails, title: e.target.value }
          }))}
          placeholder="Enter task title..."
          className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
      </div>

      {/* Description */}
      <div>
        <label htmlFor="task-desc" className="block text-sm font-medium text-slate-700 mb-1">Description</label>
        <textarea
          id="task-desc"
          value={state.taskDetails.description}
          onChange={(e) => setState(prev => ({
            ...prev,
            taskDetails: { ...prev.taskDetails, description: e.target.value }
          }))}
          rows={4}
          placeholder="Enter task description..."
          className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
      </div>

      {/* Due Date & Priority */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label htmlFor="task-due" className="block text-sm font-medium text-slate-700 mb-1">Due Date</label>
          <input
            id="task-due"
            type="datetime-local"
            value={state.taskDetails.dueDate.toISOString().slice(0, 16)}
            onChange={(e) => setState(prev => ({
              ...prev,
              taskDetails: { ...prev.taskDetails, dueDate: new Date(e.target.value) }
            }))}
            title="Select due date"
            className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>
        <div>
          <label htmlFor="task-priority" className="block text-sm font-medium text-slate-700 mb-1">Priority</label>
          <select
            id="task-priority"
            value={state.taskDetails.priority}
            onChange={(e) => setState(prev => ({
              ...prev,
              taskDetails: { ...prev.taskDetails, priority: e.target.value as 'HIGH' | 'MEDIUM' | 'LOW' }
            }))}
            title="Select task priority"
            className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          >
            <option value="HIGH">High Priority</option>
            <option value="MEDIUM">Medium Priority</option>
            <option value="LOW">Low Priority</option>
          </select>
        </div>
      </div>
    </div>
  );

  const renderChannelContent = () => {
    switch (state.channel) {
      case 'EMAIL':
        return renderEmailEditor();
      case 'PHONE':
      case 'VIDEO_CALL':
        return renderCallEditor();
      case 'AUTOMATED_MESSAGE':
      case 'PORTAL_NOTIFICATION':
        return renderTaskEditor();
      case 'IN_PERSON':
        return (
          <div className="space-y-4">
            <div className="bg-amber-50 rounded-lg p-4 border border-amber-200">
              <div className="flex items-center gap-2 mb-2">
                <User className="w-4 h-4 text-amber-600" />
                <span className="text-sm font-medium text-amber-700">In-Person Meeting</span>
              </div>
              <p className="text-xs text-amber-600">
                Use the call script for talking points during the meeting. Notes will be saved to the client record.
              </p>
            </div>
            {renderCallEditor()}
          </div>
        );
      default:
        return renderEmailEditor();
    }
  };

  if (state.status === 'COMPLETED') {
    return (
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
        <div className="bg-white rounded-xl shadow-2xl w-full max-w-md p-8 text-center">
          <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <CheckCircle className="w-8 h-8 text-green-600" />
          </div>
          <h2 className="text-xl font-bold text-slate-900 mb-2">Action Executed!</h2>
          <p className="text-slate-600 mb-4">
            The action has been recorded and will contribute to AI model training.
          </p>
          <div className="bg-slate-50 rounded-lg p-4 mb-6 text-left">
            <div className="flex items-center justify-between text-sm mb-2">
              <span className="text-slate-600">Execution Time</span>
              <span className="font-medium">{executionTime} seconds</span>
            </div>
            <div className="flex items-center justify-between text-sm mb-2">
              <span className="text-slate-600">Channel Used</span>
              <span className="font-medium">{CHANNEL_CONFIG[state.channel].label}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-slate-600">Expected Value</span>
              <span className="font-medium text-green-600">
                ${action.expectedValue.toLocaleString()}
              </span>
            </div>
          </div>
          <button
            onClick={onClose}
            className="w-full px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
            title="Close modal"
          >
            Done
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-indigo-100 rounded-lg">
              <Target className="w-5 h-5 text-indigo-600" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-slate-900">Execute Action</h2>
              <p className="text-sm text-slate-600">{action.actionName}</p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div className="text-xs text-slate-500">
              <Clock className="w-3 h-3 inline mr-1" />
              Time: {executionTime}s
            </div>
            <button
              onClick={onClose}
              className="p-1 hover:bg-slate-100 rounded"
              title="Close modal"
            >
              <X className="w-5 h-5 text-slate-500" />
            </button>
          </div>
        </div>

        {/* Client Context Bar */}
        <div className="px-6 py-3 bg-slate-50 border-b border-slate-200 flex items-center gap-6">
          <div className="flex items-center gap-2">
            <User className="w-4 h-4 text-slate-400" />
            <span className="text-sm font-medium">{action.clientName}</span>
          </div>
          <div className="flex items-center gap-2">
            <DollarSign className="w-4 h-4 text-slate-400" />
            <span className="text-sm text-slate-600">
              Expected: ${action.expectedValue.toLocaleString()}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <History className="w-4 h-4 text-slate-400" />
            <span className="text-sm text-slate-600">
              Tier: {action.clientTier}
            </span>
          </div>
          <div className="ml-auto">
            <button
              className="text-xs text-indigo-600 hover:text-indigo-700 flex items-center gap-1"
              title="View full client profile"
            >
              View Full Profile
              <ExternalLink className="w-3 h-3" />
            </button>
          </div>
        </div>

        {/* Channel Tabs */}
        <div className="px-6 py-3 border-b border-slate-200 flex items-center gap-2 overflow-x-auto">
          {Object.keys(CHANNEL_CONFIG).map(channel => (
            <ChannelTab
              key={channel}
              channel={channel as ActionChannel}
              isActive={state.channel === channel}
              onClick={() => setState(prev => ({ ...prev, channel: channel as ActionChannel }))}
            />
          ))}
        </div>

        {/* Content Area */}
        <div className="flex-1 overflow-y-auto">
          <div className="grid grid-cols-3 gap-6 p-6">
            {/* Main Editor - 2 columns */}
            <div className="col-span-2">
              {renderChannelContent()}
            </div>

            {/* Sidebar - 1 column */}
            <div className="space-y-4">
              {/* Expected Value Card */}
              <div className="bg-gradient-to-br from-green-50 to-emerald-50 rounded-lg p-4 border border-green-200">
                <div className="text-xs text-green-600 mb-1">Expected Value</div>
                <div className="text-2xl font-bold text-green-700">
                  ${action.expectedValue.toLocaleString()}
                </div>
                <div className="text-xs text-green-600 mt-2">
                  {Math.round(action.confidence * 100)}% confidence
                </div>
              </div>

              {/* Compliance Status */}
              <ComplianceStatus checks={state.complianceChecks} />

              {/* Trigger Signal */}
              <div className="bg-slate-50 rounded-lg p-4">
                <div className="text-xs font-medium text-slate-500 mb-2 uppercase tracking-wider">
                  Trigger Signal
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <span className="w-2 h-2 rounded-full bg-indigo-500" />
                  <span className="text-slate-700">{action.triggerSignal}</span>
                </div>
                <div className="text-xs text-slate-500 mt-1">
                  Strength: {Math.round(action.triggerSignalStrength * 100)}%
                </div>
              </div>

              {/* Quick Actions */}
              <div className="space-y-2">
                <button
                  onClick={() => handleCopyContent(state.emailTemplate.body)}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-600 hover:bg-slate-100 rounded-lg"
                  title="Copy content to clipboard"
                >
                  <Copy className="w-4 h-4" />
                  Copy to Clipboard
                </button>
                <button
                  onClick={handleRegenerateContent}
                  disabled={isGenerating}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-600 hover:bg-slate-100 rounded-lg"
                  title="Regenerate content with AI"
                >
                  <RefreshCw className={`w-4 h-4 ${isGenerating ? 'animate-spin' : ''}`} />
                  Regenerate with AI
                </button>
                <button
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-600 hover:bg-slate-100 rounded-lg"
                  title="Save as template"
                >
                  <Save className="w-4 h-4" />
                  Save as Template
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="px-6 py-4 border-t border-slate-200 flex items-center justify-between bg-slate-50">
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-600 hover:bg-slate-200 rounded-lg transition-colors"
            title="Cancel and close"
          >
            Cancel
          </button>

          <div className="flex items-center gap-3">
            <button
              onClick={() => setShowScheduler(!showScheduler)}
              className="flex items-center gap-2 px-4 py-2 border border-slate-300 text-slate-700 rounded-lg hover:bg-white transition-colors"
              title="Schedule for later"
            >
              <Calendar className="w-4 h-4" />
              Schedule for Later
            </button>
            <button
              onClick={() => handleExecute('SUCCESS')}
              disabled={state.status === 'EXECUTING'}
              className="flex items-center gap-2 px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
              title="Execute action now"
            >
              {state.status === 'EXECUTING' ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Executing...
                </>
              ) : (
                <>
                  <Send className="w-4 h-4" />
                  Execute Now
                </>
              )}
            </button>
          </div>
        </div>

        {/* Scheduler Modal */}
        {showScheduler && (
          <div className="absolute inset-0 bg-black/30 flex items-center justify-center">
            <div className="bg-white rounded-lg shadow-xl p-6 w-96">
              <h3 className="text-lg font-bold text-slate-900 mb-4">Schedule Action</h3>
              <label htmlFor="schedule-time" className="block text-sm font-medium text-slate-700 mb-2">
                Select date and time
              </label>
              <input
                id="schedule-time"
                type="datetime-local"
                onChange={(e) => setState(prev => ({ ...prev, selectedTime: new Date(e.target.value) }))}
                className="w-full px-3 py-2 border border-slate-200 rounded-lg mb-4"
                title="Select schedule time"
              />
              <div className="flex gap-3">
                <button
                  onClick={() => setShowScheduler(false)}
                  className="flex-1 px-4 py-2 border border-slate-300 rounded-lg"
                  title="Cancel scheduling"
                >
                  Cancel
                </button>
                <button
                  onClick={() => {
                    setShowScheduler(false);
                    // Schedule action would be handled here
                  }}
                  disabled={!state.selectedTime}
                  className="flex-1 px-4 py-2 bg-indigo-600 text-white rounded-lg disabled:opacity-50"
                  title="Confirm schedule"
                >
                  Schedule
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default ActionExecutionModal;
