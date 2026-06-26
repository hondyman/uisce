# Phase 3.22: Advanced Time-Series Feature Engineering

**Status:** 🟡 In Development  
**Start Date:** February 10, 2026  
**Target Completion:** February 15, 2026  
**Estimated LOC:** 1,500+ Python, 600+ YAML, 2,000+ documentation

---

## Overview

Phase 3.22 extends Phase 3.21 (Feature Engineering Platform) with **advanced time-series feature capabilities**:

1. **Time-Series Decomposition Service** - Trend, seasonality, residual extraction
2. **ARIMA/Prophet Integration** - Statistical forecasting & intervals
3. **Fourier Features** - Periodic pattern capture (sin/cos harmonics)
4. **Changepoint Detection** - Identify structural breaks in time series
5. **Autocorrelation Features** - Lag-based and ACF/PACF metrics
6. **Spectral Analysis** - Frequency domain features via FFT
7. **Multi-Horizon Forecasting** - 1h, 1d, 7d, 30d ahead predictions
8. **Time-Series Anomaly Detection** - Isolation Forest on decomposed components

---

## Architecture

```
Feature Time-Series (from Phase 3.21 Materialization)
    │
    ├─► Time-Series Decomposition Service (FastAPI)
    │   ├─ Additive/Multiplicative decomposition
    │   ├─ Trend extraction (loess, moving average)
    │   ├─ Seasonality extraction (seasonal decompose)
    │   └─ Residual analysis (outlier detection)
    │
    ├─► ARIMA/Prophet Service (Python)
    │   ├─ Auto ARIMA parameter selection
    │   ├─ Prophet model fitting
    │   ├─ Confidence intervals (80%, 95%)
    │   └─ Multi-horizon forecasts
    │
    ├─► Fourier Features Service (Python/NumPy)
    │   ├─ Yearly seasonality (sin/cos, 365 days)
    │   ├─ Weekly seasonality (sin/cos, 7 days)
    │   ├─ Daily seasonality (sin/cos, 24 hours)
    │   └─ Custom frequencies
    │
    ├─► Autocorrelation Service
    │   ├─ Lag features (t-1, t-7, t-30)
    │   ├─ ACF/PACF features
    │   └─ Partial autocorrelation plots
    │
    └─► Feature Store (PostgreSQL Phase 3.21)
        ├─ Decomposed components table
        ├─ Forecast intervals table
        ├─ Fourier features table
        ├─ Anomaly scores table
        └─ Time-series lineage table
```

---

## Detailed Specifications

### 1. Time-Series Decomposition Service

**Purpose:** Extract trend, seasonality, and residual components from time-series features

**Decomposition Methods:**

```python
class TimeSeriesDecomposition:
    """Decompose time-series into trend, seasonality, residual"""
    
    def decompose_additive(timeseries, period=7):
        """
        Additive decomposition: y(t) = trend(t) + seasonal(t) + residual(t)
        
        Best for: Constant seasonal magnitude
        Example: Sales data with fixed weekly pattern
        
        Returns:
            trend: Smoothed trend component
            seasonal: Periodic component (repeats every 'period')
            residual: What's left (noise + anomalies)
        """
        # Uses: seasonal_decompose with period parameter
        # Trend: Moving average or loess smoothing
        # Seasonal: Average value at each position in cycle
        pass
    
    def decompose_multiplicative(timeseries, period=7):
        """
        Multiplicative decomposition: y(t) = trend(t) * seasonal(t) * residual(t)
        
        Best for: Growing/shrinking seasonal magnitude
        Example: Revenue with increasing seasonality
        
        Returns:
            trend: Smoothed trend component
            seasonal: Multiplicative factors (1.0 = normal)
            residual: Ratio of actual to trend*seasonal
        """
        pass
    
    def decompose_robust(timeseries, period=7):
        """
        Robust decomposition: Resistant to outliers
        
        Uses: Robust loess (LOWESS) instead of standard MA
        Advantage: Outliers don't distort trend
        
        Returns:
            trend: Robust trend
            seasonal: Seasonal component
            residual: Robust residuals
        """
        pass
```

**API Endpoints:**

