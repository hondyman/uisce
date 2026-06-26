#!/usr/bin/env python3
"""
FastAPI Service for Time-Series Features
Provides HTTP endpoints for decomposition, forecasting, Fourier, lags, and anomaly detection
"""

from fastapi import FastAPI, HTTPException, Query
from fastapi.responses import JSONResponse
import numpy as np
import pandas as pd
from typing import List, Dict, Optional
import logging
import uvicorn
from datetime import datetime, timedelta
import json

# Import all services
from decomposition import TimeSeriesDecomposition, DecompositionResult
from forecasting import ARIMAForecaster, ProphetForecaster, EnsembleForecaster, ForecastResult
from features import FourierFeaturesGenerator, AutocorrelationFeaturesGenerator
from anomaly_detection import EnsembleAnomalyDetector

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="Time-Series Features Service",
    description="Advanced time-series feature engineering and analysis",
    version="3.22.0"
)

# ============================================================================
# Data Models
# ============================================================================

from pydantic import BaseModel
from typing import Optional

class TimeSeriesData(BaseModel):
    """Request payload for time-series analysis"""
    values: List[float]
    timestamps: Optional[List[str]] = None
    feature_id: Optional[str] = None
    metadata: Optional[Dict] = None

class DecompositionRequest(BaseModel):
    """Request for time-series decomposition"""
    values: List[float]
    method: str = "additive"  # additive, multiplicative, robust
    period: Optional[int] = None

class ForecastRequest(BaseModel):
    """Request for forecasting"""
    values: List[float]
    timestamps: Optional[List[str]] = None
    horizons: List[int] = [1, 24, 168, 720]
    model_type: str = "ensemble"  # arima, prophet, ensemble

class AnomalyRequest(BaseModel):
    """Request for anomaly detection"""
    values: List[float]
    timestamps: Optional[List[str]] = None

# ============================================================================
# Helper Functions
# ============================================================================

def parse_timestamps(timestamps: Optional[List[str]]) -> np.ndarray:
    """Convert timestamp strings to datetime array"""
    if timestamps is None:
        return None
    try:
        dates = pd.to_datetime(timestamps)
        return dates.values
    except:
        return None

def numpy_to_json(obj):
    """Convert numpy types to JSON-serializable types"""
    if isinstance(obj, np.ndarray):
        return obj.tolist()
    elif isinstance(obj, (np.integer, np.floating)):
        return float(obj)
    elif isinstance(obj, dict):
        return {k: numpy_to_json(v) for k, v in obj.items()}
    elif isinstance(obj, (list, tuple)):
        return [numpy_to_json(v) for v in obj]
    return obj

# ============================================================================
# Decomposition Endpoints
# ============================================================================

