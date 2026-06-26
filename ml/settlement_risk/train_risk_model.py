#!/usr/bin/env python3
"""
Settlement Risk Model Training Script

This script trains an XGBoost Gradient Boosting Classifier to predict
settlement failures based on historical trade data features.

Usage:
    python train_risk_model.py --db-url postgresql://user:pass@host:5432/dbname
"""

import argparse
import os
import sys
import json
from datetime import datetime

import pandas as pd
import numpy as np
import xgboost as xgb
from sklearn.model_selection import train_test_split, cross_val_score
from sklearn.metrics import (
    accuracy_score, 
    classification_report, 
    roc_auc_score,
    confusion_matrix,
    precision_recall_curve,
    f1_score
)
from sklearn.preprocessing import LabelEncoder
import joblib
import psycopg2


def load_data(db_url: str) -> pd.DataFrame:
    """Load settlement features from PostgreSQL."""
    print("📊 Loading data from database...")
    
    conn = psycopg2.connect(db_url)
    
    # Use materialized view if available, otherwise fall back to view
    try:
        df = pd.read_sql_query("SELECT * FROM settlement_features_materialized", conn)
        print(f"   Loaded {len(df)} records from materialized view")
    except Exception:
        df = pd.read_sql_query("SELECT * FROM settlement_features_view", conn)
        print(f"   Loaded {len(df)} records from view")
    
    conn.close()
    return df


def preprocess_data(df: pd.DataFrame) -> tuple[pd.DataFrame, pd.Series, dict]:
    """Preprocess features for model training."""
    print("🔧 Preprocessing data...")
    
    # Drop ID columns (not features)
    id_columns = ['order_id', 'customer_id', 'order_date']
    df_features = df.drop(columns=id_columns, errors='ignore')
    
    # Handle missing values
    numeric_cols = df_features.select_dtypes(include=[np.number]).columns
    df_features[numeric_cols] = df_features[numeric_cols].fillna(0)
    
    # Encode categorical variables
    label_encoders = {}
    categorical_cols = df_features.select_dtypes(include=['object']).columns
    
    for col in categorical_cols:
        if col != 'settlement_failed':
            le = LabelEncoder()
            df_features[col] = df_features[col].fillna('UNKNOWN')
            df_features[col] = le.fit_transform(df_features[col].astype(str))
            label_encoders[col] = le
            print(f"   Encoded {col}: {len(le.classes_)} categories")
    
    # Separate target and features
    y = df_features['settlement_failed'].astype(int)
    X = df_features.drop(columns=['settlement_failed'])
    
    # Store feature names for later
    feature_names = list(X.columns)
    
    print(f"   Features: {len(feature_names)}")
    print(f"   Samples: {len(X)}")
    print(f"   Positive class ratio: {y.mean():.2%}")
    
    return X, y, {'label_encoders': label_encoders, 'feature_names': feature_names}


def train_model(X: pd.DataFrame, y: pd.Series, test_size: float = 0.25) -> tuple:
    """Train XGBoost classifier with cross-validation."""
    print("🎯 Training XGBoost model...")
    
    # Split data
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=test_size, random_state=42, stratify=y
    )
    
    print(f"   Training set: {len(X_train)} samples")
    print(f"   Test set: {len(X_test)} samples")
    
    # Handle class imbalance with scale_pos_weight
    neg_count = (y_train == 0).sum()
    pos_count = (y_train == 1).sum()
    scale_pos_weight = neg_count / pos_count if pos_count > 0 else 1
    
    # Train XGBoost model with tuned hyperparameters
    model = xgb.XGBClassifier(
        objective='binary:logistic',
        eval_metric='logloss',
        n_estimators=100,
        learning_rate=0.1,
        max_depth=4,
        min_child_weight=2,
        subsample=0.8,
        colsample_bytree=0.8,
        scale_pos_weight=scale_pos_weight,
        random_state=42,
        n_jobs=-1
    )
    
    # Fit with early stopping
    model.fit(
        X_train, y_train,
        eval_set=[(X_test, y_test)],
        verbose=False
    )
    
    # Cross-validation scores
    cv_scores = cross_val_score(model, X, y, cv=5, scoring='roc_auc')
    print(f"   CV ROC-AUC: {cv_scores.mean():.4f} (+/- {cv_scores.std() * 2:.4f})")
    
    return model, X_train, X_test, y_train, y_test


def evaluate_model(model, X_test, y_test) -> dict:
    """Evaluate model performance."""
    print("\n📈 Model Evaluation:")
    
    # Predictions
    y_pred = model.predict(X_test)
    y_pred_proba = model.predict_proba(X_test)[:, 1]
    
    # Metrics
    accuracy = accuracy_score(y_test, y_pred)
    roc_auc = roc_auc_score(y_test, y_pred_proba) if len(np.unique(y_test)) > 1 else 0
    f1 = f1_score(y_test, y_pred, zero_division=0)
    
    print(f"   Accuracy: {accuracy:.4f}")
    print(f"   ROC-AUC: {roc_auc:.4f}")
    print(f"   F1 Score: {f1:.4f}")
    
    print("\n   Classification Report:")
    print(classification_report(y_test, y_pred, zero_division=0))
    
    print("   Confusion Matrix:")
    cm = confusion_matrix(y_test, y_pred)
    print(f"   {cm}")
    
    # Feature importance
    print("\n   Top 10 Feature Importances:")
    importance = model.feature_importances_
    feature_names = model.get_booster().feature_names or [f"f{i}" for i in range(len(importance))]
    importance_df = pd.DataFrame({
        'feature': feature_names,
        'importance': importance
    }).sort_values('importance', ascending=False).head(10)
    
    for _, row in importance_df.iterrows():
        print(f"   - {row['feature']}: {row['importance']:.4f}")
    
    return {
        'accuracy': float(accuracy),
        'roc_auc': float(roc_auc),
        'f1_score': float(f1),
        'feature_importance': importance_df.to_dict('records')
    }