```yaml
POST /api/v1/timeseries/decompose
├─ Request: 
│  ├─ feature_id: "feature:revenue_v1"
│  ├─ method: "additive" | "multiplicative" | "robust"
│  ├─ period: 7  # Weekly seasonality
│  └─ window_size: 90  # Analyze last 90 days
├─ Returns:
│  ├─ trend: [values for each timestamp]
│  ├─ seasonal: [seasonal component]
│  ├─ residual: [residuals/anomalies]
│  ├─ decomposition_quality: 0.95  # R² of fit
│  └─ recommended_period: 7  # Best period detected
└─ Async: Stores decomposition to feature_decomposition table

GET /api/v1/timeseries/decompose/{feature_id}?days=30
├─ Returns: Cached decomposition results
└─ Cache: Updated daily via Temporal workflow

GET /api/v1/timeseries/residuals/{feature_id}
├─ Returns: Top anomalies (>2σ from mean)
├─ Anomalies: Potential data quality issues
└─ Use: Alert on abnormal residuals
```

**Database Schema:**

```sql
CREATE TABLE feature_timeseries_decomposition (
    id UUID PRIMARY KEY,
    feature_id VARCHAR NOT NULL,
    decomposition_date DATE NOT NULL,
    method VARCHAR,  -- 'additive', 'multiplicative', 'robust'
    period INT,  -- Seasonality period (7 for weekly)
    
    -- Decomposed components (JSONB for flexibility)
    trend_values FLOAT8[],  -- Smoothed trend
    seasonal_values FLOAT8[],  -- Seasonal component
    residual_values FLOAT8[],  -- Residuals/anomalies
    
    -- Quality metrics
    variance_explained FLOAT8,  -- R² of decomposition
    residual_std FLOAT8,  -- Standard deviation of residuals
    autocorrelation_residual FLOAT8,  -- ACF at lag-1
    
    -- Detected properties
    detected_period INT,  -- Detected seasonality period
    trend_direction VARCHAR,  -- 'up', 'down', 'stable'
    has_anomalies BOOLEAN,
    
    computed_at TIMESTAMP DEFAULT NOW(),
    region VARCHAR,
    tenant_id VARCHAR,
    
    CONSTRAINT fk_feature FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id)
);

CREATE INDEX idx_timeseries_decomp_feature 
    ON feature_timeseries_decomposition(feature_id, decomposition_date DESC);
CREATE INDEX idx_timeseries_decomp_region 
    ON feature_timeseries_decomposition(region, tenant_id, decomposition_date DESC);
```

---

### 2. ARIMA/Prophet Forecasting Service

**Purpose:** Fit ARIMA and Prophet models for multi-horizon forecasting

**Models:**

```python
class ARIMAForecaster:
    """
    Auto ARIMA parameter selection + forecasting
    
    ARIMA(p,d,q):
    - p: Auto-regressive order (lag terms)
    - d: Differencing order (trend removal)
    - q: Moving average order
    
    Strategy:
    1. Auto-select (p,d,q) using AIC/BIC
    2. Fit on last 90 days of data
    3. Forecast 1h, 1d, 7d, 30d ahead
    4. Return confidence intervals (80%, 95%)
    """
    
    def fit_auto_arima(timeseries, max_p=5, max_d=2, max_q=5):
        """
        Auto ARIMA: Search parameter space for best fit
        
        Returns: ARIMA(p, d, q) model
        """
        pass
    
    def forecast_horizon(model, horizon_hours=[1, 24, 168, 720]):
        """
        Multi-horizon forecasting
        
        horizons:
        - 1h: Immediate trends
        - 24h (1d): Daily patterns
        - 168h (1 week): Weekly seasonality
        - 720h (30 days): Monthly trends
        
        Returns:
            forecast: Point estimates
            conf_lower_80: 80% lower bound
            conf_upper_80: 80% upper bound
            conf_lower_95: 95% lower bound
            conf_upper_95: 95% upper bound
        """
        pass

class ProphetForecaster:
    """
    Facebook Prophet: Additive model with seasonality
    
    y(t) = trend(t) + seasonality(t) + holidays + residual
    
    Advantages:
    - Handles missing data
    - Built-in holiday/event handling
    - Automatic changepoint detection
    - Interpretable components
    """
    
    def fit_prophet(timeseries, yearly_seasonality=True, 
                    weekly_seasonality=True, daily_seasonality=False):
        """
        Fit Prophet model
        
        Parameters:
        - interval_width: 0.95 → 95% confidence interval
        - changepoint_prior_scale: 0.05 (flexibility of trend changes)
        - seasonality_prior_scale: 10.0 (strength of seasonal component)
        """
        pass
    
    def forecast_with_intervals(model, periods=720):
        """
        Forecast with automatic confidence intervals
        
        Returns:
            yhat: Point forecast
            yhat_lower: 95% lower bound
            yhat_upper: 95% upper bound
            trend: Trend component
            seasonal: Seasonal component
        """
        pass
    
    def detect_anomalies(forecast_results, residuals):
        """
        Identify anomalies as points >3σ from forecast
        """
        pass
```

