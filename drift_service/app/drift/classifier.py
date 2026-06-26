"""Classifier-based drift detection (advanced method)"""

import numpy as np
from sklearn.ensemble import RandomForestClassifier
from sklearn.preprocessing import StandardScaler
from sklearn.metrics import roc_auc_score
from typing import Tuple
import logging

logger = logging.getLogger(__name__)

def compute_classifier_drift(baseline: np.ndarray, recent: np.ndarray, n_estimators: int = 50) -> Tuple[float, None]:
    """
    Compute drift using a binary classifier's ability to distinguish between
    baseline and recent distributions.
    
    The intuition: if a classifier can significantly separate baseline from recent,
    the distributions have drifted.
    
    Returns:
        (auc_score, None)
    
    Interpretation:
    - AUC = 0.5: Indistinguishable (no drift)
    - AUC 0.5-0.7: Weak drift signal
    - AUC 0.7-0.9: Significant drift
    - AUC > 0.9: Severe drift
    """
    if len(baseline) == 0 or len(recent) == 0:
        raise ValueError("Baseline and recent must have non-zero length")
    
    # Remove NaN
    baseline_clean = baseline[~np.isnan(baseline)]
    recent_clean = recent[~np.isnan(recent)]
    
    if len(baseline_clean) == 0 or len(recent_clean) == 0:
        raise ValueError("No valid values after NaN removal")
    
    # Combine data
    X = np.concatenate([baseline_clean, recent_clean]).reshape(-1, 1)
    y = np.concatenate([
        np.zeros(len(baseline_clean)),
        np.ones(len(recent_clean))
    ])
    
    # Standardize
    scaler = StandardScaler()
    X_scaled = scaler.fit_transform(X)
    
    # Train classifier
    try:
        clf = RandomForestClassifier(n_estimators=n_estimators, random_state=42, n_jobs=-1)
        clf.fit(X_scaled, y)
        
        # Get probability predictions
        probs = clf.predict_proba(X_scaled)[:, 1]
        
        # Compute AUC
        auc = roc_auc_score(y, probs)
        return float(auc), None
    except Exception as e:
        logger.error(f"Classifier drift computation failed: {str(e)}")
        raise

def compute_mmd_drift(baseline: np.ndarray, recent: np.ndarray, kernel: str = "rbf", sigma: float = 1.0) -> Tuple[float, None]:
    """
    Compute Maximum Mean Discrepancy (MMD) for multivariate drift detection.
    
    MMD is a robust, kernel-based distance metric between distributions.
    
    Returns:
        (mmd_statistic, None)
    
    Interpretation:
    - MMD ≈ 0: Distributions overlap significantly (no drift)
    - MMD > 0.1: Noticeable distributional difference
    - MMD > 0.3: Significant drift
    """
    if len(baseline) == 0 or len(recent) == 0:
        raise ValueError("Baseline and recent must have non-zero length")
    
    # Ensure 2D
    if baseline.ndim == 1:
        baseline = baseline.reshape(-1, 1)
    if recent.ndim == 1:
        recent = recent.reshape(-1, 1)
    
    # Kernel function
    if kernel == "rbf":
        def K(x, y, sigma=sigma):
            return np.exp(-np.sum((x - y) ** 2) / (2 * sigma ** 2))
    elif kernel == "linear":
        def K(x, y, sigma=None):
            return np.dot(x, y)
    else:
        raise ValueError(f"Unknown kernel: {kernel}")
    
    # Compute MMD
    n_baseline = len(baseline)
    n_recent = len(recent)
    
    # Intra-baseline kernel
    K_baseline = 0.0
    for i in range(n_baseline):
        for j in range(n_baseline):
            K_baseline += K(baseline[i], baseline[j])
    K_baseline /= (n_baseline ** 2)
    
    # Intra-recent kernel
    K_recent = 0.0
    for i in range(n_recent):
        for j in range(n_recent):
            K_recent += K(recent[i], recent[j])
    K_recent /= (n_recent ** 2)
    
    # Inter-sample kernel
    K_cross = 0.0
    for i in range(n_baseline):
        for j in range(n_recent):
            K_cross += K(baseline[i], recent[j])
    K_cross /= (n_baseline * n_recent)
    
    # MMD
    mmd = np.sqrt(K_baseline + K_recent - 2 * K_cross)
    
    return float(mmd), None
