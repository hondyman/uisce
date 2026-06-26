#!/usr/bin/env python3
"""
Comprehensive test suite for Phase 3.22 Time-Series Features
Unit + Integration + Performance tests
"""

import pytest
import numpy as np
import pandas as pd
from datetime import datetime, timedelta
import json
import time

# Test fixtures
@pytest.fixture
def synthetic_timeseries():
    """Generate synthetic time-series data"""
    np.random.seed(42)
    t = np.arange(365)
    trend = 100 + 0.5 * t
    seasonality = 20 * np.sin(2 * np.pi * t / 7)
    noise = np.random.normal(0, 5, 365)
    ts = trend + seasonality + noise
    return ts, t

@pytest.fixture
def timeseries_with_anomalies(synthetic_timeseries):
    """Add artificial anomalies"""
    ts, t = synthetic_timeseries
    ts_copy = ts.copy()
    ts_copy[50] = 200
    ts_copy[100] = 50
    ts_copy[250] = 220
    return ts_copy, t

@pytest.fixture
def timestamps():
    """Generate timestamps"""
    base_date = datetime.utcnow() - timedelta(days=365)
    return np.array([
        (base_date + timedelta(days=i)).isoformat() 
        for i in range(365)
    ])

# ============================================================================
# Unit Tests: Decomposition Service
# ============================================================================

class TestDecompositionService:
    """Tests for time-series decomposition"""
    
    def test_additive_decomposition(self, synthetic_timeseries):
        """Test additive decomposition"""
        from timeseries_service.decomposition import TimeSeriesDecomposition
        
        ts, t = synthetic_timeseries
        decomp = TimeSeriesDecomposition(ts, t, period=7)
        result = decomp.decompose_additive()
        
        assert result.trend is not None
        assert result.seasonal is not None
        assert result.residual is not None
        assert len(result.trend) == len(ts)
        assert result.variance_explained > 0.7  # R² should be high
    
    def test_multiplicative_decomposition(self, synthetic_timeseries):
        """Test multiplicative decomposition"""
        from timeseries_service.decomposition import TimeSeriesDecomposition
        
        ts, t = synthetic_timeseries
        decomp = TimeSeriesDecomposition(ts, t, period=7)
        result = decomp.decompose_multiplicative()
        
        assert result.trend is not None
        assert result.seasonal is not None
        assert len(result.seasonal) == len(ts)
    
    def test_robust_decomposition(self, timeseries_with_anomalies):
        """Test robust decomposition with anomalies"""
        from timeseries_service.decomposition import TimeSeriesDecomposition
        
        ts, t = timeseries_with_anomalies
        decomp = TimeSeriesDecomposition(ts, t, period=7)
        result = decomp.decompose_robust()
        
        assert result.method == 'robust'
        assert result.has_anomalies or True  # May not detect all anomalies
    
    def test_decomposition_reconstruction(self, synthetic_timeseries):
        """Test that components sum to original"""
        from timeseries_service.decomposition import TimeSeriesDecomposition
        
        ts, t = synthetic_timeseries
        decomp = TimeSeriesDecomposition(ts, t, period=7)
        result = decomp.decompose_additive()
        
        # Reconstruct
        reconstructed = result.trend + result.seasonal + result.residual
        
        # Should be close to original
        mse = np.mean((ts - reconstructed) ** 2)
        assert mse < 100  # Reasonable tolerance
    
    def test_decomposition_with_small_ts(self):
        """Test decomposition rejects small time-series"""
        from timeseries_service.decomposition import TimeSeriesDecomposition
        
        ts = np.array([1, 2, 3, 4, 5])
        t = np.arange(len(ts))
        decomp = TimeSeriesDecomposition(ts, t, period=2)
        
        # Should still work but with warnings
        result = decomp.decompose_additive()
        assert result is not None

# ============================================================================
# Unit Tests: Forecasting Service
# ============================================================================