**API Endpoints:**

```yaml
POST /api/v1/timeseries/forecast
├─ Request:
│  ├─ feature_id: "feature:revenue_v1"
│  ├─ model: "arima" | "prophet" | "ensemble"
│  ├─ horizons: [1, 24, 168, 720]  # hours
│  └─ confidence_level: 0.95
├─ Returns:
│  ├─ forecasts: {
│  │    "1h": {"point": 1050, "lower": 1000, "upper": 1100},
│  │    "24h": {"point": 23000, "lower": 22000, "upper": 24000},
│  │    ...
│  │  }
│  ├─ model_fit_quality: 0.92
│  ├─ trend: "increasing"
│  └─ seasonal_strength: 0.78
└─ Async: Stores to feature_forecasts table

GET /api/v1/timeseries/forecast/{feature_id}?horizon=24h
├─ Returns: Latest forecast for 24-hour horizon
└─ Cache: Updated every 4 hours

GET /api/v1/timeseries/forecast-accuracy/{feature_id}?days=30
├─ Returns: RMSE, MAE, MAPE on last 30 days
├─ Use: Model performance tracking
└─ Alerts: If accuracy drops >20%
```

**Database Schema:**

```sql
CREATE TABLE feature_forecasts (
    id UUID PRIMARY KEY,
    feature_id VARCHAR NOT NULL,
    forecast_timestamp TIMESTAMP,  -- When forecast was generated
    horizon_hours INT,  -- 1, 24, 168, 720
    
    -- Forecast results
    point_forecast FLOAT8,
    confidence_level FLOAT8,  -- 0.80 or 0.95
    lower_bound FLOAT8,
    upper_bound FLOAT8,
    
    -- Model details
    model_type VARCHAR,  -- 'arima', 'prophet'
    arima_params VARCHAR,  -- "(1,1,1)"
    prophet_components JSONB,  -- trend, seasonal, holiday
    
    -- Quality metrics
    model_rmse FLOAT8,
    model_mae FLOAT8,
    model_mape FLOAT8,
    fit_quality FLOAT8,  -- 0-1
    
    -- Actual vs forecast (filled later)
    actual_value FLOAT8,
    absolute_error FLOAT8,
    percentage_error FLOAT8,
    
    computed_at TIMESTAMP DEFAULT NOW(),
    region VARCHAR,
    tenant_id VARCHAR,
    
    CONSTRAINT fk_feature FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id)
);

CREATE INDEX idx_forecast_feature_horizon 
    ON feature_forecasts(feature_id, horizon_hours, forecast_timestamp DESC);
```

---

### 3. Fourier Features Service

**Purpose:** Generate periodic features via Fourier series expansion

**Fourier Features:**

