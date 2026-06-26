"""Drift detection orchestration runner"""

import asyncio
import logging
from datetime import datetime, timedelta
import numpy as np

from app.models import DriftRequest, DriftResult
from app.drift.ks import compute_ks_drift, estimate_percentile_rank
from app.drift.psi import compute_psi_drift, compute_psi_categorical
from app.drift.chi2 import compute_chi2_drift, compute_chi2_binned
from app.drift.classifier import compute_classifier_drift
from app.storage.iceberg import load_feature_values
from app.storage.postgres import store_drift_metrics, get_feature_drift_config
from app.metrics.prometheus import drift_score_gauge, drift_alerts_counter
from app.config import settings

logger = logging.getLogger(__name__)

async def run_drift_detection(req: DriftRequest) -> DriftResult:
    """
    Main orchestration function for drift detection.
    
    1. Load baseline and recent feature values
    2. Run specified algorithm
    3. Persist results to PostgreSQL
    4. Emit Prometheus metrics
    5. Return result
    """
    
    try:
        # Load feature configuration
        config = await get_feature_drift_config(req.feature_id)
        
        # Determine threshold
        threshold = req.threshold or get_default_threshold(req.method)
        
        # Calculate time windows
        now = datetime.utcnow()
        baseline_start, baseline_end = parse_window(req.baseline_window, now)
        eval_start, eval_end = parse_window(req.eval_window, now)
        
        logger.info(f"Drift detection starting for {req.feature_id} (method={req.method})")
        
        # Load data
        baseline_values = await load_feature_values(
            req.feature_id,
            baseline_start,
            baseline_end,
            req.tenant_id,
            req.region
        )
        recent_values = await load_feature_values(
            req.feature_id,
            eval_start,
            eval_end,
            req.tenant_id,
            req.region
        )
        
        if len(baseline_values) == 0 or len(recent_values) == 0:
            logger.warning(f"Insufficient data for {req.feature_id}: baseline={len(baseline_values)}, recent={len(recent_values)}")
            raise ValueError("Insufficient data to compute drift")
        
        # Compute drift
        if req.method == "ks":
            score, pvalue = await asyncio.to_thread(compute_ks_drift, baseline_values, recent_values)
        elif req.method == "psi":
            # Heuristic: use categorical if low cardinality, else binned
            if len(np.unique(baseline_values)) < 50:
                score, pvalue = await asyncio.to_thread(compute_psi_categorical, baseline_values, recent_values)
            else:
                score, pvalue = await asyncio.to_thread(compute_psi_drift, baseline_values, recent_values)
        elif req.method == "chi2":
            if len(np.unique(baseline_values)) < 50:
                score, pvalue = await asyncio.to_thread(compute_chi2_drift, baseline_values, recent_values)
            else:
                score, pvalue = await asyncio.to_thread(compute_chi2_binned, baseline_values, recent_values)
        elif req.method == "classifier":
            score, pvalue = await asyncio.to_thread(compute_classifier_drift, baseline_values, recent_values)
        else:
            raise ValueError(f"Unknown drift method: {req.method}")
        
        # Determine if drifted
        is_drifted = score > threshold
        
        # Estimate percentile rank
        percentile_rank = None
        if is_drifted and req.method == "ks":
            # Get historical KS scores for percentile calculation
            percentile_rank = await estimate_drift_percentile(req.feature_id, score)
        
        # Build result
        result = DriftResult(
            feature_id=req.feature_id,
            method=req.method,
            score=score,
            pvalue=pvalue,
            is_drifted=is_drifted,
            threshold=threshold,
            baseline_window=req.baseline_window,
            eval_window=req.eval_window,
            baseline_window_start=baseline_start,
            baseline_window_end=baseline_end,
            eval_window_start=eval_start,
            eval_window_end=eval_end,
            percentile_rank=percentile_rank,
            computed_at=datetime.utcnow(),
            tenant_id=req.tenant_id,
            region=req.region
        )
        
        # Persist to PostgreSQL
        await store_drift_metrics(result)
        
        # Emit metrics
        drift_score_gauge.labels(
            feature_id=req.feature_id,
            method=req.method,
            tenant_id=req.tenant_id,
            region=req.region
        ).set(score)
        
        if is_drifted:
            drift_alerts_counter.labels(
                feature_id=req.feature_id,
                method=req.method,
                tenant_id=req.tenant_id,
                region=req.region
            ).inc()
        
        logger.info(
            f"Drift detection complete for {req.feature_id}: "
            f"score={score:.4f}, threshold={threshold:.4f}, "
            f"drifted={is_drifted}, pvalue={pvalue}"
        )
        
        return result
        
    except Exception as e:
        logger.error(f"Drift detection failed for {req.feature_id}: {str(e)}", exc_info=True)
        raise

def get_default_threshold(method: str) -> float:
    """Get default threshold for detection method"""
    thresholds = {
        "ks": settings.KS_THRESHOLD,
        "psi": settings.PSI_THRESHOLD,
        "chi2": settings.CHI2_THRESHOLD,
        "classifier": settings.CLASSIFIER_THRESHOLD
    }
    return thresholds.get(method, 0.05)

def parse_window(window_str: str, reference_time: datetime) -> tuple:
    """
    Parse window string (e.g., "30d", "7d", "24h", "1h") into start/end times.
    
    Returns (start_time, end_time)
    """
    window_str = window_str.strip().lower()
    
    if window_str.endswith("d"):
        days = int(window_str[:-1])
        duration = timedelta(days=days)
    elif window_str.endswith("h"):
        hours = int(window_str[:-1])
        duration = timedelta(hours=hours)
    elif window_str.endswith("m"):
        minutes = int(window_str[:-1])
        duration = timedelta(minutes=minutes)
    else:
        raise ValueError(f"Invalid window format: {window_str}")
    
    end_time = reference_time
    start_time = end_time - duration
    
    return start_time, end_time

async def estimate_drift_percentile(feature_id: str, current_score: float) -> float:
    """
    Estimate percentile rank of current drift score relative to historical drifts.
    
    Returns [0, 100]:
    - 0-50: Typical
    - 50-90: Elevated
    - 90+: Extreme
    """
    from app.storage.postgres import get_historical_drift_scores
    
    try:
        historical_scores = await get_historical_drift_scores(feature_id, days=90)
        if not historical_scores:
            return 50.0
        
        sorted_scores = sorted(historical_scores)
        percentile = (sum(1 for s in sorted_scores if s <= current_score) / len(sorted_scores)) * 100
        return float(percentile)
    except Exception as e:
        logger.warning(f"Failed to estimate percentile for {feature_id}: {str(e)}")
        return 50.0
