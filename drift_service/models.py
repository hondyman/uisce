"""Pydantic models for Drift Detection Service"""

from pydantic import BaseModel, Field
from typing import Optional, List
from datetime import datetime

class DriftRequest(BaseModel):
    """Request to compute drift for a feature"""
    feature_id: str = Field(..., description="e.g., feature:orders.revenue_v1")
    method: str = Field(..., description="ks | psi | chi2 | classifier")
    baseline_window: str = Field(default="30d", description="e.g., 30d, 7d")
    eval_window: str = Field(default="1d", description="e.g., 1d, 1h")
    threshold: Optional[float] = Field(None, description="Override default threshold")
    tenant_id: str = Field(default="default")
    region: str = Field(default="us-east-1")

class DriftResult(BaseModel):
    """Result of drift computation"""
    feature_id: str
    method: str
    score: float
    pvalue: Optional[float] = None
    is_drifted: bool
    threshold: float
    baseline_window: str
    eval_window: str
    baseline_window_start: datetime
    baseline_window_end: datetime
    eval_window_start: datetime
    eval_window_end: datetime
    percentile_rank: Optional[float] = None
    affected_categories: Optional[List[str]] = None
    computed_at: datetime
    tenant_id: str
    region: str

class DriftAlert(BaseModel):
    """Alert for drifted feature"""
    feature_id: str
    method: str
    score: float
    threshold: float
    percentile_rank: float
    alert_channel: str
    created_at: datetime
    severity: str = Field(default="warning", description="info | warning | critical")

class FeatureMetadata(BaseModel):
    """Feature metadata from catalog"""
    feature_id: str
    name: str
    owner: str
    is_core: bool
    drift_config: dict

class DriftHealthReport(BaseModel):
    """Health report for a feature"""
    feature_id: str
    last_drift_check: Optional[datetime] = None
    active_drifts: int
    drift_trend: Optional[str] = None  # stable | degrading | improving
    last_alert_sent: Optional[datetime] = None
    alert_count_24h: int
