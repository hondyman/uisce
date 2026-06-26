/**
 * Next Best Action (NBA) Types
 * 
 * Type definitions for the AI-Driven Next Best Action Engine.
 * Transforms reactive advisor workflows into proactive, AI-orchestrated
 * client engagement strategies.
 */

// ============================================================================
// Signal Types
// ============================================================================

export type SignalCategory =
  | 'BEHAVIORAL'
  | 'MARKET'
  | 'LIFECYCLE'
  | 'PORTFOLIO'
  | 'ENGAGEMENT';

export type SignalSource =
  | 'CRM_ACTIVITY'
  | 'PORTFOLIO_EVENTS'
  | 'MARKET_CONDITIONS'
  | 'LIFE_EVENTS'
  | 'BEHAVIORAL_PATTERNS'
  | 'COMPETITOR_INTELLIGENCE'
  | 'SOCIAL_SIGNALS'
  | 'REGULATORY_TRIGGERS';

export type SignalType =
  | 'LARGE_WITHDRAWAL_PENDING'
  | 'EMAIL_ENGAGEMENT_DROP'
  | 'CONCENTRATED_POSITION_ALERT'
  | 'EXCESS_CASH_DRAG'
  | 'TAX_LOSS_HARVEST_OPPORTUNITY'
  | 'CONCENTRATED_POSITION_RISK'
  | 'ENGAGEMENT_DECLINE'
  | 'LOW_EMAIL_ENGAGEMENT'
  | 'VOLATILITY_EXPOSURE'
  | 'RETIREMENT_APPROACHING'
  | 'INHERITANCE_DETECTED'
  | 'JOB_CHANGE_DETECTED'
  | 'ANNIVERSARY_UPCOMING'
  | 'REBALANCING_DUE'
  | 'COMPLIANCE_DEADLINE';

export interface DetectedSignal {
  signalId: string;
  clientId: string;
  clientName: string;
  signalType: SignalType;
  signalCategory: SignalCategory;
  signalSource: SignalSource;
  detectedAt: string;
  strength: number; // 0.0 to 1.0
  expiryAt?: string;
  rawData: Record<string, unknown>;
  processedInsights?: Record<string, unknown>;
}

export interface SignalDefinition {
  definitionId: string;
  signalType: SignalType;
  signalCategory: SignalCategory;
  description: string;
  severityThreshold: number;
  recommendedActions: string[];
  mlModelId?: string;
}

// ============================================================================
// Action Types
// ============================================================================

export type ActionChannel =
  | 'PHONE'
  | 'EMAIL'
  | 'IN_PERSON'
  | 'VIDEO_CALL'
  | 'AUTOMATED_MESSAGE'
  | 'PORTAL_NOTIFICATION';

export type ActionPriority =
  | 'CRITICAL'
  | 'HIGH'
  | 'MEDIUM'
  | 'LOW'
  | 'OPTIONAL';

export type ActionCategory =
  | 'PROACTIVE_OUTREACH'
  | 'SERVICE_DELIVERY'
  | 'PORTFOLIO_MANAGEMENT'
  | 'RELATIONSHIP_BUILDING'
  | 'COMPLIANCE'
  | 'TAX_PLANNING';

export type ActionOutcome =
  | 'SUCCESS'
  | 'PARTIAL'
  | 'FAILED'
  | 'PENDING'
  | 'DISMISSED';

export interface ActionTemplate {
  emailSubject?: string;
  emailBody?: string;
  callScript?: string;
  meetingAgenda?: string;
  presentationSlides?: string[];
  followUpEmail?: string;
}

export interface ActionSuccessMetrics {
  successMetric: string;
  targetValue: number;
}

export interface NBAAction {
  actionId: string;
  actionCode: string;
  actionName: string;
  actionCategory: ActionCategory;
  description: string;
  defaultChannel: ActionChannel;
  estimatedDurationMinutes: number;
  estimatedRevenueImpact: number;
  clientValueImpact: number;
  automationEligible: boolean;
  templateContent: ActionTemplate;
  requiredAdvisorSkills: string[];
  complianceReviewRequired: boolean;
  successMetrics: ActionSuccessMetrics;
}

// ============================================================================
// NBA Recommendation Types
// ============================================================================

export interface NextBestAction {
  actionId: string;
  clientId: string;
  clientName: string;
  clientTier: 'VIP' | 'HIGH_NET_WORTH' | 'STANDARD';
  actionType: string;
  actionName: string;
  actionCategory: ActionCategory;
  confidence: number; // 0.0 to 1.0
  urgencyScore: number; // 0.0 to 1.0
  expectedValue: number; // Dollar amount
  successProbability: number; // 0.0 to 1.0
  triggerSignal: SignalType;
  triggerSignalStrength: number;
  reasoning: string;
  recommendedChannel: ActionChannel;
  estimatedDurationMinutes: number;
  templateContent: ActionTemplate;
  recommendedAt: string;
  expiresAt?: string;
  priority: ActionPriority;
}

export interface NBARecommendationBatch {
  advisorId: string;
  generatedAt: string;
  recommendations: NextBestAction[];
  totalExpectedValue: number;
  totalEstimatedTime: number;
  criticalCount: number;
  highValueCount: number;
}

// ============================================================================
// Outcome Tracking Types
// ============================================================================

export interface NBAActionOutcome {
  outcomeId: string;
  actionId: string;
  clientId: string;
  advisorId: string;
  triggerSignalType: SignalType;
  recommendedAt: string;
  executedAt?: string;
  completedAt?: string;
  executionChannel: ActionChannel;
  