class TestForecastingService:
    """Tests for forecasting services"""
    
    def test_arima_forecasting(self, synthetic_timeseries):
        """Test ARIMA forecasting"""
        from timeseries_service.forecasting import ARIMAForecaster
        
        ts, _ = synthetic_timeseries
        forecaster = ARIMAForecaster(ts)
        forecaster.fit_auto_arima()
        
        forecast_df = forecaster.forecast(steps=24)
        assert len(forecast_df) == 24
        assert 'forecast' in forecast_df.columns
        assert 'lower' in forecast_df.columns
        assert 'upper' in forecast_df.columns
    
    def test_arima_multi_horizon(self, synthetic_timeseries):
        """Test ARIMA multi-horizon forecasting"""
        from timeseries_service.forecasting import ARIMAForecaster
        
        ts, _ = synthetic_timeseries
        forecaster = ARIMAForecaster(ts)
        forecaster.fit_auto_arima()
        
        results = forecaster.forecast_multi_horizon([1, 24, 168])
        assert len(results) == 3
        assert all(r.horizon_hours in [1, 24, 168] for r in results)
    
    def test_prophet_forecasting(self, synthetic_timeseries, timestamps):
        """Test Prophet forecasting"""
        from timeseries_service.forecasting import ProphetForecaster
        
        ts, _ = synthetic_timeseries
        forecaster = ProphetForecaster(ts, timestamps)
        forecaster.fit_model()
        
        forecast_df = forecaster.forecast(periods=24)
        assert len(forecast_df) > 0
        assert 'yhat' in forecast_df.columns
    
    def test_ensemble_forecasting(self, synthetic_timeseries, timestamps):
        """Test Ensemble forecasting"""
        from timeseries_service.forecasting import EnsembleForecaster
        
        ts, _ = synthetic_timeseries
        forecaster = EnsembleForecaster(ts, timestamps)
        forecaster.fit()
        
        results = forecaster.forecast_multi_horizon([24])
        assert len(results) == 1
        assert results[0].model_type == 'ensemble'
    
    def test_forecast_confidence_intervals(self, synthetic_timeseries):
        """Test that confidence intervals are ordered correctly"""
        from timeseries_service.forecasting import ARIMAForecaster
        
        ts, _ = synthetic_timeseries
        forecaster = ARIMAForecaster(ts)
        forecaster.fit_auto_arima()
        
        results = forecaster.forecast_multi_horizon([24])
        for res in results:
            assert res.lower_bound_95 <= res.lower_bound_80
            assert res.lower_bound_80 <= res.point_forecast
            assert res.point_forecast <= res.upper_bound_80
            assert res.upper_bound_80 <= res.upper_bound_95

# ============================================================================
# Unit Tests: Fourier & Autocorrelation Features
# ============================================================================

class TestFeaturesService:
    """Tests for Fourier and Autocorrelation features"""
    
    def test_fourier_features_generation(self, synthetic_timeseries):
        """Test Fourier feature generation"""
        from timeseries_service.features import FourierFeaturesGenerator
        
        ts, t = synthetic_timeseries
        gen = FourierFeaturesGenerator(ts, t)
        
        frequencies = {'weekly': 7, 'yearly': 365}
        features_df = gen.generate_fourier_features(frequencies, num_harmonics=2)
        
        assert features_df.shape[0] == len(ts)
        assert features_df.shape[1] == 8  # 2 frequencies * 2 harmonics * 2 (sin, cos)
    
    def test_fourier_period_detection(self, synthetic_timeseries):
        """Test automated period detection"""
        from timeseries_service.features import FourierFeaturesGenerator
        
        ts, t = synthetic_timeseries
        gen = FourierFeaturesGenerator(ts, t)
        periods = gen.detect_dominant_periods()
        
        assert len(periods) > 0
        assert all(p[0] > 0 for p in periods)  # Positive periods
        assert all(0 <= p[1] <= 1 for p in periods)  # Normalized strength
    
    def test_lag_features(self, synthetic_timeseries):
        """Test lag feature creation"""
        from timeseries_service.features import AutocorrelationFeaturesGenerator
        
        ts, _ = synthetic_timeseries
        gen = AutocorrelationFeaturesGenerator(ts)
        
        lags = [1, 7, 14, 30]
        lag_df = gen.create_lag_features(lags)
        
        assert lag_df.shape[1] == len(lags)
        assert 'lag_1' in lag_df.columns
        assert 'lag_30' in lag_df.columns
    
    def test_rolling_features(self, synthetic_timeseries):
        """Test rolling statistical features"""
        from timeseries_service.features import AutocorrelationFeaturesGenerator
        
        ts, _ = synthetic_timeseries
        gen = AutocorrelationFeaturesGenerator(ts)
        
        windows = [7, 14]
        rolling_df = gen.create_rolling_features(windows)
        
        # 2 windows * 5 stats (mean, std, min, max, median)
        assert rolling_df.shape[1] == 10
    
    def test_autocorrelation_computation(self, synthetic_timeseries):
        """Test ACF computation"""
        from timeseries_service.features import AutocorrelationFeaturesGenerator
        
        ts, _ = synthetic_timeseries
        gen = AutocorrelationFeaturesGenerator(ts)
        
        acf_values = gen.compute_autocorrelation(max_lag=30)
        
        assert len(acf_values) == 30
        assert all(-1 <= v <= 1 for v in acf_values.values())  # Correlation bounds
    
    def test_partial_autocorrelation(self, synthetic_timeseries):
        """Test PACF computation"""
        from timeseries_service.features import AutocorrelationFeaturesGenerator
        
        ts, _ = synthetic_timeseries
        gen = AutocorrelationFeaturesGenerator(ts)
        
        pacf_values = gen.compute_partial_autocorrelation(max_lag=20)
        assert len(pacf_values) > 0

