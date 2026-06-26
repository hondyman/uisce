"""Prometheus metrics instrumentation"""

from prometheus_client import Counter, Gauge, Histogram, generate_latest, CONTENT_TYPE_LATEST
from fastapi.responses import Response
import logging

logger = logging.getLogger(__name__)

# Drift detection metrics
drift_score_gauge = Gauge(
    'drift_score',
    'Drift detection score (method-dependent)',
    ['feature_id', 'method', 'tenant_id', 'region']
)

drift_alerts_counter = Counter(
    'drift_alerts_total',
    'Total drift alerts sent',
    ['feature_id', 'method', 'tenant_id', 'region']
)

drift_detection_duration = Histogram(
    'drift_detection_duration_seconds',
    'Time to compute drift for a feature',
    ['method', 'tenant_id'],
    buckets=[0.1, 0.5, 1.0, 2.0, 5.0, 10.0]
)

feature_values_loaded = Counter(
    'feature_values_loaded_total',
    'Total feature values loaded',
    ['feature_id', 'tenant_id', 'region']
)

drift_computation_errors = Counter(
    'drift_computation_errors_total',
    'Total drift computation errors',
    ['method', 'error_type']
)

drifted_features_active = Gauge(
    'drifted_features_active',
    'Number of currently active drifted features',
    ['tenant_id', 'region']
)

alert_failures = Counter(
    'alert_failures_total',
    'Total alert sending failures',
    ['channel', 'feature_id']
)

def setup_metrics():
    """Initialize metrics (called at startup)"""
    logger.info("Prometheus metrics initialized")

async def metrics_endpoint():
    """Prometheus /metrics endpoint"""
    return Response(
        content=generate_latest(),
        media_type=CONTENT_TYPE_LATEST
    )
