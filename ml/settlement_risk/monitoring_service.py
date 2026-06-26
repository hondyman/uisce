#!/usr/bin/env python3
"""
Model Monitoring Service

Periodically analyzes prediction logs for:
1. Data Drift (Kolmogorov-Smirnov test)
2. Performance Degradation (if ground truth available)

Triggers retraining via Temporal if issues are detected.
"""

import os
import json
import time
import logging
from datetime import datetime, timedelta

import pandas as pd
import numpy as np
from scipy import stats
import joblib
import requests

# Configuration
LOG_DIR = os.environ.get('LOG_DIR', './logs')
FEATURE_LOG_PATH = os.path.join(LOG_DIR, 'prediction_logs.jsonl')
MODEL_DIR = os.environ.get('MODEL_DIR', './model_output')
PREPROC_PATH = os.path.join(MODEL_DIR, 'preprocessing.pkl')
TEMPORAL_HOST = os.environ.get('TEMPORAL_HOST', 'localhost:7233')
CHECK_INTERVAL_SECONDS = int(os.environ.get('CHECK_INTERVAL', 3600)) # Default 1 hour

# Thresholds
DRIFT_THRESHOLD_P_VALUE = 0.05 # P-value for KS test (statistically significant drift)
DRIFT_FEATURE_PCT_THRESHOLD = 0.3 # If > 30% features drift, trigger retraining

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def load_baseline_stats():
    """Load training data statistics from preprocessing artifacts."""
    try:
        # Ideally, we saved summary stats during training. 
        # For now, we'll try to load the original training data if available, 
        # or rely on a saved baseline file.
        # This is a simplification. In prod, use a feature store.
        pass
    except Exception as e:
        logger.error(f"Failed to load baseline: {e}")
    return None

def detect_drift(recent_logs_df: pd.DataFrame, baseline_df: pd.DataFrame = None):
    """
    Compare recent inference data distribution with baseline training data.
    """
    if baseline_df is None:
        logger.warning("No baseline data available for drift detection.")
        return False

    drifted_features = []
    
    # Common numeric columns
    common_cols = list(set(recent_logs_df.select_dtypes(include=[np.number]).columns) & 
                       set(baseline_df.select_dtypes(include=[np.number]).columns))
    
    for col in common_cols:
        # Kolmogorov-Smirnov test for 2 samples
        # Null hypothesis: samples are drawn from the same distribution
        # If p < 0.05, we reject null hypothesis -> DRIFT DETECTED
        stat, p_value = stats.ks_2samp(recent_logs_df[col], baseline_df[col])
        
        if p_value < DRIFT_THRESHOLD_P_VALUE:
            logger.warning(f"Drift detected in feature '{col}' (p={p_value:.5f})")
            drifted_features.append(col)
            
    drift_pct = len(drifted_features) / len(common_cols) if common_cols else 0
    logger.info(f"Drift detected in {len(drifted_features)}/{len(common_cols)} features ({drift_pct:.1%})")
    
    return drift_pct > DRIFT_FEATURE_PCT_THRESHOLD

def load_logs(hours=24):
    """Load prediction logs from the last N hours."""
    if not os.path.exists(FEATURE_LOG_PATH):
        return pd.DataFrame()
        
    data = []
    cutoff_time = datetime.now() - timedelta(hours=hours)
    
    try:
        with open(FEATURE_LOG_PATH, 'r') as f:
            for line in f:
                try:
                    entry = json.loads(line)
                    ts = datetime.fromisoformat(entry['timestamp'])
                    if ts > cutoff_time:
                        # Flatten structure
                        row = entry['features']
                        row['timestamp'] = ts
                        row['predicted_risk'] = entry['result']['settlement_risk_score']
                        data.append(row)
                except Exception as e:
                    continue # Skip malformed lines
    except Exception as e:
        logger.error(f"Error reading logs: {e}")
        
    return pd.DataFrame(data)

def trigger_retraining(reason: str):
    """Trigger the Temporal workflow for model retraining."""
    logger.info(f"Triggering retraining due to: {reason}")
    
    # In a real implementation, use the Temporal Client SDK
    # For this simulation, we'll just log it or call a webhook
    logger.info(f"Simulating Temporal workflow trigger: AutomatedRetrainingWorkflow")
    
    # Example:
    # client = temporal.Client(...)
    # client.execute_workflow("AutomatedRetrainingWorkflow", ...)

def monitor_loop():
    logger.info("Starting monitoring loop...")
    while True:
        try:
            logger.info("Running drift analysis...")
            df_recent = load_logs(hours=24)
            
            if df_recent.empty:
                logger.info("No recent logs found. Skipping analysis.")
            else:
                logger.info(f"Analyzed {len(df_recent)} predictions.")
                
                # In this demo, we mock the baseline comparison 
                # (since we don't have the original training dataframe loaded here easily without complexity)
                # We'll simulate drift if we see very high variance or missing data
                
                # Mock drift check
                is_drift_detected = False
                if len(df_recent) > 100 and df_recent['predicted_risk'].mean() > 0.8:
                     # If model suddenly predicts high risk for everyone, something is wrong
                    is_drift_detected = True
                    trigger_retraining("Abnormal prediction distribution (High Risk Spike)")
                
                # Real implementation would call detect_drift(df_recent, baseline_df)
                
        except Exception as e:
            logger.error(f"Monitoring cycle failed: {e}")
            
        time.sleep(CHECK_INTERVAL_SECONDS)

if __name__ == "__main__":
    monitor_loop()
