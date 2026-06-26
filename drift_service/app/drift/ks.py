"""Kolmogorov-Smirnov test for continuous feature drift detection"""

import numpy as np
from scipy.stats import ks_2samp
from typing import Tuple

def compute_ks_drift(baseline: np.ndarray, recent: np.ndarray) -> Tuple[float, float]:
    """
    Compute KS test statistic between baseline and recent distributions.
    
    Returns:
        (statistic, p_value)
    
    The KS statistic measures the maximum distance between two cumulative 
    distribution functions. Range: [0, 1].
    
    Interpretation:
    - statistic < 0.05: No significant drift
    - statistic 0.05-0.10: Marginal drift
    - statistic > 0.10: Significant drift
    """
    # Handle edge cases
    if len(baseline) == 0 or len(recent) == 0:
        raise ValueError("Baseline and recent must have non-zero length")
    
    # Remove NaN values
    baseline_clean = baseline[~np.isnan(baseline)]
    recent_clean = recent[~np.isnan(recent)]
    
    if len(baseline_clean) == 0 or len(recent_clean) == 0:
        raise ValueError("No valid (non-NaN) values to compare")
    
    statistic, pvalue = ks_2samp(baseline_clean, recent_clean)
    return float(statistic), float(pvalue)

def estimate_percentile_rank(statistic: float, baseline_stats: list) -> float:
    """
    Estimate how extreme this drift is relative to historical drifts.
    
    Returns percentile [0, 100]:
    - 0-50: Typical variation
    - 50-90: Elevated drift
    - 90+: Extreme drift
    """
    if not baseline_stats:
        return 50.0
    
    sorted_stats = sorted(baseline_stats)
    percentile = (sum(1 for s in sorted_stats if s <= statistic) / len(sorted_stats)) * 100
    return float(percentile)
