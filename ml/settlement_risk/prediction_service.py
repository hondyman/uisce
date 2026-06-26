#!/usr/bin/env python3
"""
Settlement Risk Prediction Service

A Flask microservice that exposes the trained XGBoost model
for settlement risk prediction via REST API.

Usage:
    python prediction_service.py
    
Endpoints:
    POST /predict - Get settlement risk score
    GET /health - Health check
    GET /metrics - Prometheus metrics
"""

import os
import logging
from datetime import datetime

from flask import Flask, request, jsonify
import joblib
import pandas as pd
import numpy as np
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize Flask app
app = Flask(__name__)

# Prometheus metrics
PREDICTION_COUNT = Counter('settlement_risk_predictions_total', 'Total predictions made')
PREDICTION_LATENCY = Histogram('settlement_risk_prediction_latency_seconds', 'Prediction latency')
HIGH_RISK_COUNT = Counter('settlement_risk_high_risk_total', 'Total high-risk predictions (score > 0.7)')

# Model paths
MODEL_DIR = os.environ.get('MODEL_DIR', './model_output')
MODEL_PATH = os.path.join(MODEL_DIR, 'settlement_risk_model.pkl')
PREPROC_PATH = os.path.join(MODEL_DIR, 'preprocessing.pkl')

# Global model storage
model = None
preprocessing_info = None
model_loaded_at = None


def load_model():
    """Load trained model and preprocessing info."""
    global model, preprocessing_info, model_loaded_at
    
    try:
        model = joblib.load(MODEL_PATH)
        preprocessing_info = joblib.load(PREPROC_PATH)
        model_loaded_at = datetime.now()
        logger.info(f"Model loaded successfully from {MODEL_PATH}")
        return True
    except FileNotFoundError as e:
        logger.warning(f"Model not found: {e}. Using fallback scoring.")
        return False
    except Exception as e:
        logger.error(f"Error loading model: {e}")
        return False


def preprocess_input(features: dict) -> pd.DataFrame:
    """Preprocess input features to match training format."""
    # Expected features
    expected_features = preprocessing_info.get('feature_names', []) if preprocessing_info else []
    
    df = pd.DataFrame([features])
    
    # Handle categorical encoding
    if preprocessing_info and 'label_encoders' in preprocessing_info:
        for col, le in preprocessing_info['label_encoders'].items():
            if col in df.columns:
                # Handle unseen categories
                val = df[col].iloc[0]
                if val in le.classes_:
                    df[col] = le.transform([val])
                else:
                    df[col] = 0  # Default for unseen categories
    
    # Ensure all expected features are present
    for feat in expected_features:
        if feat not in df.columns:
            df[feat] = 0
    
    # Select only expected features in correct order
    if expected_features:
        df = df[expected_features]
    
    return df


def fallback_risk_score(features: dict) -> float:
    """
    Calculate a rule-based risk score when model is unavailable.
    This provides reasonable defaults while model is being trained.
    """
    score = 0.05  # Base risk
    
    # Cross-border trades are riskier
    if features.get('is_cross_border', 0) == 1:
        score += 0.1
    
    # Missing data increases risk
    if features.get('is_missing_postal_code', 0) == 1:
        score += 0.05
    if features.get('is_missing_ship_date', 0) == 1:
        score += 0.1
    
    # Previous failures indicate higher risk
    prev_fails = features.get('customer_previous_fails', 0)
    if prev_fails > 0:
        score += min(0.2, prev_fails * 0.05)
    
    # High-value orders have higher risk
    value = features.get('order_total_value', 0)
    if value > 10000:
        score += 0.1
    elif value > 5000:
        score += 0.05
    
    # New customers are riskier
    history = features.get('customer_trade_history_count', 0)
    if history < 5:
        score += 0.1
    
    return min(score, 1.0)


