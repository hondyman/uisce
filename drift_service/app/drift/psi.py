"""Population Stability Index (PSI) for categorical feature drift detection"""

import numpy as np
from typing import Tuple

def compute_psi_drift(baseline: np.ndarray, recent: np.ndarray, bins: int = 10) -> Tuple[float, None]:
    """
    Compute Population Stability Index between baseline and recent distributions.
    
    PSI measures distributional shift for categorical or binned continuous features.
    
    Returns:
        (psi_statistic, None)
    
    PSI = SUM[(baseline_pct - recent_pct) * LN(baseline_pct / recent_pct)]
    
    Interpretation:
    - PSI < 0.10: No significant drift (typical variation)
    - PSI 0.10-0.25: Significant drift (investigate)
    - PSI > 0.25: Major distributional shift (action required)
    """
    # Handle edge cases
    if len(baseline) == 0 or len(recent) == 0:
        raise ValueError("Baseline and recent must have non-zero length")
    
    # Remove NaN values
    baseline_clean = baseline[~np.isnan(baseline)]
    recent_clean = recent[~np.isnan(recent)]
    
    if len(baseline_clean) == 0 or len(recent_clean) == 0:
        raise ValueError("No valid (non-NaN) values to compare")
    
    # Create histograms
    baseline_hist, bin_edges = np.histogram(baseline_clean, bins=bins)
    recent_hist, _ = np.histogram(recent_clean, bins=bin_edges)
    
    # Convert to percentages (add small epsilon to avoid log(0))
    epsilon = 1e-10
    baseline_pct = (baseline_hist + epsilon) / (baseline_hist.sum() + epsilon * len(baseline_hist))
    recent_pct = (recent_hist + epsilon) / (recent_hist.sum() + epsilon * len(recent_hist))
    
    # Compute PSI
    psi = np.sum((baseline_pct - recent_pct) * np.log(baseline_pct / recent_pct))
    
    return float(psi), None

def compute_psi_categorical(baseline: np.ndarray, recent: np.ndarray) -> Tuple[float, None]:
    """
    Compute PSI for categorical (non-numeric) features.
    
    Uses category frequencies instead of binning.
    """
    if len(baseline) == 0 or len(recent) == 0:
        raise ValueError("Baseline and recent must have non-zero length")
    
    # Get unique categories from both
    all_categories = set(baseline) | set(recent)
    
    epsilon = 1e-10
    psi = 0.0
    
    for category in all_categories:
        baseline_count = (baseline == category).sum() + epsilon
        recent_count = (recent == category).sum() + epsilon
        
        baseline_pct = baseline_count / (len(baseline) + epsilon)
        recent_pct = recent_count / (len(recent) + epsilon)
        
        psi += (baseline_pct - recent_pct) * np.log(baseline_pct / recent_pct)
    
    return float(psi), None