def save_model(model, preprocessing_info: dict, metrics: dict, output_dir: str):
    """Save trained model and metadata."""
    print(f"\n💾 Saving model to {output_dir}...")
    
    os.makedirs(output_dir, exist_ok=True)
    
    # Save model
    model_path = os.path.join(output_dir, 'settlement_risk_model.pkl')
    joblib.dump(model, model_path)
    print(f"   Model: {model_path}")
    
    # Save preprocessing info
    preproc_path = os.path.join(output_dir, 'preprocessing.pkl')
    joblib.dump(preprocessing_info, preproc_path)
    print(f"   Preprocessing: {preproc_path}")
    
    # Save metadata
    metadata = {
        'model_type': 'XGBClassifier',
        'trained_at': datetime.now().isoformat(),
        'metrics': metrics,
        'feature_names': preprocessing_info['feature_names'],
        'version': '1.0.0'
    }
    metadata_path = os.path.join(output_dir, 'model_metadata.json')
    with open(metadata_path, 'w') as f:
        json.dump(metadata, f, indent=2)
    print(f"   Metadata: {metadata_path}")
    
    print("\n✅ Model training complete!")


def generate_synthetic_data(n_samples: int = 1000) -> pd.DataFrame:
    """Generate synthetic training data for development/testing."""
    print("🧪 Generating synthetic training data...")
    
    np.random.seed(42)
    
    data = {
        'order_id': range(1, n_samples + 1),
        'customer_id': [f'CUST{i % 100:03d}' for i in range(n_samples)],
        'order_date': pd.date_range(start='2023-01-01', periods=n_samples, freq='H'),
        'line_item_count': np.random.poisson(3, n_samples) + 1,
        'is_cross_border': np.random.binomial(1, 0.3, n_samples),
        'order_to_ship_days': np.abs(np.random.normal(5, 3, n_samples)),
        'customer_country': np.random.choice(['USA', 'UK', 'Germany', 'France', 'Japan'], n_samples),
        'customer_trade_history_count': np.random.poisson(10, n_samples),
        'customer_previous_fails': np.random.poisson(0.5, n_samples),
        'is_missing_postal_code': np.random.binomial(1, 0.05, n_samples),
        'is_missing_ship_date': np.random.binomial(1, 0.02, n_samples),
        'is_missing_address': np.random.binomial(1, 0.01, n_samples),
        'order_freight_cost': np.abs(np.random.normal(50, 30, n_samples)),
        'order_total_value': np.abs(np.random.lognormal(6, 1, n_samples)),
        'shipper_id': np.random.choice([1, 2, 3], n_samples),
        'order_day_of_week': np.random.randint(0, 7, n_samples),
        'order_month': np.random.randint(1, 13, n_samples),
        'days_until_required': np.abs(np.random.normal(14, 7, n_samples)),
    }
    
    df = pd.DataFrame(data)
    
    # Generate target with some realistic correlations
    fail_prob = (
        0.02 +  # Base rate
        0.1 * df['is_cross_border'] +
        0.05 * (df['customer_previous_fails'] > 0).astype(int) +
        0.03 * df['is_missing_postal_code'] +
        0.02 * (df['order_total_value'] > 5000).astype(int) +
        0.01 * (df['customer_trade_history_count'] < 5).astype(int)
    )
    fail_prob = np.clip(fail_prob, 0, 1)
    df['settlement_failed'] = np.random.binomial(1, fail_prob)
    
    print(f"   Generated {n_samples} samples with {df['settlement_failed'].sum()} failures ({df['settlement_failed'].mean():.1%})")
    
    return df


def main():
    parser = argparse.ArgumentParser(description='Train Settlement Risk Model')
    parser.add_argument('--db-url', type=str, help='PostgreSQL connection URL')
    parser.add_argument('--synthetic', action='store_true', help='Use synthetic data for testing')
    parser.add_argument('--output-dir', type=str, default='./model_output', help='Output directory')
    parser.add_argument('--samples', type=int, default=5000, help='Number of synthetic samples')
    args = parser.parse_args()
    
    print("=" * 60)
    print("🚀 Settlement Risk Model Training Pipeline")
    print("=" * 60)
    
    # Load data
    if args.synthetic:
        df = generate_synthetic_data(args.samples)
    elif args.db_url:
        df = load_data(args.db_url)
    else:
        print("❌ Error: Please provide --db-url or use --synthetic flag")
        sys.exit(1)
    
    # Preprocess
    X, y, preprocessing_info = preprocess_data(df)
    
    # Train
    model, X_train, X_test, y_train, y_test = train_model(X, y)
    
    # Evaluate
    metrics = evaluate_model(model, X_test, y_test)
    
    # Save
    save_model(model, preprocessing_info, metrics, args.output_dir)


if __name__ == '__main__':
    main()