  // Outcome metrics
  clientResponded: boolean;
  responseTimeHours?: number;
  actionSuccessful: boolean;
  revenueGenerated?: number;
  clientSatisfactionChange?: number;
  aumChange?: number;
  
  // Feedback
  advisorFeedback?: string;
  advisorRating?: number; // 1-5
  dismissReason?: string;
}

export interface OutcomeStats {
  totalActions: number;
  executedActions: number;
  successfulActions: number;
  successRate: number;
  avgResponseTimeHours: number;
  totalRevenueGenerated: number;
  avgSatisfactionChange: number;
}

// ============================================================================
// Analytics Types
// ============================================================================

export interface ModelMetrics {
  f1Score: number;
  precisionAtK: number;
  recallAtK: number;
  avgPredictedValue: number;
  avgActualValue: number;
  modelVersion: string;
  trainedAt: string;
  samplesUsed: number;
}

export interface NBAAnalytics {
  period: 'DAY' | 'WEEK' | 'MONTH' | 'QUARTER';
  startDate: string;
  endDate: string;
  
  // Action metrics
  totalRecommendations: number;
  actionsExecuted: number;
  actionsDismissed: number;
  actionsExpired: number;
  executionRate: number;
  
  // Outcome metrics
  successRate: number;
  avgConfidence: number;
  avgUrgency: number;
  totalRevenueImpact: number;
  avgRevenuePerAction: number;
  
  // Model performance
  modelAccuracy: number;
  topPerformingActions: ActionPerformance[];
  signalEffectiveness: SignalEffectiveness[];
  
  // Advisor performance
  advisorRankings: AdvisorPerformance[];
}

export interface ActionPerformance {
  actionCode: string;
  actionName: string;
  executionCount: number;
  successRate: number;
  avgRevenue: number;
  avgSatisfactionChange: number;
}

export interface SignalEffectiveness {
  signalType: SignalType;
  detectionCount: number;
  actionsTaken: number;
  conversionRate: number;
  avgImpact: number;
}

export interface AdvisorPerformance {
  advisorId: string;
  advisorName: string;
  actionsExecuted: number;
  successRate: number;
  totalRevenue: number;
  avgResponseTime: number;
  clientSatisfaction: number;
}

// ============================================================================
// Client Context Types
// ============================================================================

export interface ClientProfile {
  clientId: string;
  name: string;
  tier: 'VIP' | 'HIGH_NET_WORTH' | 'STANDARD';
  age: number;
  netWorth: number;
  aum: number;
  tenureYears: number;
  numAccounts: number;
  annualFees: number;
  riskToleranceScore: number;
  liquidityNeedsScore: number;
  taxBracket: number;
  retirementYearsAway: number;
  portfolioReturnYtd: number;
  portfolioReturn3yr: number;
  sharpeRatio: number;
  maxDrawdownYtd: number;
  equityAllocation: number;
  fixedIncomeAllocation: number;
  alternativeAllocation: number;
  cashAllocation: number;
  avgMeetingFrequency: number;
  lastMeetingDaysAgo: number;
  emailOpenRate: number;
  portalLogins90d: number;
  referralsGiven: number;
  satisfactionScore: number;
  flightRiskScore: number;
}

export interface ClientSignalHistory {
  clientId: string;
  signals: DetectedSignal[];
  actionsReceived: NextBestAction[];
  outcomes: NBAActionOutcome[];
}

// ============================================================================
// API Request/Response Types
// ============================================================================

export interface GetRecommendationsRequest {
  advisorId: string;
  limit?: number;
  filterCategory?: ActionCategory;
  filterPriority?: ActionPriority;
  minConfidence?: number;
  minExpectedValue?: number;
}

export interface ExecuteActionRequest {
  actionId: string;
  advisorId: string;
  channel: ActionChannel;
  notes?: string;
}

export interface CompleteActionRequest {
  actionId: string;
  outcome: ActionOutcome;
  notes: string;
  revenueGenerated?: number;
  clientResponded: boolean;
  advisorRating?: number;
}

export interface DismissActionRequest {
  actionId: string;
  reason: 'NOT_RELEVANT' | 'ALREADY_DONE' | 'WRONG_TIMING' | 'CLIENT_OPTED_OUT' | 'OTHER';
  notes?: string;
}

// ============================================================================
// WebSocket Event Types
// ============================================================================

export type NBAEventType =
  | 'NEW_RECOMMENDATION'
  | 'RECOMMENDATION_UPDATED'
  | 'RECOMMENDATION_EXPIRED'
  | 'SIGNAL_DETECTED'
  | 'ACTION_COMPLETED'
  | 'MODEL_UPDATED';

export interface NBAWebSocketEvent {
  eventType: NBAEventType;
  timestamp: string;
  payload: NextBestAction | DetectedSignal | NBAActionOutcome | ModelMetrics;
}

// ============================================================================
// Component Props Types
// ============================================================================

export interface NBADashboardProps {
  advisorId: string;
  tenantId: string;
  datasourceId: string;
}

export interface ActionCardProps {
  action: NextBestAction;
  onExecute: () => void;
  onDismiss: (reason: string) => void;
  onViewDetails: () => void;
}

export interface ActionExecutionModalProps {
  action: NextBestAction;
  onComplete: (outcome: CompleteActionRequest) => void;
  onClose: () => void;
}

export interface SignalMonitorProps {
  clientId?: string;
  limit?: number;
  onSignalClick?: (signal: DetectedSignal) => void;
}

export interface NBAAnalyticsDashboardProps {
  advisorId?: string;
  period: 'DAY' | 'WEEK' | 'MONTH' | 'QUARTER';
}