@app.post("/decompose")
async def decompose(request: DecompositionRequest):
    """
    Decompose time-series into trend, seasonal, and residual components
    
    Methods:
    - additive: y(t) = trend + seasonal + residual
    - multiplicative: y(t) = trend × seasonal × residual
    - robust: LOWESS-based, resistant to outliers
    """
    try:
        ts = np.array(request.values, dtype=np.float64)
        
        if len(ts) < 10:
            raise ValueError("Time-series must have at least 10 points")
        
        # Auto-detect period if not provided
        period = request.period or max(7, len(ts) // 52)
        
        # Decompose
        decomp = TimeSeriesDecomposition(ts, np.arange(len(ts)), period=period)
        
        if request.method == "multiplicative":
            result = decomp.decompose_multiplicative()
        elif request.method == "robust":
            result = decomp.decompose_robust()
        else:
            result = decomp.decompose_additive()
        
        # Return decomposed components
        return JSONResponse({
            "feature_id": request.feature_id or "unknown",
            "method": request.method,
            "period": period,
            "components": {
                "trend": numpy_to_json(result.trend),
                "seasonal": numpy_to_json(result.seasonal),
                "residual": numpy_to_json(result.residual)
            },
            "quality_metrics": {
                "variance_explained_r2": float(result.variance_explained),
                "residual_std": float(result.residual_std),
                "has_anomalies": result.has_anomalies,
                "n_anomalies": int(len(result.anomaly_indices)) 
                    if result.anomaly_indices is not None else 0
            },
            "timestamp": datetime.utcnow().isoformat()
        })
    
    except Exception as e:
        logger.error(f"Decomposition failed: {e}")
        raise HTTPException(status_code=400, detail=str(e))

@app.get("/decompose/{feature_id}")
async def get_decomposition(feature_id: str):
    """
    Retrieve cached decomposition for a feature
    (Mock endpoint - in production would query database)
    """
    return {
        "feature_id": feature_id,
        "message": "Decomposition retrieval not implemented (DB integration needed)"
    }

@app.get("/residuals/{feature_id}")
async def get_residuals_anomalies(feature_id: str):
    """Get anomalies detected in decomposition residuals"""
    return {
        "feature_id": feature_id,
        "message": "Residual anomaly retrieval not implemented (DB integration needed)"
    }

# ============================================================================
# Forecasting Endpoints
# ============================================================================

@app.post("/forecast")
async def forecast(request: ForecastRequest):
    """
    Multi-horizon forecasting with confidence intervals
    
    Models:
    - arima: Auto ARIMA with AIC/BIC selection
    - prophet: Additive model with seasonality
    - ensemble: Combined ARIMA + Prophet
    """
    try:
        ts = np.array(request.values, dtype=np.float64)
        timestamps = parse_timestamps(request.timestamps)
        
        if len(ts) < 20:
            raise ValueError("Time-series must have at least 20 points for forecasting")
        
        results = []
        
        if request.model_type == "arima":
            forecaster = ARIMAForecaster(ts)
            forecaster.fit_auto_arima()
            forecast_results = forecaster.forecast_multi_horizon(request.horizons)
            model_type = "arima"
        
        elif request.model_type == "prophet":
            if timestamps is None:
                raise ValueError("Prophet requires timestamps")
            forecaster = ProphetForecaster(ts, timestamps)
            forecaster.fit_model()
            forecast_results = forecaster.forecast_multi_horizon(request.horizons)
            model_type = "prophet"
        
        else:  # ensemble
            if timestamps is None:
                # Prophet fallback with generated timestamps
                base_date = datetime.utcnow() - timedelta(days=len(ts))
                timestamps = np.array([
                    (base_date + timedelta(hours=i)).isoformat() 
                    for i in range(len(ts))
                ])
            forecaster = EnsembleForecaster(ts, timestamps)
            forecaster.fit()
            forecast_results = forecaster.forecast_multi_horizon(request.horizons)
            model_type = "ensemble"
        
        # Convert results to JSON
        for res in forecast_results:
            results.append({
                "horizon_hours": res.horizon_hours,
                "point_forecast": float(res.point_forecast),
                "confidence_80": {
                    "lower": float(res.lower_bound_80),
                    "upper": float(res.upper_bound_80)
                },
                "confidence_95": {
                    "lower": float(res.lower_bound_95),
                    "upper": float(res.upper_bound_95)
                },
                "accuracy_metrics": {
                    "rmse": float(res.rmse) if res.rmse else None,
                    "mae": float(res.mae) if res.mae else None,
                    "mape": float(res.mape) if res.mape else None
                }
            })
        
        return JSONResponse({
            "feature_id": request.feature_id or "unknown",
            "model_type": model_type,
            "n_historical_points": len(ts),
            "forecasts": results,
            "timestamp": datetime.utcnow().isoformat()
        })
    
    except Exception as e:
        logger.error(f"Forecast failed: {e}")
        raise HTTPException(status_code=400, detail=str(e))

@app.get("/forecast/{feature_id}")
async def get_forecast(feature_id: str, horizon: Optional[int] = Query(24)):
    """Retrieve cached forecast for a feature"""
    return {
        "feature_id": feature_id,
        "horizon": horizon,
        "message": "Forecast retrieval not implemented (DB integration needed)"
    }

@app.get("/forecast-accuracy/{feature_id}")
async def get_forecast_accuracy(feature_id: str):
    """Get forecast accuracy metrics"""
    return {
        "feature_id": feature_id,
        "message": "Forecast accuracy retrieval not implemented (DB integration needed)"
    }

# ============================================================================
# Fourier Features Endpoints
# ============================================================================

@app.post("/fourier-features")
async def generate_fourier_features(request: TimeSeriesData):
    """
    Generate Fourier (sin/cos) features for periodic patterns
    
    Generates harmonics for yearly, weekly, and optionally daily seasonality
    """
    try:
        ts = np.array(request.values, dtype=np.float64)
        
        if len(ts) < 20:
            raise ValueError("Time-series must have at least 20 points")
        
        gen = FourierFeaturesGenerator(ts, np.arange(len(ts)))
        result = gen.get_result(num_harmonics=3)
        
        # Convert features to list
        features_list = result.features_df.to_dict('list')
        
        return JSONResponse({
            "feature_id": request.feature_id or "unknown",
            "n_features_generated": len(features_list),
            "features": {k: [float(v) for v in vals] for k, vals in features_list.items()},
            "detected_periods": [
                {
                    "period": float(period),
                    "strength": float(strength)
                }
                for period, strength in result.detected_periods
            ],
            "dominant_period": float(result.dominant_period),
            "timestamp": datetime.utcnow().isoformat()
        })
    
    except Exception as e:
        logger.error(f"Fourier feature generation failed: {e}")
        raise HTTPException(status_code=400, detail=str(e))

# ============================================================================
# Autocorrelation Features Endpoints
# ============================================================================

@app.post("/autocorrelation-features")
async def generate_autocorrelation_features(request: TimeSeriesData):
    """
    Generate lag-based and autocorrelation features
    
    Generates: lags (1, 7, 14, 30), rolling stats (7, 14, 30 day windows),
               ACF, PACF at multiple lags
    """
    try:
        ts = np.array(request.values, dtype=np.float64)
        
        if len(ts) < 50:
            raise ValueError("Time-series must have at least 50 points")
        
        gen = AutocorrelationFeaturesGenerator(ts)
        
        # Lag features
        lag_features = gen.create_lag_features([1, 7, 14, 30])
        lag_dict = lag_features.to_dict('list')
        
        # Rolling features
        rolling_features = gen.create_rolling_features([7, 14, 30])
        rolling_dict = rolling_features.to_dict('list')
        
        # ACF/PACF features
        acf_features = gen.get_autocorrelation_features()
        
        # Combine all
        all_features = {}
        all_features.update(lag_dict)
        all_features.update(rolling_dict)
        all_features.update(acf_features)
        
        return JSONResponse({
            "feature_id": request.feature_id or "unknown",
            "feature_categories": {
                "lag_features": list(lag_dict.keys()),
                "rolling_features": list(rolling_dict.keys()),
                "acf_pacf_features": list(acf_features.keys())
            },
            "total_features": len(all_features),
            "features": {k: (v if not isinstance(v, list) else [float(x) for x in v])
                        for k, v in all_features.items()},
            "timestamp": datetime.utcnow().isoformat()
        })
    
    except Exception as e:
        logger.error(f"Autocorrelation feature generation failed: {e}")
        raise HTTPException(status_code=400, detail=str(e))

# ============================================================================
# Anomaly Detection Endpoints
# ============================================================================

@app.post("/detect-anomalies")
async def detect_anomalies(request: AnomalyRequest):
    """
    Detect anomalies using ensemble of methods
    
    Methods:
    - Statistical: Z-score, IQR, Modified Z-score
    - Machine Learning: Isolation Forest, DBSCAN
    - Voting: 2+ methods must agree for classification
    """
    try:
        ts = np.array(request.values, dtype=np.float64)
        timestamps = parse_timestamps(request.timestamps)
        
        if len(ts) < 20:
            raise ValueError("Time-series must have at least 20 points")
        
        detector = EnsembleAnomalyDetector(ts, timestamps)
        result = detector.detect()
        
        return JSONResponse({
            "feature_id": request.feature_id or "unknown",
            "n_total_points": len(ts),
            "n_anomalies": result.n_anomalies,
            "anomaly_percentage": float(result.anomaly_percentage),
            "anomaly_indices": result.anomaly_indices,
            "anomaly_scores": [float(s) for s in result.anomaly_scores],
            "is_anomaly": [bool(a) for a in result.is_anomaly],
            "detection_methods": result.anomaly_types,
            "voting_threshold": result.thresholds.get('voting_threshold', 2),
            "timestamp": datetime.utcnow().isoformat()
        })
    
    except Exception as e:
        logger.error(f"Anomaly detection failed: {e}")
        raise HTTPException(status_code=400, detail=str(e))

# ============================================================================
# Health & Status Endpoints
# ============================================================================

@app.get("/health")
async def health_check():
    """Service health check"""
    return {
        "status": "healthy",
        "service": "time-series-features",
        "version": "3.22.0",
        "timestamp": datetime.utcnow().isoformat()
    }

@app.get("/capabilities")
async def get_capabilities():
    """Get list of available capabilities"""
    return {
        "services": [
            {
                "name": "Decomposition",
                "endpoint": "/decompose",
                "methods": ["additive", "multiplicative", "robust"]
            },
            {
                "name": "Forecasting",
                "endpoint": "/forecast",
                "models": ["arima", "prophet", "ensemble"]
            },
            {
                "name": "Fourier Features",
                "endpoint": "/fourier-features",
                "description": "Periodic pattern capture"
            },
            {
                "name": "Autocorrelation Features",
                "endpoint": "/autocorrelation-features",
                "features": ["lags", "rolling_stats", "acf", "pacf"]
            },
            {
                "name": "Anomaly Detection",
                "endpoint": "/detect-anomalies",
                "methods": ["statistical", "isolation_forest", "dbscan", "ensemble"]
            }
        ],
        "timestamp": datetime.utcnow().isoformat()
    }

# ============================================================================
# Main
# ============================================================================

if __name__ == "__main__":
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=8001,
        log_level="info"
    )
