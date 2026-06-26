/**
 * Phase 3: Scenario Analysis & Stress Testing - Type Definitions
 * 
 * Comprehensive TypeScript interfaces for:
 * - Stress scenarios configuration
 * - Simulation execution & results
 * - Collaboration & annotations
 * - Real-time data streaming
 * 
 * @module types/scenarios
 */

/**
 * Stress Scenario Configuration
 * Defines market stress parameters for simulation
 */
export interface StressScenario {
  id: string;
  name: string;
  description?: string;
  
  // Market factor shocks (percentage or basis points)
  equityMarketMove: number;        // -100 to +100 (%)
  interestRateShift: number;       // -500 to +500 (bps, basis points)
  volatilityChange: number;        // -100 to +200 (%)
  creditSpreadWidening: number;    // -100 to +500 (bps)
  currencyShift?: number;          // Optional currency move (%)
  commodityPriceChange?: number;   // Optional commodity move (%)
  
  // Scope & metadata
  portfoliosIncluded: string[];    // Portfolio IDs to simulate
  scope: 'all-portfolios' | 'selected' | 'comparison-pair';
  
  // Timestamps & ownership
  createdAt: Date;
  createdBy: string;               // User ID
  updatedAt?: Date;
  tags?: string[];
  
  // Historical reference
  isHistorical?: boolean;          // e.g., "2008 Financial Crisis"
  historicalDate?: Date;
}

/**
 * Simulation Execution Run
 * Represents a single stress test simulation job
 */
export interface SimulationRun {
  id: string;
  simulationId: string;            // Unique execution ID
  scenarioId: string;              // Reference to StressScenario
  
  // Status tracking
  status: 'queued' | 'running' | 'completed' | 'failed' | 'aborted';
  progress: number;                // 0-100 (%)
  
  // Execution details
  portfoliosTotal: number;
  portfoliosProcessed: number;
  portfoliosFailed: number;
  
  // Timing
  startedAt: Date;
  completedAt?: Date;
  estimatedDuration: number;       // seconds
  actualDuration?: number;
  
  // Performance
  executionEngine: 'wasm' | 'js' | 'native';
  threatLevel?: number;            // CPU/memory stress (0-100)
  
  // Error handling
  errorMessage?: string;
  failedPortfolios?: Array<{
    portfolioId: string;
    error: string;
    timestamp: Date;
  }>;
}

/**
 * Individual Portfolio Simulation Result
 * PnL and risk metrics from a single scenario run
 */
export interface SimulationResult {
  id: string;
  runId: string;
  simulationId: string;
  scenarioId: string;
  portfolioId: string;
  portfolioName: string;
  
  // Core results
  simulatedPnL: number;            // Millions USD
  simulatedPnLPercent: number;     // % of AUM
  
  // Risk metrics
  volatility?: number;             // Annualized (%)
  var95_1d?: number;               // Value at Risk 95% confidence (1 day, millions)
  expectedShortfall?: number;      // CVaR (millions)
  
  // Confidence & validation
  confidenceLevel: number;         // 0-100 (%)
  validationStatus: 'valid' | 'warning' | 'error';
  validationMessage?: string;
  
  // Processing metadata
  processingTimeMs: number;
  dataQuality: number;             // 0-100
  completedAt: Date;
  
  // Factor attribution (optional detail)
  factorAttribution?: {
    equitySensitivity: number;
    rateSensitivity: number;
    creditSensitivity: number;
    volatilitySensitivity: number;
  };
}

/**
 * Multi-Scenario Comparison Data
 * Side-by-side results for multiple stress scenarios
 */
export interface ScenarioComparison {
  id: string;
  comparisonName: string;
  portfolioId: string;
  portfolioName: string;
  
  // Results for each scenario
  results: Map<string, SimulationResult>;  // scenarioId -> result
  
  // Computed metrics
  maxDrawdown: number;             // Worst case PnL across scenarios
  minDrawdown: number;
  pnlVariance: number;             // Spread in outcomes
  correlationToHistorical?: number;
  
