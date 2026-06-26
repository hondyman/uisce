"""Alert notification service"""

import logging
import json
import aiohttp
from datetime import datetime
from typing import Optional

from app.models import DriftResult
from app.config import settings
from app.metrics.prometheus import alert_failures

logger = logging.getLogger(__name__)

async def send_alert(result: DriftResult) -> None:
    """
    Send alert for drifted feature.
    Supports: email, webhook, PagerDuty.
    """
    if not result.is_drifted:
        return
    
    try:
        # Format message
        message = format_alert_message(result)
        
        # Send to configured channels
        await send_webhook_alert(result, message)
        await send_email_alert(result, message)
        await send_pagerduty_alert(result, message)
        
        logger.info(f"Alert sent for {result.feature_id}")
    except Exception as e:
        logger.error(f"Failed to send alert for {result.feature_id}: {str(e)}")
        drift_computation_errors.labels(channel="alert", feature_id=result.feature_id).inc()
        raise

def format_alert_message(result: DriftResult) -> str:
    """Format alert message"""
    return f"""
🚨 **FEATURE DRIFT ALERT**

**Feature**: {result.feature_id}
**Method**: {result.method}
**Score**: {result.score:.4f}
**Threshold**: {result.threshold:.4f}
**Is Drifted**: {result.is_drifted}
**P-Value**: {result.pvalue}
**Percentile Rank**: {result.percentile_rank}
**Baseline Window**: {result.baseline_window}
**Eval Window**: {result.eval_window}
**Tenant**: {result.tenant_id}
**Region**: {result.region}
**Timestamp**: {result.computed_at.isoformat()}

📊 Interpretation:
- If score > threshold, feature distribution has shifted beyond acceptable limits.
- Percentile rank indicates how extreme this drift is historically.
- Investigate root cause, retrain model, or update thresholds.
    """

async def send_webhook_alert(result: DriftResult, message: str) -> None:
    """Send alert via webhook"""
    if not settings.ALERT_WEBHOOK_URL:
        return
    
    try:
        payload = {
            "feature_id": result.feature_id,
            "method": result.method,
            "score": result.score,
            "threshold": result.threshold,
            "is_drifted": result.is_drifted,
            "pvalue": result.pvalue,
            "percentile_rank": result.percentile_rank,
            "computed_at": result.computed_at.isoformat(),
            "tenant_id": result.tenant_id,
            "region": result.region,
            "message": message
        }
        
        async with aiohttp.ClientSession() as session:
            async with session.post(
                settings.ALERT_WEBHOOK_URL,
                json=payload,
                timeout=aiohttp.ClientTimeout(total=10)
            ) as resp:
                if resp.status != 200:
                    logger.warning(f"Webhook alert failed with status {resp.status}")
                    alert_failures.labels(channel="webhook", feature_id=result.feature_id).inc()
    except Exception as e:
        logger.error(f"Webhook alert failed: {str(e)}")
        alert_failures.labels(channel="webhook", feature_id=result.feature_id).inc()

async def send_email_alert(result: DriftResult, message: str) -> None:
    """Send alert via email"""
    if not settings.ALERT_EMAIL_RECIPIENTS:
        return
    
    try:
        # Would integrate with mail service (e.g., SendGrid, AWS SES)
        recipients = settings.ALERT_EMAIL_RECIPIENTS.split(",")
        
        logger.info(f"Email alert would be sent to {recipients} for {result.feature_id}")
        # TODO: Implement actual email sending
    except Exception as e:
        logger.error(f"Email alert failed: {str(e)}")
        alert_failures.labels(channel="email", feature_id=result.feature_id).inc()

async def send_pagerduty_alert(result: DriftResult, message: str) -> None:
    """Send alert to PagerDuty"""
    if not settings.ALERT_PAGERDUTY_KEY:
        return
    
    try:
        severity = "critical" if result.percentile_rank and result.percentile_rank > 90 else "warning"
        
        payload = {
            "routing_key": settings.ALERT_PAGERDUTY_KEY,
            "event_action": "trigger",
            "dedup_key": f"drift-{result.feature_id}-{result.computed_at.isoformat()}",
            "payload": {
                "summary": f"Feature drift detected: {result.feature_id}",
                "severity": severity,
                "source": "drift-detection-service",
                "custom_details": {
                    "feature_id": result.feature_id,
                    "score": result.score,
                    "threshold": result.threshold,
                    "method": result.method,
                    "percentile_rank": result.percentile_rank
                }
            }
        }
        
        async with aiohttp.ClientSession() as session:
            async with session.post(
                "https://events.pagerduty.com/v2/enqueue",
                json=payload,
                timeout=aiohttp.ClientTimeout(total=10)
            ) as resp:
                if resp.status != 202:
                    logger.warning(f"PagerDuty alert failed with status {resp.status}")
                    alert_failures.labels(channel="pagerduty", feature_id=result.feature_id).inc()
    except Exception as e:
        logger.error(f"PagerDuty alert failed: {str(e)}")
        alert_failures.labels(channel="pagerduty", feature_id=result.feature_id).inc()