@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint."""
    return jsonify({
        'status': 'healthy',
        'model_loaded': model is not None,
        'model_loaded_at': model_loaded_at.isoformat() if model_loaded_at else None,
        'timestamp': datetime.now().isoformat()
    })


@app.route('/metrics', methods=['GET'])
def metrics():
    """Prometheus metrics endpoint."""
    return generate_latest(), 200, {'Content-Type': CONTENT_TYPE_LATEST}


import json
import shap

# Feature log path
LOG_DIR = os.environ.get('LOG_DIR', './logs')
FEATURE_LOG_PATH = os.path.join(LOG_DIR, 'prediction_logs.jsonl')
os.makedirs(LOG_DIR, exist_ok=True)

# SHAP Explainer
explainer = None

def load_explainer():
    """Initialize SHAP explainer from model."""
    global explainer
    if model:
        try:
            # Use TreeExplainer for XGBoost
            explainer = shap.TreeExplainer(model)
            logger.info("SHAP explainer initialized")
        except Exception as e:
            logger.error(f"Failed to initialize SHAP explainer: {e}")

# Call after model load
if load_model():
    load_explainer()

def log_prediction(features: dict, result: dict):
    """Log prediction for drift detection."""
    try:
        log_entry = {
            'timestamp': datetime.now().isoformat(),
            'features': features,
            'result': result
        }
        with open(FEATURE_LOG_PATH, 'a') as f:
            f.write(json.dumps(log_entry) + '\n')
    except Exception as e:
        logger.error(f"Failed to log prediction: {e}")

@app.route('/explain', methods=['POST'])
def explain():
    """
    Get SHAP explanations for a prediction.
    """
    try:
        features = request.get_json()
        if not features or not model or not explainer:
            return jsonify({'error': 'Service not ready or invalid input'}), 400

        df_features = preprocess_input(features)
        
        # Calculate SHAP values
        shap_values = explainer.shap_values(df_features)
        
        # XGBoost binary classification output
        if isinstance(shap_values, list):
             # For binary classification, we care about the positive class
            feature_impacts = shap_values[1][0] if len(shap_values) > 1 else shap_values[0]
        else:
            feature_impacts = shap_values[0]

        # Map to feature names
        explanation = []
        feature_names = df_features.columns
        for i, name in enumerate(feature_names):
            explanation.append({
                'feature': name,
                'feature_value': float(df_features.iloc[0, i]),
                'impact': float(feature_impacts[i])
            })
        
        # Sort by absolute impact
        explanation.sort(key=lambda x: abs(x['impact']), reverse=True)
        
        return jsonify({
            'explanation': explanation[:10], # Top 10 drivers
            'base_value': float(explainer.expected_value[-1] if isinstance(explainer.expected_value, list) else explainer.expected_value)
        })

    except Exception as e:
        logger.error(f"Explanation error: {e}")
        return jsonify({'error': str(e)}), 500

@app.route('/predict', methods=['POST'])
def predict():
    """
    Predict settlement risk score with logging.
    """
    PREDICTION_COUNT.inc()
    
    with PREDICTION_LATENCY.time():
        try:
            features = request.get_json()
            if not features:
                return jsonify({'error': 'No features provided'}), 400
            
            using_fallback = False
            risk_score = 0.0
            
            if model is not None:
                # Use trained model
                try:
                    df_features = preprocess_input(features)
                    prediction_proba = model.predict_proba(df_features)[:, 1]
                    risk_score = float(prediction_proba[0])
                except Exception as e:
                    logger.warning(f"Model prediction failed, using fallback: {e}")
                    risk_score = fallback_risk_score(features)
                    using_fallback = True
            else:
                # Use rule-based fallback
                risk_score = fallback_risk_score(features)
                using_fallback = True
            
            # Clamp score to [0, 1]
            risk_score = max(0.0, min(1.0, risk_score))
            
            # Categorize risk
            if risk_score >= 0.75:
                risk_category = 'CRITICAL'
            elif risk_score >= 0.5:
                risk_category = 'HIGH'
            elif risk_score >= 0.25:
                risk_category = 'MEDIUM'
            else:
                risk_category = 'LOW'
            
            # Track high-risk predictions
            if risk_score > 0.7:
                HIGH_RISK_COUNT.inc()
            
            response = {
                'settlement_risk_score': round(risk_score, 4),
                'risk_category': risk_category,
                'model_version': '1.0.0',
                'using_fallback': using_fallback,
                'timestamp': datetime.now().isoformat()
            }
            
            # Log for monitoring
            log_prediction(features, response)
            
            logger.info(f"Prediction: score={risk_score:.4f}, category={risk_category}, fallback={using_fallback}")
            return jsonify(response)
            
        except Exception as e:
            logger.error(f"Prediction error: {e}")
            return jsonify({'error': str(e)}), 500


@app.route('/reload', methods=['POST'])
def reload_model():
    """Reload the model from disk (for hot updates)."""
    success = load_model()
    if success:
        load_explainer() # Reload explainer too
        return jsonify({'status': 'reloaded', 'timestamp': datetime.now().isoformat()})
    else:
        return jsonify({'status': 'failed', 'message': 'Model file not found'}), 500


if __name__ == '__main__':
    port = int(os.environ.get('PORT', 5000))
    debug = os.environ.get('DEBUG', 'false').lower() == 'true'
    
    logger.info(f"Starting Settlement Risk Prediction Service on port {port}")
    app.run(host='0.0.0.0', port=port, debug=debug)
