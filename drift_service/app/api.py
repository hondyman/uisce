"""FastAPI routes for Drift Detection Service"""

from fastapi import APIRouter, HTTPException, BackgroundTasks
from typing import List
import logging

from app.models import DriftRequest, DriftResult, DriftHealthReport
from app.drift.runner import run_drift_detection
from app.storage.postgres import get_feature_metadata, get_feature_health

logger = logging.getLogger(__name__)
router = APIRouter()

@router.post("/drift/detect", response_model=DriftResult)
async def detect_drift(req: DriftRequest, background_tasks: BackgroundTasks):
    """
    Compute drift for a feature using specified method.
    
    Methods:
    - ks: Kolmogorov-Smirnov test (continuous)
    - psi: Population Stability Index (categorical or binned)
    - chi2: Chi-square test (categorical)
    - classifier: Classifier-based drift detection (advanced)
    """
    try:
        result = await run_drift_detection(req)
        
        # Schedule alerting if drifted
        if result.is_drifted:
            background_tasks.add_task(alert_on_drift, result)
        
        return result
    except Exception as e:
        logger.error(f"Drift detection failed for {req.feature_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/drift/batch")
async def detect_drift_batch(requests: List[DriftRequest]):
    """Compute drift for multiple features (batched)"""
    results = []
    for req in requests:
        try:
            result = await run_drift_detection(req)
            results.append(result)
        except Exception as e:
            logger.error(f"Batch drift detection failed for {req.feature_id}: {str(e)}")
            results.append({"feature_id": req.feature_id, "error": str(e)})
    return {"results": results, "total": len(results)}

@router.get("/drift/health/{feature_id}", response_model=DriftHealthReport)
async def get_drift_health(feature_id: str, tenant_id: str = "default", region: str = "us-east-1"):
    """Get health report for a feature (drift history, active incidents)"""
    try:
        health = await get_feature_health(feature_id, tenant_id, region)
        return health
    except Exception as e:
        logger.error(f"Failed to get health for {feature_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/drift/active")
async def get_active_drifts(tenant_id: str = "default", region: str = "us-east-1"):
    """List all currently active drifts (is_drifted=true in last 7 days)"""
    try:
        # Query active_drifts materialized view
        from app.storage.postgres import get_active_drifts
        drifts = await get_active_drifts(tenant_id, region)
        return {"count": len(drifts), "drifts": drifts}
    except Exception as e:
        logger.error(f"Failed to get active drifts: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/drift/metrics/{feature_id}")
async def get_drift_metrics(feature_id: str, days: int = 30):
    """Get drift metrics history for a feature (for graphing)"""
    try:
        from app.storage.postgres import get_drift_metrics_history
        metrics = await get_drift_metrics_history(feature_id, days)
        return {"feature_id": feature_id, "metrics": metrics}
    except Exception as e:
        logger.error(f"Failed to get metrics for {feature_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/features/metadata/{feature_id}")
async def get_feature_info(feature_id: str):
    """Get feature metadata (owner, drift config, is_core)"""
    try:
        metadata = await get_feature_metadata(feature_id)
        return metadata
    except Exception as e:
        logger.error(f"Failed to get metadata for {feature_id}: {str(e)}")
        raise HTTPException(status_code=404, detail=f"Feature not found: {feature_id}")

async def alert_on_drift(result):
    """Send alert for drifted feature (background task)"""
    from app.alerts.notify import send_alert
    try:
        await send_alert(result)
        logger.info(f"Alert sent for drifted feature {result.feature_id}")
    except Exception as e:
        logger.error(f"Failed to send alert for {result.feature_id}: {str(e)}")