```python
class FourierFeatures:
    """
    Capture periodic patterns using sin/cos harmonics
    
    Why Fourier:
    - Linear models can't capture periodicity directly
    - sin/cos features enable ML models to learn cycles
    - More flexible than dummy variables
    - Fewer features than one-hot encoding
    
    Fourier series: f(t) = a₀ + Σ(aₙ*cos(nt) + bₙ*sin(nt))
    
    For time series with frequency f (e.g., 1/365 for yearly):
    - sin(2π*f*t), cos(2π*f*t)  [fundamental]
    - sin(4π*f*t), cos(4π*f*t)  [2nd harmonic]
    - sin(6π*f*t), cos(6π*f*t)  [3rd harmonic]
    - ...
    """
    
    def generate_fourier_features(timestamps, frequencies, num_harmonics=3):
        """
        Generate Fourier features for multiple seasonal periods
        
        Parameters:
        - frequencies: [1/365, 1/7, 1/1]  # yearly, weekly, daily
        - num_harmonics: 3 (use up to 3rd harmonic)
        
        Returns: DataFrame with columns
        - sin_yearly_1, cos_yearly_1, sin_yearly_2, cos_yearly_2, sin_yearly_3, cos_yearly_3
        - sin_weekly_1, cos_weekly_1, ...
        - sin_daily_1, cos_daily_1, ...
        """
        features = pd.DataFrame(index=timestamps)
        
        for freq_name, frequency in frequencies.items():
            for harmonic in range(1, num_harmonics + 1):
                angle = 2 * np.pi * harmonic * frequency * timestamps
                features[f'sin_{freq_name}_{harmonic}'] = np.sin(angle)
                features[f'cos_{freq_name}_{harmonic}'] = np.cos(angle)
        
        return features
    
    def detect_seasonalities(timeseries):
        """
        Auto-detect dominant seasonalities (periods)
        
        Uses: FFT (Fast Fourier Transform)
        - Top 3 frequencies = dominant periods
        
        Returns: [(period, strength), ...]
        """
        fft = np.fft.fft(timeseries)
        power = np.abs(fft) ** 2
        top_freqs = np.argsort(power)[-3:]
        
        periods = [(len(timeseries) / freq, power[freq]) for freq in top_freqs]
        return periods
```

**API Endpoints:**

```yaml
POST /api/v1/timeseries/fourier-features
├─ Request:
│  ├─ feature_id: "feature:revenue_v1"
│  ├─ frequencies: ["yearly", "weekly", "daily"]
│  └─ num_harmonics: 3
├─ Returns:
│  ├─ features: DataFrame with sin/cos columns
│  ├─ detected_periods: [365.2, 7.0, 1.0]
│  └─ feature_importance: [0.6, 0.3, 0.1]  # by variance explained
└─ Use: Feed to ML models for seasonality capture

GET /api/v1/timeseries/seasonality/{feature_id}
├─ Returns: Most prominent seasonal period
├─ Example: {"period_days": 7, "strength": 0.78, "harmonics": 3}
└─ Cache: Updated weekly
```

**Example Output:**

```
Index     sin_yearly_1  cos_yearly_1  sin_weekly_1  cos_weekly_1
2026-02-01    0.342        0.939         0.782        0.623
2026-02-02    0.355        0.935         0.901        0.434
...
```

---

### 4. Autocorrelation Features Service

**Purpose:** Generate lag-based and correlation features

**Features:**

```python
class AutocorrelationFeatures:
    """
    Capture dependency on past values
    
    Lag features: Earlier values in series
    - lag_1: y(t-1)
    - lag_7: y(t-7)  # Weekly lag
    - lag_30: y(t-30)  # Monthly lag
    
    ACF/PACF: Correlation with past
    - acf_lag1: Autocorrelation at lag 1
    - pacf_lag1: Partial autocorrelation at lag 1
    """
    
    def create_lag_features(timeseries, lags=[1, 7, 14, 30]):
        """
        Create lagged versions of the series
        
        Returns: DataFrame with columns lag_1, lag_7, lag_14, lag_30
        """
        df = pd.DataFrame({'value': timeseries})
        for lag in lags:
            df[f'lag_{lag}'] = df['value'].shift(lag)
        return df.drop('value', axis=1)
    
    def create_rolling_features(timeseries, windows=[7, 14, 30]):
        """
        Rolling statistical features
        
        Returns: mean, std, min, max for each window
        """
        features = pd.DataFrame()
        for window in windows:
            features[f'rolling_mean_{window}'] = timeseries.rolling(window).mean()
            features[f'rolling_std_{window}'] = timeseries.rolling(window).std()
            features[f'rolling_min_{window}'] = timeseries.rolling(window).min()
            features[f'rolling_max_{window}'] = timeseries.rolling(window).max()
        return features
    
    def compute_acf_features(timeseries, max_lag=30):
        """
        Autocorrelation Function features
        
        acf(k) = correlation(y(t), y(t-k))
        
        Returns: ACF values at lags 1, 7, 14, 30
        """
        from statsmodels.graphics.tsaplots import acf
        acf_values = acf(timeseries, nlags=max_lag)
        return {f'acf_lag_{lag}': acf_values[lag] 
                for lag in [1, 7, 14, 30]}
    
    def compute_pacf_features(timeseries, max_lag=30):
        """
        Partial Autocorrelation Function features
        
        Correlation after removing intervening influences
        """
        from statsmodels.graphics.tsaplots import pacf
        pacf_values = pacf(timeseries, nlags=max_lag)
        return {f'pacf_lag_{lag}': pacf_values[lag] 
                for lag in [1, 7, 14, 30]}
```