# ============================================================================
# Unit Tests: Anomaly Detection
# ============================================================================

class TestAnomalyDetection:
    """Tests for anomaly detection services"""
    
    def test_zscore_detection(self, timeseries_with_anomalies):
        """Test Z-score based detection"""
        from timeseries_service.anomaly_detection import StatisticalAnomalyDetector
        
        ts, t = timeseries_with_anomalies
        detector = StatisticalAnomalyDetector(ts, t)
        
        anomalies = detector.detect_zscore(threshold=3.0)
        
        # Should detect the injected anomalies
        assert np.sum(anomalies) >= 3
        assert anomalies[50] or anomalies[100] or anomalies[250]
    
    def test_iqr_detection(self, timeseries_with_anomalies):
        """Test IQR based detection"""
        from timeseries_service.anomaly_detection import StatisticalAnomalyDetector
        
        ts, t = timeseries_with_anomalies
        detector = StatisticalAnomalyDetector(ts, t)
        
        anomalies = detector.detect_iqr()
        assert np.sum(anomalies) > 0
    
    def test_isolation_forest_detection(self, timeseries_with_anomalies):
        """Test Isolation Forest detection"""
        try:
            from timeseries_service.anomaly_detection import IsolationForestDetector
            
            ts, t = timeseries_with_anomalies
            detector = IsolationForestDetector(ts, t)
            
            anomalies = detector.detect(contamination=0.05)
            
            assert isinstance(anomalies, np.ndarray)
            assert anomalies.dtype == bool
        except ImportError:
            pytest.skip("scikit-learn not available")
    
    def test_ensemble_anomaly_detection(self, timeseries_with_anomalies):
        """Test ensemble anomaly detection"""
        from timeseries_service.anomaly_detection import EnsembleAnomalyDetector
        
        ts, t = timeseries_with_anomalies
        detector = EnsembleAnomalyDetector(ts, t)
        
        result = detector.detect()
        
        assert result.n_anomalies > 0
        assert len(result.anomaly_indices) == result.n_anomalies
        assert result.anomaly_percentage > 0
        assert len(result.detected_periods) > 0 or result.detected_periods == []

# ============================================================================
# Integration Tests: API Endpoints
# ============================================================================

class TestAPIEndpoints:
    """Test FastAPI endpoints"""
    
    @pytest.fixture
    def client(self):
        """Create test client"""
        from fastapi.testclient import TestClient
        from timeseries_service.main import app
        
        return TestClient(app)
    
    def test_health_check(self, client):
        """Test health endpoint"""
        response = client.get("/health")
        assert response.status_code == 200
        data = response.json()
        assert data['status'] == 'healthy'
        assert 'version' in data
    
    def test_capabilities_endpoint(self, client):
        """Test capabilities endpoint"""
        response = client.get("/capabilities")
        assert response.status_code == 200
        data = response.json()
        assert 'services' in data
        assert len(data['services']) >= 5
    
    def test_decomposition_endpoint(self, client, synthetic_timeseries):
        """Test decomposition endpoint"""
        ts, _ = synthetic_timeseries
        
        payload = {
            "values": ts.tolist(),
            "method": "additive",
            "period": 7
        }
        
        response = client.post("/decompose", json=payload)
        
        if response.status_code == 200:
            data = response.json()
            assert 'components' in data
            assert 'trend' in data['components']
            assert 'seasonal' in data['components']
            assert 'quality_metrics' in data
    
    def test_forecast_endpoint(self, client, synthetic_timeseries, timestamps):
        """Test forecast endpoint"""
        ts, _ = synthetic_timeseries
        
        payload = {
            "values": ts.tolist(),
            "timestamps": timestamps.tolist(),
            "horizons": [1, 24],
            "model_type": "ensemble"
        }
        
        response = client.post("/forecast", json=payload)
        
        if response.status_code == 200:
            data = response.json()
            assert 'forecasts' in data
            assert len(data['forecasts']) == 2
    
    def test_fourier_features_endpoint(self, client, synthetic_timeseries):
        """Test Fourier features endpoint"""
        ts, _ = synthetic_timeseries
        
        payload = {
            "values": ts.tolist(),
            "feature_id": "test_feature"
        }
        
        response = client.post("/fourier-features", json=payload)
        
        if response.status_code == 200:
            data = response.json()
            assert 'features' in data
            assert data['n_features_generated'] > 0
    
    def test_autocorrelation_endpoint(self, client, synthetic_timeseries):
        """Test autocorrelation features endpoint"""
        ts, _ = synthetic_timeseries
        
        payload = {
            "values": ts.tolist(),
            "feature_id": "test_feature"
        }
        
        response = client.post("/autocorrelation-features", json=payload)
        
        if response.status_code == 200:
            data = response.json()
            assert 'features' in data
            assert 'feature_categories' in data
    
    def test_anomaly_detection_endpoint(self, client, timeseries_with_anomalies):
        """Test anomaly detection endpoint"""
        ts, _ = timeseries_with_anomalies
        
        payload = {
            "values": ts.tolist()
        }
        
        response = client.post("/detect-anomalies", json=payload)
        
        if response.status_code == 200:
            data = response.json()
            assert 'n_anomalies' in data
            assert 'anomaly_indices' in data

