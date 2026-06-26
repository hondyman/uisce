#!/usr/bin/env python3
"""
Advanced Time-Series Anomaly Detection
Combines statistical methods with machine learning
"""

import numpy as np
import pandas as pd
from typing import List, Optional, Tuple, Dict
from dataclasses import dataclass
import logging

logger = logging.getLogger(__name__)

@dataclass
class AnomalyResult:
    """Result of anomaly detection"""
    timestamps: np.ndarray
    values: np.ndarray
    anomaly_scores: np.ndarray  # [0-1], higher = more anomalous
    is_anomaly: np.ndarray  # Boolean array
    anomaly_indices: List[int]
    anomaly_types: List[str]  # 'statistical', 'isolation_forest', 'dbscan'
    n_anomalies: int
    anomaly_percentage: float
    thresholds: Dict[str, float]  # Method thresholds used

class StatisticalAnomalyDetector:
    """
    Statistical anomaly detection
    
    Methods:
    1. Z-score: Points >threshold σ from mean
    2. IQR: Points outside 1.5 * IQR from quartiles
    3. Modified Z-score: Using median instead of mean (robust)
    """
    
    def __init__(self, timeseries: np.ndarray, timestamps: np.ndarray = None):
        self.timeseries = np.asarray(timeseries, dtype=np.float64)
        self.timestamps = timestamps if timestamps is not None else np.arange(len(timeseries))
        self.n = len(timeseries)
    
    def detect_zscore(self, threshold: float = 3.0) -> np.ndarray:
        """
        Z-score based detection
        
        z(i) = (y(i) - μ) / σ
        Anomaly if |z(i)| > threshold
        
        Args:
            threshold: Number of std devs (default 3 = 0.3% in normal distribution)
        
        Returns: Boolean array of anomalies
        """
        mean = np.mean(self.timeseries)
        std = np.std(self.timeseries)
        
        if std == 0:
            return np.zeros(self.n, dtype=bool)
        
        z_scores = np.abs((self.timeseries - mean) / std)
        anomalies = z_scores > threshold
        
        n_anom = np.sum(anomalies)
        logger.info(f"Z-score: Found {n_anom} anomalies with threshold={threshold}")
        
        return anomalies
    
    def detect_iqr(self, multiplier: float = 1.5) -> np.ndarray:
        """
        Interquartile range based detection
        
        IQR = Q3 - Q1
        Lower bound = Q1 - multiplier * IQR
        Upper bound = Q3 + multiplier * IQR
        
        Args:
            multiplier: IQR multiplier (default 1.5 = standard Tukey)
        
        Returns: Boolean array of anomalies
        """
        q1 = np.percentile(self.timeseries, 25)
        q3 = np.percentile(self.timeseries, 75)
        iqr = q3 - q1
        
        lower = q1 - multiplier * iqr
        upper = q3 + multiplier * iqr
        
        anomalies = (self.timeseries < lower) | (self.timeseries > upper)
        
        n_anom = np.sum(anomalies)
        logger.info(f"IQR: Found {n_anom} anomalies, bounds=[{lower:.2f}, {upper:.2f}]")
        
        return anomalies
    
    def detect_modified_zscore(self, threshold: float = 3.5) -> np.ndarray:
        """
        Modified Z-score using median (more robust to outliers)
        
        Median Absolute Deviation: MAD = median(|y(i) - median(Y)|)
        Modified z-score: 0.6745 * (y(i) - median(Y)) / MAD
        
        Args:
            threshold: Number of MAD units (default 3.5)
        
        Returns: Boolean array of anomalies
        """
        median = np.median(self.timeseries)
        mad = np.median(np.abs(self.timeseries - median))
        
        if mad == 0:
            return np.zeros(self.n, dtype=bool)
        
        modified_z = 0.6745 * (self.timeseries - median) / mad
        anomalies = np.abs(modified_z) > threshold
        
        n_anom = np.sum(anomalies)
        logger.info(f"Modified Z-score: Found {n_anom} anomalies with threshold={threshold}")
        
        return anomalies

