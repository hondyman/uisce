#!/usr/bin/env python3
"""
Time-Series Decomposition Service
Extracts trend, seasonality, and residual components from time-series features.
"""

import numpy as np
import pandas as pd
from typing import Dict, Tuple, Optional
from datetime import datetime, timedelta
from dataclasses import dataclass
import logging

logger = logging.getLogger(__name__)

@dataclass
class DecompositionResult:
    """Result of time-series decomposition"""
    trend: np.ndarray
    seasonal: np.ndarray
    residual: np.ndarray
    timestamp: np.ndarray
    method: str
    period: int
    variance_explained: float
    residual_std: float
    detected_period: Optional[int] = None
    has_anomalies: bool = False
    anomaly_indices: Optional[np.ndarray] = None

class TimeSeriesDecomposition:
    """
    Decompose time-series into trend, seasonality, residual components
    
    Supported methods:
    - Additive: y(t) = trend(t) + seasonal(t) + residual(t)
    - Multiplicative: y(t) = trend(t) * seasonal(t) * residual(t)
    - Robust: Using robust loess resistant to outliers
    """
    
    def __init__(self, timeseries: np.ndarray, timestamps: np.ndarray, 
                 period: Optional[int] = None):
        """
        Initialize with time-series data
        
        Args:
            timeseries: Time-series values (1D array)
            timestamps: Timestamps corresponding to values
            period: Seasonality period (auto-detect if None)
        """
        self.timeseries = np.asarray(timeseries, dtype=np.float64)
        self.timestamps = timestamps
        self.period = period or self._detect_period()
        self.n = len(self.timeseries)
    
    def _detect_period(self) -> int:
        """
        Auto-detect seasonality period using FFT
        
        Returns: Detected period (or default 7 if no clear periodicity)
        """
        if len(self.timeseries) < 100:
            return 7  # Default weekly for short series
        
        # Compute FFT
        fft = np.fft.fft(self.timeseries)
        power = np.abs(fft) ** 2
        
        # Find dominant frequencies (skip DC component)
        freq_index = np.argsort(power[1:len(power)//2])[-1]
        detected_period = int(len(self.timeseries) / (freq_index + 1))
        
        # Sanity check: Period should be reasonable (3-365)
        detected_period = np.clip(detected_period, 3, 365)
        
        logger.info(f"Detected seasonality period: {detected_period}")
        return detected_period
    
    def decompose_additive(self) -> DecompositionResult:
        """
        Additive decomposition: y(t) = trend(t) + seasonal(t) + residual(t)
        
        Best for: Constant seasonal magnitude
        
        Returns: DecompositionResult with components
        """
        # 1. Extract trend using centered moving average
        window = max(self.period, 7)
        trend = self._moving_average_centered(self.timeseries, window)
        
        # 2. Detrend to extract seasonality
        detrended = self.timeseries - trend
        
        # 3. Compute seasonal component (average at each position in cycle)
        seasonal = np.zeros(self.n)
        for i in range(self.period):
            indices = np.arange(i, self.n, self.period)
            seasonal[indices] = np.nanmean(detrended[indices])
        
        # Center seasonal component (should average to 0)
        seasonal = seasonal - np.nanmean(seasonal)
        
        # 4. Compute residual
        residual = self.timeseries - trend - seasonal
        
        # 5. Calculate quality metrics
        variance_explained = self._calculate_r_squared(self.timeseries, trend + seasonal)
        residual_std = np.nanstd(residual)
        
        # 6. Detect anomalies in residuals
        anomalies, anomaly_indices = self._detect_anomalies_statistical(residual)
        
        logger.info(f"Additive decomposition: R²={variance_explained:.3f}, "
                   f"residual_std={residual_std:.3f}, anomalies={np.sum(anomalies)}")
        
        return DecompositionResult(
            trend=trend,
            seasonal=seasonal,
            residual=residual,
            timestamp=self.timestamps,
            method='additive',
            period=self.period,
            variance_explained=variance_explained,
            residual_std=residual_std,
            detected_period=self.period,
            has_anomalies=np.any(anomalies),
            anomaly_indices=anomaly_indices
        )
    
    def decompose_multiplicative(self) -> DecompositionResult:
        """
        Multiplicative decomposition: y(t) = trend(t) * seasonal(t) * residual(t)
        
        Best for: Growing/shrinking seasonal magnitude
        
        Returns: DecompositionResult with components
        """
        # 1. Extract trend using centered moving average
        window = max(self.period, 7)
        trend = self._moving_average_centered(self.timeseries, window)
        
        # Avoid division by zero
        trend = np.where(trend > 1e-9, trend, 1e-9)
        
        # 2. Compute seasonal component (ratios)
        detrended = self.timeseries / trend
        seasonal = np.zeros(self.n)
        for i in range(self.period):
            indices = np.arange(i, self.n, self.period)
            seasonal[indices] = np.nanmean(detrended[indices])
        
        # Center seasonal component (should average to 1.0)
        seasonal = seasonal / np.nanmean(seasonal)
        
        # 3. Compute residual (as ratios)
        residual = self.timeseries / (trend * seasonal + 1e-9)
        
        # 4. Calculate quality metrics
        variance_explained = self._calculate_r_squared(self.timeseries, trend * seasonal)
        residual_std = np.nanstd(np.log(np.abs(residual) + 1e-9))
        
        # 5. Detect anomalies
        anomalies, anomaly_indices = self._detect_anomalies_statistical(np.log(np.abs(residual) + 1e-9))
        
        logger.info(f"Multiplicative decomposition: R²={variance_explained:.3f}, "
                   f"residual_std={residual_std:.3f}, anomalies={np.sum(anomalies)}")
        
        return DecompositionResult(
            trend=trend,
            seasonal=seasonal,
            residual=residual,
            timestamp=self.timestamps,
            method='multiplicative',
            period=self.period,
            variance_explained=variance_explained,
            residual_std=residual_std,
            detected_period=self.period,
            has_anomalies=np.any(anomalies),
            anomaly_indices=anomaly_indices
        )
    
    def decompose_robust(self) -> DecompositionResult:
        """
        Robust decomposition using LOWESS (resistant to outliers)
        
        More robust to outliers than standard moving average
        
        Returns: DecompositionResult with components
        """
        try:
            from statsmodels.nonparametric.smoothers_lowess import lowess
        except ImportError:
            logger.warning("statsmodels not available, falling back to additive")
            return self.decompose_additive()
        
        # 1. Extract trend using LOWESS (robust locally weighted regression)
        frac = max(0.1, min(0.5, self.period / self.n))
        x = np.arange(self.n)
        lowess_result = lowess(self.timeseries, x, frac=frac, it=3)
        trend = lowess_result[:, 1]
        
        # 2. Detrend
        detrended = self.timeseries - trend
        
        # 3. Extract seasonal component
        seasonal = np.zeros(self.n)
        for i in range(self.period):
            indices = np.arange(i, self.n, self.period)
            # Use median instead of mean for robustness
            seasonal[indices] = np.nanmedian(detrended[indices])
        
        seasonal = seasonal - np.nanmedian(seasonal)
        
        # 4. Residual
        residual = self.timeseries - trend - seasonal
        
        # 5. Quality metrics
        variance_explained = self._calculate_r_squared(self.timeseries, trend + seasonal)
        residual_std = np.nanstd(residual)
        
        # 6. Detect anomalies
        anomalies, anomaly_indices = self._detect_anomalies_statistical(residual, threshold=2.5)
        
        logger.info(f"Robust decomposition: R²={variance_explained:.3f}, "
                   f"residual_std={residual_std:.3f}, anomalies={np.sum(anomalies)}")
        
        return DecompositionResult(
            trend=trend,
            seasonal=seasonal,
            residual=residual,
            timestamp=self.timestamps,
            method='robust',
            period=self.period,
            variance_explained=variance_explained,
            residual_std=residual_std,
            detected_period=self.period,
            has_anomalies=np.any(anomalies),
            anomaly_indices=anomaly_indices
        )
    
    # Helper methods
    
    def _moving_average_centered(self, series: np.ndarray, window: int) -> np.ndarray:
        """Centered moving average"""
        kernel = np.ones(window) / window
        padded = np.pad(series, (window//2, window//2), mode='edge')
        smoothed = np.convolve(padded, kernel, mode='valid')
        return smoothed[:self.n]
    
    def _calculate_r_squared(self, actual: np.ndarray, predicted: np.ndarray) -> float:
        """Calculate R² (coefficient of determination)"""
        ss_res = np.nansum((actual - predicted) ** 2)
        ss_tot = np.nansum((actual - np.nanmean(actual)) ** 2)
        r_squared = 1 - (ss_res / (ss_tot + 1e-9))
        return np.clip(r_squared, 0, 1)
    
    def _detect_anomalies_statistical(self, residuals: np.ndarray, 
                                     threshold: float = 3.0) -> Tuple[np.ndarray, np.ndarray]:
        """Detect anomalies as points >threshold * std from mean"""
        mean = np.nanmean(residuals)
        std = np.nanstd(residuals)
        
        anomalies = np.abs(residuals - mean) > (threshold * std)
        anomaly_indices = np.where(anomalies)[0]
        
        return anomalies, anomaly_indices
    
    def get_decomposition_df(self, result: DecompositionResult) -> pd.DataFrame:
        """Return decomposition as pandas DataFrame"""
        return pd.DataFrame({
            'timestamp': result.timestamp,
            'original': self.timeseries,
            'trend': result.trend,
            'seasonal': result.seasonal,
            'residual': result.residual
        })

if __name__ == '__main__':
    # Example usage
    logging.basicConfig(level=logging.INFO)
    
    # Generate synthetic time-series
    t = np.arange(365)
    trend = 100 + 0.5 * t
    seasonality = 20 * np.sin(2 * np.pi * t / 7)
    noise = np.random.normal(0, 5, 365)
    ts = trend + seasonality + noise
    
    # Decompose
    decomp = TimeSeriesDecomposition(
        timeseries=ts,
        timestamps=t,
        period=7
    )
    
    result = decomp.decompose_additive()
    df = decomp.get_decomposition_df(result)
    
    print(f"\nDecomposition Results:")
    print(f"  Variance Explained: {result.variance_explained:.3f}")
    print(f"  Residual Std: {result.residual_std:.3f}")
    print(f"  Anomalies Found: {result.has_anomalies}")
    print(f"\nFirst 10 rows:")
    print(df.head(10))