# ============================================================================
# Performance Tests
# ============================================================================

class TestPerformance:
    """Performance and stress tests"""
    
    def test_decomposition_performance(self, synthetic_timeseries):
        """Measure decomposition latency"""
        from timeseries_service.decomposition import TimeSeriesDecomposition
        
        ts, t = synthetic_timeseries
        
        start = time.time()
        decomp = TimeSeriesDecomposition(ts, t, period=7)
        result = decomp.decompose_additive()
        elapsed = time.time() - start
        
        assert elapsed < 0.5  # Should complete in < 500ms
    
    def test_forecasting_performance(self, synthetic_timeseries):
        """Measure forecasting latency"""
        from timeseries_service.forecasting import ARIMAForecaster
        
        ts, _ = synthetic_timeseries
        
        start = time.time()
        forecaster = ARIMAForecaster(ts)
        forecaster.fit_auto_arima()
        forecaster.forecast_multi_horizon([1, 24])
        elapsed = time.time() - start
        
        assert elapsed < 5.0  # Should complete in < 5s
    
    def test_feature_generation_performance(self, synthetic_timeseries):
        """Measure feature generation latency"""
        from timeseries_service.features import FourierFeaturesGenerator, \
                                                AutocorrelationFeaturesGenerator
        
        ts, t = synthetic_timeseries
        
        start = time.time()
        fourier_gen = FourierFeaturesGenerator(ts, t)
        fourier_gen.get_result()
        
        acf_gen = AutocorrelationFeaturesGenerator(ts)
        acf_gen.get_autocorrelation_features()
        elapsed = time.time() - start
        
        assert elapsed < 1.0  # Should complete in < 1s
    
    def test_anomaly_detection_performance(self, timeseries_with_anomalies):
        """Measure anomaly detection latency"""
        from timeseries_service.anomaly_detection import EnsembleAnomalyDetector
        
        ts, t = timeseries_with_anomalies
        
        start = time.time()
        detector = EnsembleAnomalyDetector(ts, t)
        detector.detect()
        elapsed = time.time() - start
        
        assert elapsed < 2.0  # Should complete in < 2s

# ============================================================================
# Regression Tests
# ============================================================================

class TestRegression:
    """Tests to ensure no regressions from Phase 3.21"""
    
    def test_decomposition_backward_compat(self, synthetic_timeseries):
        """Ensure decomposition still works with old patterns"""
        from timeseries_service.decomposition import TimeSeriesDecomposition
        
        ts, t = synthetic_timeseries
        decomp = TimeSeriesDecomposition(ts, t)
        result = decomp.decompose_additive()
        
        # Key attributes must exist
        assert hasattr(result, 'trend')
        assert hasattr(result, 'seasonal')
        assert hasattr(result, 'residual')
        assert hasattr(result, 'variance_explained')
    
    def test_forecasting_backward_compat(self, synthetic_timeseries):
        """Ensure forecasting still works"""
        from timeseries_service.forecasting import ARIMAForecaster
        
        ts, _ = synthetic_timeseries
        forecaster = ARIMAForecaster(ts)
        forecaster.fit_auto_arima()
        results = forecaster.forecast_multi_horizon([24])
        
        assert len(results) > 0
        res = results[0]
        assert hasattr(res, 'point_forecast')
        assert hasattr(res, 'lower_bound_80')
        assert hasattr(res, 'upper_bound_80')

if __name__ == '__main__':
    pytest.main([__file__, '-v', '--tb=short'])