**Database Schema:**

```sql
CREATE TABLE feature_timeseries_features (
    id UUID PRIMARY KEY,
    feature_id VARCHAR NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    
    -- Lag features
    lag_1 FLOAT8, lag_7 FLOAT8, lag_14 FLOAT8, lag_30 FLOAT8,
    
    -- Rolling statistics
    rolling_mean_7 FLOAT8, rolling_std_7 FLOAT8,
    rolling_mean_30 FLOAT8, rolling_std_30 FLOAT8,
    
    -- Autocorrelation features
    acf_lag_1 FLOAT8, acf_lag_7 FLOAT8, acf_lag_14 FLOAT8, acf_lag_30 FLOAT8,
    pacf_lag_1 FLOAT8, pacf_lag_7 FLOAT8, pacf_lag_14 FLOAT8, pacf_lag_30 FLOAT8,
    
    -- Fourier features (stored as JSON for flexibility)
    fourier_features JSONB,  -- {sin_yearly_1: 0.34, cos_yearly_1: 0.93, ...}
    
    region VARCHAR,
    tenant_id VARCHAR,
    
    CONSTRAINT fk_feature FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id)
);

CREATE INDEX idx_timeseries_features_feature_timestamp 
    ON feature_timeseries_features(feature_id, timestamp DESC);
```

---

### 5. Time-Series Anomaly Detection

**Purpose:** Detect unusual patterns in decomposed components

**Anomaly Detection:**

```python
class TimeSeriesAnomalyDetector:
    """
    Detect anomalies in time-series residuals
    """
    
    def detect_statistical_anomalies(residuals, threshold_sigma=3):
        """
        Simple: Points >threshold_sigma from mean
        
        Anomalies: residual > mean + threshold_sigma * std
        """
        mean = residuals.mean()
        std = residuals.std()
        anomalies = np.abs(residuals - mean) > (threshold_sigma * std)
        return anomalies, residuals[anomalies]
    
    def detect_isolation_forest_anomalies(residuals, contamination=0.05):
        """
        Advanced: Isolation Forest (unsupervised)
        
        Good for: Multivariate anomalies
        """
        from sklearn.ensemble import IsolationForest
        detector = IsolationForest(contamination=contamination)
        anomaly_scores = detector.fit_predict(residuals.reshape(-1, 1))
        return anomaly_scores == -1
    
    def detect_dbscan_anomalies(residuals, eps=1.5, min_samples=5):
        """
        DBSCAN clustering: Outliers are points not in clusters
        """
        from sklearn.cluster import DBSCAN
        clustering = DBSCAN(eps=eps, min_samples=min_samples).fit(residuals.reshape(-1, 1))
        return clustering.labels_ == -1  # -1 = outlier label
```

---

## Implementation Roadmap

### Week 1: Core Services

**Day 1-2: Time-Series Decomposition Service**
- [ ] Implement decompose_additive, multiplicative, robust
- [ ] FastAPI endpoints
- [ ] PostgreSQL schema
- [ ] Unit tests (decomposition accuracy)

**Day 2-3: ARIMA/Prophet Service**
- [ ] Auto ARIMA parameter selection
- [ ] Prophet model fitting
- [ ] Multi-horizon forecasting
- [ ] Confidence interval computation
- [ ] Unit tests (forecast accuracy)