  // Metadata
  createdAt: Date;
  comparedScenarioIds: string[];
}

/**
 * Collaborative Annotation
 * Comments, notes, and team insights on simulation results
 */
export interface Annotation {
  id: string;
  annotationId: string;            // Unique annotation ID
  simulationId: string;            // Which simulation
  
  // Author info
  userId: string;
  userName: string;
  userAvatar?: string;
  userRole?: string;               // Portfolio Manager, Risk Analyst, etc.
  
  // Content
  text: string;
  type: 'comment' | 'finding' | 'concern' | 'insight' | 'question';
  
  // Cell reference (optional)
  cellReference?: {
    portfolioName?: string;
    scenarioName?: string;
    metric?: string;               // 'pnl' | 'var' | 'volatility'
  };
  
  // Mentions & threading
  mentions?: string[];             // @mention user IDs
  parentAnnotationId?: string;     // For threaded replies
  replies?: Annotation[];
  
  // Metadata
  createdAt: Date;
  updatedAt?: Date;
  isEdited: boolean;
  isPinned: boolean;
  
  // Attachments
  attachments?: Array<{
    id: string;
    name: string;
    type: string;
    url: string;
  }>;
  
  // Tags for sorting
  tags?: string[];
}

/**
 * Real-Time Collaboration State
 * Tracks active users and their activity
 */
export interface CollaborationState {
  simulationId: string;
  activeUsers: Array<{
    userId: string;
    userName: string;
    userAvatar?: string;
    joinedAt: Date;
    lastActivity: Date;
    currentView: 'results' | 'annotations' | 'comparison';
  }>;
  annotations: Annotation[];
  lastUpdate: Date;
}

/**
 * WebSocket Streaming Message Types
 * Real-time updates during simulation execution
 */
export type SimulationStreamMessage =
  | { type: 'connected'; simulationId: string; }
  | { type: 'progress'; progress: number; portfoliosProcessed: number; totalPortfolios: number; estimatedSecondsRemaining: number; }
  | { type: 'result'; result: SimulationResult; }
  | { type: 'completed'; totalResultsCount: number; succeeded: number; failed: number; }
  | { type: 'error'; errorMessage: string; portfolio?: string; }
  | { type: 'annotation-added'; annotation: Annotation; }
  | { type: 'annotation-removed'; annotationId: string; }
  | { type: 'user-joined'; user: CollaborationState['activeUsers'][0]; }
  | { type: 'user-left'; userId: string; };

/**
 * Scenario Export Format
 * Package for sharing/archiving
 */
export interface ScenarioExport {
  format: 'pdf' | 'excel' | 'json' | 'csv';
  simulationId: string;
  scenarios: StressScenario[];
  results: SimulationResult[];
  annotations?: Annotation[];
  metadata: {
    exportedAt: Date;
    exportedBy: string;
    includeAnnotations: boolean;
    includeCharts: boolean;
  };
}

/**
 * Form validation state for scenario configuration
 */
export interface ScenarioFormErrors {
  scenarioName?: string;
  equityMarketMove?: string;
  interestRateShift?: string;
  volatilityChange?: string;
  creditSpreadWidening?: string;
  portfoliosIncluded?: string;
  general?: string;
}

/**
 * API Request/Response structures
 */

export interface StartSimulationRequest {
  scenarioId: string;
  portfolioIds?: string[];
  scope?: 'all-portfolios' | 'selected';
}

export interface StartSimulationResponse {
  simulationId: string;
  status: 'queued';
  estimatedDuration: number;
  queuePosition?: number;
}

export interface SimulationStatusResponse {
  run: SimulationRun;
  results: SimulationResult[];
  progress: number;
  status: SimulationRun['status'];
}

export interface AddAnnotationRequest {
  simulationId: string;
  userId: string;
  text: string;
  type: Annotation['type'];
  cellReference?: Annotation['cellReference'];
  mentions?: string[];
}

export interface AddAnnotationResponse {
  annotation: Annotation;
  createdAt: Date;
}

export interface AnnotationsResponse {
  annotations: Annotation[];
  total: number;
  lastUpdated: Date;
}

