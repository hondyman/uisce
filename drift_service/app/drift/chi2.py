"""Chi-square test for categorical feature drift detection"""

import numpy as np
from scipy.stats import chisquare
from typing import Tuple

def compute_chi2_drift(baseline: np.ndarray, recent: np.ndarray) -> Tuple[float, float]:
    """
    Compute Chi-square test statistic for categorical distributions.
    
    Returns:
        (chi2_statistic, p_value)
    
    Chi-square tests if observed (recent) distribution matches expected (baseline).
    
    Interpretation:
    - p_value > 0.05: Distributions likely same (no drift)
    - p_value < 0.05: Distributions differ significantly (drift detected)
    """
    if len(baseline) == 0 or len(recent) == 0:
        raise ValueError("Baseline and recent must have non-zero length")
    
    # Get unique categories
    all_categories = set(baseline) | set(recent)
    
    # Build frequency tables
    expected_freq = []
    observed_freq = []
    
    for category in sorted(all_categories):
        expected_count = (baseline == category).sum()
        observed_count = (recent == category).sum()
        expected_freq.append(max(expected_count, 1))  # Avoid 0 in chi-square
        observed_freq.append(max(observed_count, 1))
    
    # Normalize expected to match observed total
    expected_freq = np.array(expected_freq, dtype=float)
    observed_freq = np.array(observed_freq, dtype=float)
    expected_freq = expected_freq * (observed_freq.sum() / expected_freq.sum())
    
    # Compute chi-square
    chi2_stat, pvalue = chisquare(observed_freq, expected_freq)
    
    return float(chi2_stat), float(pvalue)

def compute_chi2_binned(baseline: np.ndarray, recent: np.ndarray, bins: int = 10) -> Tuple[float, float]:
    """
    Compute Chi-square test for continuous features (binned).
    """
    if len(baseline) == 0 or len(recent) == 0:
        raise ValueError("Baseline and recent must have non-zero length")
    
    # Create bins based on baseline
    _, bin_edges = np.histogram(baseline, bins=bins)
    
    # Digitize both distributions
    baseline_binned = np.digitize(baseline, bin_edges)
    recent_binned = np.digitize(recent, bin_edges)
    
    # Get counts per bin
    baseline_counts = np.bincount(baseline_binned)
    recent_counts = np.bincount(recent_binned)
    
    # Pad to same length
    max_len = max(len(baseline_counts), len(recent_counts))
    baseline_counts = np.pad(baseline_counts, (0, max_len - len(baseline_counts)))
    recent_counts = np.pad(recent_counts, (0, max_len - len(recent_counts)))
    
    # Normalize baseline to match recent total
    baseline_counts = baseline_counts * (recent_counts.sum() / max(baseline_counts.sum(), 1))
    
    # Chi-square test
    chi2_stat, pvalue = chisquare(recent_counts, baseline_counts)
    
    return float(chi2_stat), float(pvalue)
