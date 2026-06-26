"""PostgreSQL storage layer for drift metrics"""

import psycopg2
from psycopg2.pool import SimpleConnectionPool
import logging
from datetime import datetime
from typing import List, Optional

from app.models import DriftResult, DriftHealthReport, FeatureMetadata
from app.config import settings

logger = logging.getLogger(__name__)

# Connection pool
_pool = None

def get_connection_pool():
    global _pool
    if _pool is None:
        _pool = SimpleConnectionPool(
            1, 20,
            host=settings.POSTGRES_HOST,
            port=settings.POSTGRES_PORT,
            user=settings.POSTGRES_USER,
            password=settings.POSTGRES_PASSWORD,
            database=settings.POSTGRES_DB
        )
    return _pool

async def store_drift_metrics(result: DriftResult) -> None:
    """Persist drift detection result to feature_drift_metrics table"""
    pool = get_connection_pool()
    conn = pool.getconn()
    
    try:
        cur = conn.cursor()
        cur.execute("""
            INSERT INTO feature_drift_metrics (
                feature_id,
                method,
                score,
                pvalue,
                is_drifted,
                threshold,
                baseline_window_start,
                baseline_window_end,
                eval_window_start,
                eval_window_end,
                percentile_rank,
                alert_sent,
                computed_at,
                recorded_at,
                tenant_id,
                region
            ) VALUES (
                %s, %s, %s, %s, %s, %s,
                %s, %s, %s, %s, %s, %s,
                %s, %s, %s, %s
            )
        """, (
            result.feature_id,
            result.method,
            result.score,
            result.pvalue,
            result.is_drifted,
            result.threshold,
            result.baseline_window_start,
            result.baseline_window_end,
            result.eval_window_start,
            result.eval_window_end,
            result.percentile_rank,
            False,  # alert_sent: set to False, will update after alerting
            result.computed_at,
            datetime.utcnow(),
            result.tenant_id,
            result.region
        ))
        conn.commit()
        logger.info(f"Stored drift metrics for {result.feature_id}")
    except Exception as e:
        conn.rollback()
        logger.error(f"Failed to store drift metrics: {str(e)}")
        raise
    finally:
        pool.putconn(conn)

async def get_feature_drift_config(feature_id: str) -> dict:
    """Load feature drift configuration from feature_catalog"""
    pool = get_connection_pool()
    conn = pool.getconn()
    
    try:
        cur = conn.cursor()
        cur.execute("""
            SELECT properties->'drift_config' as config
            FROM feature_catalog
            WHERE feature_id = %s
        """, (feature_id,))
        
        row = cur.fetchone()
        if row and row[0]:
            return row[0]
        else:
            return {}
    except Exception as e:
        logger.error(f"Failed to load drift config for {feature_id}: {str(e)}")
        raise
    finally:
        pool.putconn(conn)

async def get_feature_metadata(feature_id: str) -> FeatureMetadata:
    """Load feature metadata"""
    pool = get_connection_pool()
    conn = pool.getconn()
    
    try:
        cur = conn.cursor()
        cur.execute("""
            SELECT feature_id, name, owner, is_core, properties->'drift_config' as drift_config
            FROM feature_catalog
            WHERE feature_id = %s
        """, (feature_id,))
        
        row = cur.fetchone()
        if not row:
            raise ValueError(f"Feature not found: {feature_id}")
        
        return FeatureMetadata(
            feature_id=row[0],
            name=row[1],
            owner=row[2],
            is_core=row[3],
            drift_config=row[4] or {}
        )
    except Exception as e:
        logger.error(f"Failed to load metadata for {feature_id}: {str(e)}")
        raise
    finally:
        pool.putconn(conn)

