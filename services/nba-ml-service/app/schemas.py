"""
Pydantic schemas for NBA ML Service API.
"""

from datetime import datetime
from typing import Optional, List, Dict, Any
from pydantic import BaseModel, Field
from enum import Enum


class SignalCategory(str, Enum):
    BEHAVIORAL = "BEHAVIORAL"
    MARKET = "MARKET"
    LIFECYCLE = "LIFECYCLE"
    PORTFOLIO = "PORTFOLIO"
    ENGAGEMENT = "ENGAGEMENT"


class SignalType(str, Enum):
    LARGE_WITHDRAWAL_PENDING = "LARGE_WITHDRAWAL_PENDING"
    EMAIL_ENGAGEMENT_DROP = "EMAIL_ENGAGEMENT_DROP"
    CONCENTRATED_POSITION_ALERT = "CONCENTRATED_POSITION_ALERT"
    EXCESS_CASH_DRAG = "EXCESS_CASH_DRAG"
    TAX_LOSS_HARVEST_OPPORTUNITY = "TAX_LOSS_HARVEST_OPPORTUNITY"
    CONCENTRATED_POSITION_RISK = "CONCENTRATED_POSITION_RISK"
    ENGAGEMENT_DECLINE = "ENGAGEMENT_DECLINE"
    LOW_EMAIL_ENGAGEMENT = "LOW_EMAIL_ENGAGEMENT"
    VOLATILITY_EXPOSURE = "VOLATILITY_EXPOSURE"
    RETIREMENT_APPROACHING = "RETIREMENT_APPROACHING"
    INHERITANCE_DETECTED = "INHERITANCE_DETECTED"
    JOB_CHANGE_DETECTED = "JOB_CHANGE_DETECTED"
    ANNIVERSARY_UPCOMING = "ANNIVERSARY_UPCOMING"
    REBALANCING_DUE = "REBALANCING_DUE"
    COMPLIANCE_DEADLINE = "COMPLIANCE_DEADLINE"


class ActionChannel(str, Enum):
    PHONE = "PHONE"
    EMAIL = "EMAIL"
    IN_PERSON = "IN_PERSON"
    VIDEO_CALL = "VIDEO_CALL"
    AUTOMATED_MESSAGE = "AUTOMATED_MESSAGE"
    PORTAL_NOTIFICATION = "PORTAL_NOTIFICATION"


class ActionPriority(str, Enum):
    CRITICAL = "CRITICAL"
    HIGH = "HIGH"
    MEDIUM = "MEDIUM"
    LOW = "LOW"
    OPTIONAL = "OPTIONAL"


class ActionCategory(str, Enum):
    PROACTIVE_OUTREACH = "PROACTIVE_OUTREACH"
    SERVICE_DELIVERY = "SERVICE_DELIVERY"
    PORTFOLIO_MANAGEMENT = "PORTFOLIO_MANAGEMENT"
    RELATIONSHIP_BUILDING = "RELATIONSHIP_BUILDING"
    COMPLIANCE = "COMPLIANCE"
    TAX_PLANNING = "TAX_PLANNING"


class DetectedSignal(BaseModel):
    """Input signal detected from client monitoring."""
    signal_id: str = Field(..., description="Unique signal identifier")
    signal_type: str = Field(..., description="Type of signal detected")
    category: str = Field(..., description="Signal category")
    strength: float = Field(..., ge=0.0, le=1.0, description="Signal strength/confidence")
    detected_at: datetime = Field(default_factory=datetime.utcnow)
    raw_data: Dict[str, Any] = Field(default_factory=dict)
    client_tier: Optional[str] = Field(None, description="Client tier (VIP, HIGH_NET_WORTH, STANDARD)")
    expiry_at: Optional[datetime] = None


class ClientProfile(BaseModel):
    """Client profile features for ML model."""
    age: Optional[int] = None
    net_worth: Optional[float] = None
    aum: Optional[float] = None
    tenure_years: Optional[float] = None
    num_accounts: Optional[int] = None
    annual_fees: Optional[float] = None
    risk_tolerance_score: Optional[float] = None
    liquidity_needs_score: Optional[float] = None
    tax_bracket: Optional[float] = None
    retirement_years_away: Optional[int] = None
    portfolio_return_ytd: Optional[float] = None
    portfolio_return_3yr: Optional[float] = None
    sharpe_ratio: Optional[float] = None
    max_drawdown_ytd: Optional[float] = None
    equity_allocation: Optional[float] = None
    fixed_income_allocation: Optional[float] = None
    alternative_allocation: Optional[float] = None
    cash_allocation: Optional[float] = None
    avg_meeting_frequency: Optional[float] = None
    last_meeting_days_ago: Optional[int] = None
    email_open_rate: Optional[float] = None
    portal_logins_90d: Optional[int] = None
    referrals_given: Optional[int] = None
    satisfaction_score: Optional[float] = None
    flight_risk_score: Optional[float] = None


class ActionTemplate(BaseModel):
    """Template content for action execution."""
    email_subject: Optional[str] = None
    email_body: Optional[str] = None
    call_script: Optional[str] = None
    meeting_agenda: Optional[str] = None
    presentation_slides: Optional[List[str]] = None
    follow_up_email: Optional[str] = None


class NextBestActionRecommendation(BaseModel):
    """A single NBA recommendation from the model."""
    action_id: str = Field(..., description="Unique action identifier")
    action_type: str = Field(..., description="Action type code")
    action_name: str = Field(..., description="Human-readable action name")
    action_category: ActionCategory
    confidence: float = Field(..., ge=0.0, le=1.0, description="Model confidence in this recommendation")
    urgency_score: float = Field(..., ge=0.0, le=1.0, description="Urgency score")
    expected_value: float = Field(..., description="Expected revenue impact in dollars")
    success_probability: float = Field(..., ge=0.0, le=1.0, description="Probability of successful outcome")
    trigger_signal: str = Field(..., description="Signal that triggered this recommendation")
    reasoning: str = Field(..., description="AI-generated explanation")
    recommended_channel: ActionChannel
    estimated_duration_minutes: int = Field(default=30)
    template_content: ActionTemplate = Field(default_factory=ActionTemplate)
    priority: ActionPriority = Field(default=ActionPriority.MEDIUM)


class PredictionRequest(BaseModel):
    """Request body for prediction endpoint."""
    client_id: str = Field(..., description="Client UUID")
    signal: DetectedSignal
    text_context: Optional[str] = Field(None, description="Additional text context (CRM notes, emails)")
    client_profile: Optional[ClientProfile] = None


class PredictionResponse(BaseModel):
    """Response from prediction endpoint."""
    client_id: str
    recommendations: List[NextBestActionRecommendation]
    generated_at: datetime


class BatchPredictionRequest(BaseModel):
    """Request for batch predictions."""
    requests: List[PredictionRequest]


class BatchPredictionResponse(BaseModel):
    """Response for batch predictions."""
    results: List[PredictionResponse]


class TrainingRequest(BaseModel):
    """Request to trigger model retraining."""
    lookback_days: int = Field(default=90, ge=7, le=365)
    min_samples: int = Field(default=100, ge=10)


class TrainingMetrics(BaseModel):
    """Metrics from model training."""
    f1_score: float
    precision_at_k: float
    recall_at_k: float
    num_samples: int
    training_time_seconds: float


class HealthResponse(BaseModel):
    """Health check response."""
    status: str
    model_loaded: bool
    version: str


class ModelInfo(BaseModel):
    """Model information."""
    model_version: str
    num_actions: int
    trained_at: Optional[datetime] = None
    training_samples: Optional[int] = None
    metrics: Optional[TrainingMetrics] = None