/**
 * Component Props Interfaces
 */

export interface ScenarioConfigDialogProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (scenario: StressScenario) => Promise<void>;
  isLoading?: boolean;
  portfolios?: Array<{ id: string; name: string; aum: number }>;
}

export interface SimulationProgressProps {
  simulationRun: SimulationRun;
  results: SimulationResult[];
  onAbort: () => Promise<void>;
  isAborting?: boolean;
}

export interface MultiScenarioComparisonProps {
  scenarios: StressScenario[];
  results: SimulationResult[];
  portfolios: Array<{ id: string; name: string }>;
  annotations: Annotation[];
  onAddAnnotation: (annotation: Annotation) => Promise<void>;
}

export interface CollaborativeAnnotationsPanelProps {
  simulationId: string;
  annotations: Annotation[];
  activeUsers: CollaborationState['activeUsers'];
  onAddAnnotation: (text: string, type: Annotation['type']) => Promise<void>;
  onReplyToAnnotation: (parentId: string, text: string) => Promise<void>;
  onPinAnnotation: (annotationId: string) => Promise<void>;
}

export interface StreamingResultsTableProps {
  results: SimulationResult[];
  isLive?: boolean;
  portfolioCount: number;
  processedCount: number;
  onRowClick?: (result: SimulationResult) => void;
}

/**
 * Hook Return Types
 */

export interface UseSimulationReturn {
  // State
  run: SimulationRun | null;
  results: SimulationResult[];
  isRunning: boolean;
  progress: number;
  error: string | null;
  
  // Methods
  startSimulation: (scenario: StressScenario, portfolios?: string[]) => Promise<string>;
  abortSimulation: (runId: string) => Promise<void>;
  getResults: (runId: string) => Promise<SimulationResult[]>;
  
  // Cleanup
  reset: () => void;
}

export interface UseSimulationResultsReturn {
  // State
  results: SimulationResult[];
  isStreaming: boolean;
  lastUpdate: Date | null;
  connectionStatus: 'connected' | 'disconnected' | 'reconnecting';
  
  // Methods
  subscribe: (runId: string) => void;
  unsubscribe: () => void;
  
  // Events
  onResultReceived?: (result: SimulationResult) => void;
  onStreamComplete?: () => void;
  onStreamError?: (error: string) => void;
}

export interface UseAnnotationsReturn {
  // State
  annotations: Annotation[];
  isLoading: boolean;
  error: string | null;
  
  // Methods
  fetchAnnotations: (simulationId: string) => Promise<void>;
  addAnnotation: (annotation: Omit<Annotation, 'id' | 'createdAt'>) => Promise<Annotation>;
  updateAnnotation: (id: string, updates: Partial<Annotation>) => Promise<Annotation>;
  deleteAnnotation: (id: string) => Promise<void>;
  pinAnnotation: (id: string) => Promise<void>;
  replyToAnnotation: (parentId: string, reply: Omit<Annotation, 'id' | 'createdAt'>) => Promise<Annotation>;
}

/**
 * Utility Type Helpers
 */

export type ScenarioMetric = 'pnl' | 'volatility' | 'var' | 'sharpeRatio';
export type ScenarioStatus = 'draft' | 'ready' | 'running' | 'completed' | 'archived';
export type UserRole = 'portfolio-manager' | 'risk-analyst' | 'compliance-officer' | 'viewer';

/**
 * Constants for UI/Validation
 */

export const SCENARIO_CONFIG_CONSTRAINTS = {
  equityMarketMove: { min: -100, max: 100, step: 1 },
  interestRateShift: { min: -500, max: 500, step: 10 },
  volatilityChange: { min: -100, max: 200, step: 5 },
  creditSpreadWidening: { min: -100, max: 500, step: 10 },
} as const;

export const SIMULATION_TIMEOUT_MS = 300000; // 5 minutes
export const STREAMING_UPDATE_INTERVAL_MS = 500; // 500ms
export const ANNOTATION_FETCH_DEBOUNCE_MS = 300;