class IsolationForestDetector:
    """
    Isolation Forest anomaly detection (unsupervised ML)
    
    Algorithm:
    1. Randomly select feature and split value
    2. Repeat to build isolation trees
    3. Anomalies = normal points take longer paths to isolate
    """
    
    def __init__(self, timeseries: np.ndarray, timestamps: np.ndarray = None):
        self.timeseries = np.asarray(timeseries, dtype=np.float64)
        self.timestamps = timestamps if timestamps is not None else np.arange(len(timeseries))
        self.n = len(timeseries)
    
    def detect(self, contamination: float = 0.05, random_state: int = 42) -> np.ndarray:
        """
        Detect anomalies using Isolation Forest
        
        Args:
            contamination: Expected proportion of anomalies (0.01-0.5)
            random_state: Random seed for reproducibility
        
        Returns: Boolean array of anomalies
        """
        try:
            from sklearn.ensemble import IsolationForest
            
            # Prepare feature matrix: value, lag-1, lag-7 (3D)
            features = self._create_features()
            
            # Fit isolation forest
            iso_forest = IsolationForest(
                contamination=contamination,
                random_state=random_state,
                n_estimators=100,
                max_samples='auto'
            )
            
            predictions = iso_forest.fit_predict(features)
            anomalies = predictions == -1
            
            # Get anomaly scores
            scores = -iso_forest.score_samples(features)
            
            n_anom = np.sum(anomalies)
            logger.info(f"Isolation Forest: Found {n_anom} anomalies "
                       f"(contamination={contamination})")
            
            self.anomaly_scores = scores
            
            return anomalies
        
        except ImportError:
            logger.warning("scikit-learn not available, using statistical fallback")
            detector = StatisticalAnomalyDetector(self.timeseries, self.timestamps)
            return detector.detect_zscore()
    
    def _create_features(self) -> np.ndarray:
        """Create feature matrix for IF"""
        features = [self.timeseries]
        
        # Lag-1
        lag1 = np.concatenate([[self.timeseries[0]], self.timeseries[:-1]])
        features.append(lag1)
        
        # Lag-7
        lag7 = np.concatenate([np.full(7, self.timeseries[0]), self.timeseries[:-7]])
        features.append(lag7)
        
        return np.column_stack(features)

class DBSCANDetector:
    """
    DBSCAN-based anomaly detection
    
    Identifies points in low-density regions as anomalies
    """
    
    def __init__(self, timeseries: np.ndarray, timestamps: np.ndarray = None):
        self.timeseries = np.asarray(timeseries, dtype=np.float64)
        self.timestamps = timestamps if timestamps is not None else np.arange(len(timeseries))
        self.n = len(timeseries)
    
    def detect(self, eps: float = 0.5, min_samples: int = 5) -> np.ndarray:
        """
        Detect anomalies using DBSCAN
        
        Args:
            eps: Maximum distance between points in cluster
            min_samples: Minimum points in cluster
        
        Returns: Boolean array (True = outlier)
        """
        try:
            from sklearn.cluster import DBSCAN
            from sklearn.preprocessing import StandardScaler
            
            # Prepare features
            features = self._create_features()
            features = StandardScaler().fit_transform(features)
            
            # DBSCAN clustering
            dbscan = DBSCAN(eps=eps, min_samples=min_samples)
            labels = dbscan.fit_predict(features)
            
            # Points labeled -1 are outliers
            anomalies = labels == -1
            
            n_anom = np.sum(anomalies)
            logger.info(f"DBSCAN: Found {n_anom} anomalies (eps={eps}, "
                       f"min_samples={min_samples})")
            
            return anomalies
        
        except ImportError:
            logger.warning("scikit-learn not available, using IQR fallback")
            detector = StatisticalAnomalyDetector(self.timeseries, self.timestamps)
            return detector.detect_iqr()
    
    def _create_features(self) -> np.ndarray:
        """Create feature matrix for DBSCAN"""
        return np.column_stack([
            self.timeseries,
            np.concatenate([[self.timeseries[0]], self.timeseries[:-1]]),
            np.concatenate([np.full(7, self.timeseries[0]), self.timeseries[:-7]])
        ])