async def get_feature_health(feature_id: str, tenant_id: str, region: str) -> DriftHealthReport:
    """Get comprehensive health report for a feature"""
    pool = get_connection_pool()
    conn = pool.getconn()
    
    try:
        cur = conn.cursor()
        
        # Get last drift check, active drifts, alerts
        cur.execute("""
            SELECT
                MAX(recorded_at) as last_check,
                SUM(CASE WHEN is_drifted THEN 1 ELSE 0 END) as active_drifts,
                SUM(CASE WHEN is_drifted AND alert_sent THEN 1 ELSE 0 END) as alerts_24h
            FROM feature_drift_metrics
            WHERE feature_id = %s
                AND tenant_id = %s
                AND region = %s
                AND recorded_at >= NOW() - INTERVAL '7 days'
        """, (feature_id, tenant_id, region))
        
        row = cur.fetchone()
        
        return DriftHealthReport(
            feature_id=feature_id,
            last_drift_check=row[0] if row else None,
            active_drifts=row[1] or 0 if row else 0,
            alert_count_24h=row[2] or 0 if row else 0
        )
    except Exception as e:
        logger.error(f"Failed to get health for {feature_id}: {str(e)}")
        raise
    finally:
        pool.putconn(conn)

async def get_active_drifts(tenant_id: str, region: str) -> List[dict]:
    """Get all currently active drifts (materialized view query)"""
    pool = get_connection_pool()
    conn = pool.getconn()
    
    try:
        cur = conn.cursor()
        cur.execute("""
            SELECT feature_id, method, score, pvalue, is_drifted, percentile_rank, eval_window_end
            FROM active_drifts
            WHERE recorded_at >= NOW() - INTERVAL '7 days'
            ORDER BY percentile_rank DESC NULLS LAST
            LIMIT 100
        """)
        
        drifts = []
        for row in cur.fetchall():
            drifts.append({
                "feature_id": row[0],
                "method": row[1],
                "score": row[2],
                "pvalue": row[3],
                "is_drifted": row[4],
                "percentile_rank": row[5],
                "eval_window_end": row[6]
            })
        
        return drifts
    except Exception as e:
        logger.error(f"Failed to get active drifts: {str(e)}")
        raise
    finally:
        pool.putconn(conn)

async def get_drift_metrics_history(feature_id: str, days: int = 30) -> List[dict]:
    """Get drift metrics history for graphing"""
    pool = get_connection_pool()
    conn = pool.getconn()
    
    try:
        cur = conn.cursor()
        cur.execute("""
            SELECT recorded_at, method, score, is_drifted, threshold
            FROM feature_drift_metrics
            WHERE feature_id = %s
                AND recorded_at >= NOW() - INTERVAL %s
            ORDER BY recorded_at ASC
        """, (feature_id, f"{days} days"))
        
        metrics = []
        for row in cur.fetchall():
            metrics.append({
                "timestamp": row[0].isoformat(),
                "method": row[1],
                "score": row[2],
                "is_drifted": row[3],
                "threshold": row[4]
            })
        
        return metrics
    except Exception as e:
        logger.error(f"Failed to get metrics history for {feature_id}: {str(e)}")
        raise
    finally:
        pool.putconn(conn)

async def get_historical_drift_scores(feature_id: str, days: int = 90) -> List[float]:
    """Get historical drift scores for percentile calculation"""
    pool = get_connection_pool()
    conn = pool.getconn()
    
    try:
        cur = conn.cursor()
        cur.execute("""
            SELECT score FROM feature_drift_metrics
            WHERE feature_id = %s
                AND recorded_at >= NOW() - INTERVAL %s
            ORDER BY recorded_at DESC
        """, (feature_id, f"{days} days"))
        
        scores = [row[0] for row in cur.fetchall()]
        return scores
    except Exception as e:
        logger.error(f"Failed to get historical scores for {feature_id}: {str(e)}")
        raise
    finally:
        pool.putconn(conn)

async def mark_alert_sent(drift_id: str) -> None:
    """Mark that alert was sent for a drift metric"""
    pool = get_connection_pool()
    conn = pool.getconn()
    
    try:
        cur = conn.cursor()
        cur.execute("""
            UPDATE feature_drift_metrics
            SET alert_sent = TRUE
            WHERE drift_id = %s
        """, (drift_id,))
        conn.commit()
    except Exception as e:
        conn.rollback()
        logger.error(f"Failed to mark alert sent: {str(e)}")
        raise
    finally:
        pool.putconn(conn)
