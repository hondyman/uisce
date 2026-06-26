#!/usr/bin/env python3
"""
Fourier Features and Autocorrelation Service
Generate periodic and autocorrelation-based features for time-series.
"""

import numpy as np
import pandas as pd
from typing import Dict, List, Tuple
from dataclasses import dataclass
import logging

logger = logging.getLogger(__name__)

@dataclass
class FourierFeaturesResult:
    """Result of Fourier features generation"""
    features_df: pd.DataFrame
    detected_periods: List[Tuple[float, float]]  # (period, strength)
    dominant_period: float
    explained_variance: List[float]

class FourierFeaturesGenerator:
    """
    Generate Fourier (sin/cos) features for periodic patterns
    
    y(t) = a₀ + Σ(aₙ*cos(2πnt/T) + bₙ*sin(2πnt/T))
    
    Where:
    - T = period (e.g., 365 for yearly, 7 for weekly)
    - n = harmonic number (1, 2, 3, ...)
    """
    
    def __init__(self, timeseries: np.ndarray, timestamps: np.ndarray):
        """Initialize with time-series"""
        self.timeseries = np.asarray(timeseries, dtype=np.float64)
        self.timestamps = timestamps
        self.n = len(timeseries)
    
    def detect_dominant_periods(self, max_lag: int = 180) -> List[Tuple[float, float]]:
        """
        Auto-detect dominant seasonal periods using FFT
        
        Returns: List of (period, strength) tuples, sorted by strength
        """
        # Compute FFT
        fft = np.fft.fft(self.timeseries)
        power = np.abs(fft) ** 2
        
        # Normalize power spectrum
        power = power / np.sum(power)
        
        # Find top periods (skip DC component and Nyquist)
        freqs = np.fft.fftfreq(self.n)
        top_indices = np.argsort(power[1:self.n//2])[-5:]
        
        periods = []
        for idx in top_indices[::-1]:  # Reverse to get descending strength
            if idx == 0:
                continue
            period = self.n / (idx + 1)
            if 2 <= period <= max_lag:  # Reasonable period range
                strength = power[idx + 1]
                periods.append((float(period), float(strength)))
        
        return periods
    
    def generate_fourier_features(self, frequencies: Dict[str, float], 
                                 num_harmonics: int = 3) -> pd.DataFrame:
        """
        Generate Fourier (sin/cos) features
        
        Args:
            frequencies: Dict of {name: period} e.g., {'yearly': 365.25, 'weekly': 7}
            num_harmonics: Number of harmonics to generate per frequency
        
        Returns: DataFrame with sin/cos columns
        
        Example:
            frequencies = {
                'yearly': 365.25,
                'weekly': 7.0,
                'daily': 1.0
            }
            features = gen.generate_fourier_features(frequencies, num_harmonics=3)
            # Generates: sin_yearly_1, cos_yearly_1, sin_yearly_2, cos_yearly_2, ...
        """
        features_dict = {}
        t = np.arange(self.n)
        
        for freq_name, period in frequencies.items():
            for harmonic in range(1, num_harmonics + 1):
                # Angular frequency: 2π * harmonic / period
                angle = 2 * np.pi * harmonic * t / period
                
                features_dict[f'sin_{freq_name}_{harmonic}'] = np.sin(angle)
                features_dict[f'cos_{freq_name}_{harmonic}'] = np.cos(angle)
        
        features_df = pd.DataFrame(features_dict)
        
        logger.info(f"Generated {len(features_dict)} Fourier features "
                   f"from {len(frequencies)} frequencies with {num_harmonics} harmonics")
        
        return features_df
    
    def compute_feature_importance(self, features_df: pd.DataFrame) -> Dict[str, float]:
        """
        Compute importance of each Fourier feature
        
        Metric: Correlation with original time-series
        
        Returns: Dict of {feature_name: importance}
        """
        importance = {}
        
        for col in features_df.columns:
            corr = np.corrcoef(self.timeseries, features_df[col])[0, 1]
            importance[col] = abs(float(corr))
        
        # Normalize to [0, 1]
        max_imp = max(importance.values()) if importance else 1.0
        importance = {k: v / max_imp for k, v in importance.items()}
        
        return importance
    
    def reconstruct_from_features(self, features_df: pd.DataFrame, 
                                 feature_names: List[str]) -> np.ndarray:
        """
        Reconstruct signal from selected Fourier features
        
        Useful for: Understanding how much of signal is captured by periodicity
        """
        selected_features = features_df[feature_names].values
        
        # Fit linear model to reconstruct
        try:
            from sklearn.linear_model import LinearRegression
            model = LinearRegression()
            model.fit(selected_features, self.timeseries)
            reconstructed = model.predict(selected_features)
            return np.asarray(reconstructed)
        except:
            logger.warning("Reconstruction failed, returning zeros")
            return np.zeros(self.n)
    
    def get_result(self, frequencies: Dict[str, float] = None,
                   num_harmonics: int = 3) -> FourierFeaturesResult:
        """
        Complete Fourier feature extraction
        
        Args:
            frequencies: Custom frequencies, or auto-detect if None
            num_harmonics: Number of harmonics per frequency
        
        Returns: FourierFeaturesResult with features and diagnostics
        """
        # Auto-detect if not provided
        if frequencies is None:
            # Use standard seasonal periods
            frequencies = {
                'yearly': 365.25,
                'weekly': 7.0
            }
            # Add daily if high-frequency data
            if self.n > 1000:
                frequencies['daily'] = 1.0
        
        # Generate features
        features_df = self.generate_fourier_features(frequencies, num_harmonics)
        
        # Compute importance
        importance_dict = self.compute_feature_importance(features_df)
        importance_scores = sorted(importance_dict.values(), reverse=True)
        
        # Detect periods
        periods = self.detect_dominant_periods()
        
        # Compute explained variance by top features
        top_features = sorted(importance_dict, key=importance_dict.get, reverse=True)[:5]
        reconstructed = self.reconstruct_from_features(features_df, top_features)
        
        from sklearn.metrics import r2_score
        try:
            explained_var = [r2_score(self.timeseries, features_df[col].values) 
                            for col in top_features]
        except:
            explained_var = []
        
        return FourierFeaturesResult(
            features_df=features_df,
            detected_periods=periods,
            dominant_period=periods[0][0] if periods else 7.0,
            explained_variance=explained_var
        )

class AutocorrelationFeaturesGenerator:
    """
    Generate lag-based and autocorrelation features
    """
    
    def __init__(self, timeseries: np.ndarray):
        """Initialize with time-series"""
        self.timeseries = np.asarray(timeseries, dtype=np.float64)
        self.n = len(timeseries)
    
    def create_lag_features(self, lags: List[int] = [1, 7, 14, 30]) -> pd.DataFrame:
        """
        Create lagged versions of the time-series
        
        lag_k = y(t-k)
        
        Returns: DataFrame with lag columns
        """
        df = pd.DataFrame({'value': self.timeseries})
        
        for lag in lags:
            df[f'lag_{lag}'] = df['value'].shift(lag)
        
        # Remove NaN rows
        df = df.dropna()
        
        logger.info(f"Created {len(lags)} lag features")
        return df.drop('value', axis=1)
    
    def create_rolling_features(self, windows: List[int] = [7, 14, 30]) -> pd.DataFrame:
        """
        Create rolling statistical features
        
        Returns: mean, std, min, max for each window
        """
        features_dict = {}
        
        for window in windows:
            features_dict[f'rolling_mean_{window}'] = self._rolling_mean(window)
            features_dict[f'rolling_std_{window}'] = self._rolling_std(window)
            features_dict[f'rolling_min_{window}'] = self._rolling_min(window)
            features_dict[f'rolling_max_{window}'] = self._rolling_max(window)
            features_dict[f'rolling_median_{window}'] = self._rolling_median(window)
        
        features_df = pd.DataFrame(features_dict)
        logger.info(f"Created rolling features from {len(windows)} windows")
        return features_df
    
    def compute_autocorrelation(self, max_lag: int = 30) -> Dict[int, float]:
        """
        Compute autocorrelation function (ACF)
        
        acf(k) = correlation(y(t), y(t-k))
        
        Returns: Dict of {lag: acf_value}
        """
        acf_values = {}
        mean = np.mean(self.timeseries)
        c0 = np.sum((self.timeseries - mean) ** 2) / self.n
        
        for lag in range(1, min(max_lag + 1, self.n)):
            c_k = np.sum((self.timeseries[:-lag] - mean) * 
                        (self.timeseries[lag:] - mean)) / self.n
            acf_values[lag] = float(c_k / (c0 + 1e-9))
        
        return acf_values
    
    def compute_partial_autocorrelation(self, max_lag: int = 30) -> Dict[int, float]:
        """
        Compute partial autocorrelation function (PACF)
        
        Correlation after removing effects of intermediate lags
        """
        try:
            from statsmodels.graphics.tsaplots import pacf as compute_pacf
            pacf_values = compute_pacf(self.timeseries, nlags=max_lag, method='ywm')
            return {i: float(v) for i, v in enumerate(pacf_values[1:], start=1)}
        except:
            # Fallback: simple lag-1 partial correlation
            logger.warning("PACF computation failed, using fallback")
            acf = self.compute_autocorrelation(max_lag)
            return {1: float(acf.get(1, 0.0))}
    
    def get_autocorrelation_features(self, acf_lags: List[int] = [1, 7, 14, 30],
                                    pacf_lags: List[int] = [1, 7, 14, 30]) -> Dict[str, float]:
        """
        Get ACF/PACF values at specific lags
        
        Returns: Dict with features like acf_lag_1, pacf_lag_7, etc.
        """
        acf_values = self.compute_autocorrelation(max(acf_lags) if acf_lags else 30)
        pacf_values = self.compute_partial_autocorrelation(max(pacf_lags) if pacf_lags else 30)
        
        features = {}
        
        for lag in acf_lags:
            features[f'acf_lag_{lag}'] = acf_values.get(lag, 0.0)
        
        for lag in pacf_lags:
            features[f'pacf_lag_{lag}'] = pacf_values.get(lag, 0.0)
        
        return features
    
    # Helper methods for rolling statistics
    def _rolling_mean(self, window: int) -> np.ndarray:
        """Simple rolling mean"""
        result = np.full(self.n, np.nan)
        for i in range(window - 1, self.n):
            result[i] = np.mean(self.timeseries[i - window + 1:i + 1])
        return result
    
    def _rolling_std(self, window: int) -> np.ndarray:
        """Rolling standard deviation"""
        result = np.full(self.n, np.nan)
        for i in range(window - 1, self.n):
            result[i] = np.std(self.timeseries[i - window + 1:i + 1])
        return result
    
    def _rolling_min(self, window: int) -> np.ndarray:
        """Rolling minimum"""
        result = np.full(self.n, np.nan)
        for i in range(window - 1, self.n):
            result[i] = np.min(self.timeseries[i - window + 1:i + 1])
        return result
    
    def _rolling_max(self, window: int) -> np.ndarray:
        """Rolling maximum"""
        result = np.full(self.n, np.nan)
        for i in range(window - 1, self.n):
            result[i] = np.max(self.timeseries[i - window + 1:i + 1])
        return result
    
    def _rolling_median(self, window: int) -> np.ndarray:
        """Rolling median"""
        result = np.full(self.n, np.nan)
        for i in range(window - 1, self.n):
            result[i] = np.median(self.timeseries[i - window + 1:i + 1])
        return result

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    
    # Generate synthetic data
    t = np.arange(365)
    ts = 100 + 0.5 * t + 20 * np.sin(2 * np.pi * t / 7) + np.random.normal(0, 5, 365)
    
    # Test Fourier
    print("Testing Fourier Features...")
    fourier_gen = FourierFeaturesGenerator(ts, t)
    result = fourier_gen.get_result()
    print(f"  Generated {result.features_df.shape[1]} features")
    print(f"  Detected periods: {result.detected_periods[:3]}")
    
    # Test Autocorrelation
    print("\nTesting Autocorrelation Features...")
    acf_gen = AutocorrelationFeaturesGenerator(ts)
    acf_features = acf_gen.get_autocorrelation_features()
    print(f"  ACF at lag 1: {acf_features['acf_lag_1']:.3f}")
    print(f"  ACF at lag 7: {acf_features['acf_lag_7']:.3f}")