**Day 3-4: Fourier Features Service**
- [ ] Generate Fourier features (yearly, weekly, daily)
- [ ] Auto-detect dominant seasonalities (FFT)
- [ ] Feature importance scoring
- [ ] Unit tests

**Day 4-5: Autocorrelation & Anomaly Detection**
- [ ] Lag features, rolling statistics
- [ ] ACF/PACF computation
- [ ] Statistical + Isolation Forest anomaly detection
- [ ] Unit tests

### Week 2: Integration & Deployment

**Day 6-7: Kubernetes Deployment**
- [ ] Create Kubernetes manifests
- [ ] HPA configuration (CPU/memory targets)
- [ ] ConfigMaps & Secrets
- [ ] Health check endpoints

**Day 7-8: Monitoring & Alerting**
- [ ] Prometheus metrics (forecast accuracy, decomposition quality)
- [ ] Grafana dashboards (5 panels for time-series metrics)
- [ ] Alert rules (7 alerts for anomalies, forecast drift, decomposition issues)
- [ ] SLO targets

**Day 8: CI/CD & Testing**
- [ ] Extend GitHub Actions pipeline (new stages)
- [ ] E2E integration tests (~25 tests)
- [ ] Performance benchmarks
- [ ] Documentation

---

## Success Criteria

| Criterion | Target | Status |
|-----------|--------|--------|
| **Decomposition Accuracy** | R² > 0.90 on test features | ⏳ |
| **Forecast RMSE** | <5% of feature mean | ⏳ |
| **API Latency** | <2 seconds | ⏳ |
| **Fourier Features** | Capture 80%+ seasonality variance | ⏳ |
| **Anomaly Detection** | >90% precision, >80% recall | ⏳ |
| **Test Coverage** | >80% | ⏳ |
| **Deployment** | Kubernetes HA (3-10 replicas) | ⏳ |
| **Documentation** | >2,000 lines with examples | ⏳ |

---

## Expected Deliverables

### Code (1,500+ LOC)
- [ ] `timeseries_service/` - All 5 components
- [ ] `timeseries_service/decomposition.py` (300+ lines)
- [ ] `timeseries_service/forecasting.py` (400+ lines)
- [ ] `timeseries_service/fourier.py` (200+ lines)
- [ ] `timeseries_service/autocorrelation.py` (200+ lines)
- [ ] `timeseries_service/anomaly_detection.py` (200+ lines)

### Infrastructure (600+ YAML)
- [ ] `k8s/timeseries-deployment.yaml`
- [ ] `k8s/timeseries-config.yaml`
- [ ] `k8s/timeseries-monitoring.yaml`

### Testing (500+ LOC)
- [ ] `tests/integration/test_phase_3_22.py`
- [ ] Performance benchmarks
- [ ] Accuracy validation tests

### Documentation (2,000+ lines)
- [ ] `PHASE_3_22_COMPLETE.md`
- [ ] Algorithm deep-dives
- [ ] Deployment guide
- [ ] API reference

---

## Phase 3.21 Integration Points

**Using Phase 3.21 Features:**
- Feature values from `feature_catalog` & materialization
- Drift detection to identify when to refit models
- Feature importance to weight seasonal patterns
- PostgreSQL for storing decomposition results
- Prometheus for metrics
- Grafana for dashboards

**Enhancing Phase 3.21:**
- Better feature freshness prediction (via forecasts)
- Anomaly detection complements drift detection
- Fourier + importance = better feature engineering
- Seasonal understanding improves materialization quality

---

## Continuous Integration

**CI/CD Stages (Add to Phase 3.21 pipeline):**
1. Unit tests (decomposition, forecasting)
2. Stat tests (accuracy benchmarks)
3. Integration tests (E2E with PostgreSQL)
4. Performance tests (latency <2s)
5. Security scan (same as Phase 3.21)
6. Build Docker image
7. Deploy to staging
8. Smoke test endpoints

---

## Next Phases (3.23+)

**Phase 3.23:** Automated Feature Discovery
- Top 500+ features discovery
- Genetic algorithm based selection
- AutoML integration

**Phase 3.24:** Global Distribution
- Multi-region routing
- Cross-region drift correlation
- Federated learning

---

**Ready to begin Phase 3.22 implementation?** ✅

Next: Create decomposition service with FastAPI endpoints