class EnsembleAnomalyDetector:
    """
    Ensemble combining multiple detection methods
    
    Combines:
    1. Statistical (Z-score, IQR, Modified Z)
    2. Isolation Forest
    3. DBSCAN
    
    Voting scheme: Point is anomaly if 2+ methods agree
    """
    
    def __init__(self, timeseries: np.ndarray, timestamps: np.ndarray = None,
                 weights: Dict[str, float] = None):
        self.timeseries = np.asarray(timeseries, dtype=np.float64)
        self.timestamps = timestamps if timestamps is not None else np.arange(len(timeseries))
        self.n = len(timeseries)
        self.weights = weights or {
            'zscore': 1.0,
            'iqr': 1.0,
            'modified_zscore': 1.2,
            'isolation_forest': 1.2,
            'dbscan': 1.0
        }
    
    def detect(self) -> AnomalyResult:
        """
        Run ensemble detection
        
        Returns: AnomalyResult with combined predictions and diagnostics
        """
        detections = {}
        scores = np.zeros(self.n)
        
        # Statistical methods
        stat_detector = StatisticalAnomalyDetector(self.timeseries, self.timestamps)
        
        detections['zscore'] = stat_detector.detect_zscore(threshold=3.0)
        detections['iqr'] = stat_detector.detect_iqr()
        detections['modified_zscore'] = stat_detector.detect_modified_zscore()
        
        scores += self.weights['zscore'] * detections['zscore'].astype(float)
        scores += self.weights['iqr'] * detections['iqr'].astype(float)
        scores += self.weights['modified_zscore'] * detections['modified_zscore'].astype(float)
        
        # ML methods
        try:
            if_detector = IsolationForestDetector(self.timeseries, self.timestamps)
            detections['isolation_forest'] = if_detector.detect(contamination=0.05)
            scores += self.weights['isolation_forest'] * detections['isolation_forest'].astype(float)
            
            dbscan_detector = DBSCANDetector(self.timeseries, self.timestamps)
            detections['dbscan'] = dbscan_detector.detect()
            scores += self.weights['dbscan'] * detections['dbscan'].astype(float)
        
        except Exception as e:
            logger.warning(f"ML-based detection failed: {e}")
        
        # Normalize scores
        scores = scores / np.sum(list(self.weights.values()))
        
        # Voting: 2+ methods agree
        total_votes = np.zeros(self.n)
        for method, detection in detections.items():
            total_votes += detection.astype(int)
        
        voting_threshold = 2
        is_anomaly = total_votes >= voting_threshold
        
        # Get anomaly types
        anomaly_types = []
        for method, detection in detections.items():
            if np.any(detection):
                anomaly_types.append(method)
        
        anomaly_indices = np.where(is_anomaly)[0].tolist()
        
        return AnomalyResult(
            timestamps=self.timestamps,
            values=self.timeseries,
            anomaly_scores=scores,
            is_anomaly=is_anomaly,
            anomaly_indices=anomaly_indices,
            anomaly_types=anomaly_types,
            n_anomalies=int(np.sum(is_anomaly)),
            anomaly_percentage=float(np.sum(is_anomaly) / self.n * 100),
            thresholds={
                'zscore': 3.0,
                'iqr': 1.5,
                'modified_zscore': 3.5,
                'voting_threshold': voting_threshold
            }
        )

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    
    # Generate synthetic data with anomalies
    np.random.seed(42)
    t = np.arange(365)
    ts = 100 + 0.5 * t + 20 * np.sin(2 * np.pi * t / 7) + np.random.normal(0, 5, 365)
    
    # Inject anomalies
    ts[50] = 200
    ts[100] = 50
    ts[250] = 220
    
    print("Testing Anomaly Detection Methods...")
    
    # Statistical methods
    print("\n1. Statistical Methods:")
    stat = StatisticalAnomalyDetector(ts, t)
    
    zscore_anom = stat.detect_zscore()
    print(f"   Z-score: {np.sum(zscore_anom)} anomalies")
    
    iqr_anom = stat.detect_iqr()
    print(f"   IQR: {np.sum(iqr_anom)} anomalies")
    
    mz_anom = stat.detect_modified_zscore()
    print(f"   Modified Z-score: {np.sum(mz_anom)} anomalies")
    
    # Ensemble
    print("\n2. Ensemble Detection:")
    ensemble = EnsembleAnomalyDetector(ts, t)
    result = ensemble.detect()
    print(f"   Found {result.n_anomalies} anomalies ({result.anomaly_percentage:.1f}%)")
    print(f"   Anomaly indices: {result.anomaly_indices}")
    print(f"   Methods used: {result.anomaly_types}")
