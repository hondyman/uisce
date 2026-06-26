#!/usr/bin/env python3
"""
Sync Optimization ML Model Training
"""

import pandas as pd
import numpy as np
from sklearn.ensemble import GradientBoostingRegressor
from sklearn.model_selection import train_test_split
from sklearn.metrics import mean_squared_error, mean_absolute_error
import joblib
import boto3
import json
from datetime import datetime, timedelta
import argparse

class SyncOptimizerTrainer:
    def __init__(self, s3_bucket, db_connection):
        self.s3_bucket = s3_bucket
        self.db_connection = db_connection
        self.s3_client = boto3.client('s3')
        
    def load_training_data(self):
        """Load sync optimization data from database"""
        query = """
        SELECT 
            sj.id,
            sj.user_id,
            sj.calendar_id,
            sj.started_at,
            sj.completed_at,
            sj.status,
            sj.total_events,
            sj.processed_events,
            EXTRACT(EPOCH FROM (sj.completed_at - sj.started_at)) as duration_seconds,
            EXTRACT(HOUR FROM sj.started_at) as hour_of_day,
            EXTRACT(DOW FROM sj.started_at) as day_of_week,
            EXTRACT(EPOCH FROM (NOW() - u.created_at)) / 86400 as user_tenure_days,
            EXTRACT(EPOCH FROM (NOW() - c.created_at)) / 86400 as calendar_age_days,
            c.provider,
            c.sync_frequency,
            u.timezone
        FROM sync_jobs sj
        JOIN users u ON sj.user_id = u.id
        JOIN calendars c ON sj.calendar_id = c.id
        WHERE sj.status = 'completed'
        AND sj.started_at > NOW() - INTERVAL '90 days'
        AND sj.duration_seconds > 0
        """
        
        df = pd.read_sql_query(query, self.db_connection)
        return df
    
    def engineer_features(self, df):
        """Create ML features for sync optimization"""
        features = pd.DataFrame()
        
        # Time-based features
        features['hour_of_day'] = df['hour_of_day']
        features['day_of_week'] = df['day_of_week']
        features['is_weekend'] = (df['day_of_week'] >= 5).astype(int)
        features['is_business_hours'] = (
            (df['hour_of_day'] >= 9) & 
            (df['hour_of_day'] <= 17) & 
            (df['day_of_week'] < 5)
        ).astype(int)
        
        # User features
        features['user_tenure_days'] = df['user_tenure_days']
        features['calendar_age_days'] = df['calendar_age_days']
        features['total_events'] = df['total_events']
        features['event_density'] = df['total_events'] / (df['calendar_age_days'] + 1)
        
        # Encode categorical variables
        features = pd.get_dummies(features, columns=['provider', 'sync_frequency'])
        
        return features
    
    def train_model(self):
        """Train sync duration prediction model"""
        print("Loading training data...")
        df = self.load_training_data()
        
        if len(df) == 0:
            print("No training data available")
            return None
        
        print(f"Loaded {len(df)} sync records")
        
        # Engineer features
        print("Engineering features...")
        X = self.engineer_features(df)
        y = df['duration_seconds']
        
        # Split data
        X_train, X_test, y_train, y_test = train_test_split(
            X, y, test_size=0.2, random_state=42
        )
        
        # Train model
        print("Training model...")
        model = GradientBoostingRegressor(
            n_estimators=100,
            max_depth=5,
            learning_rate=0.1,
            min_samples_split=10,
            min_samples_leaf=5,
            random_state=42
        )
        model.fit(X_train, y_train)
        
        # Evaluate
        print("\nEvaluating model...")
        y_pred = model.predict(X_test)
        
        mse = mean_squared_error(y_test, y_pred)
        mae = mean_absolute_error(y_test, y_pred)
        rmse = np.sqrt(mse)
        
        print(f"MSE: {mse:.2f}")
        print(f"MAE: {mae:.2f} seconds")
        print(f"RMSE: {rmse:.2f} seconds")
        
        # Calculate improvement potential
        avg_duration = y_test.mean()
        improvement = (avg_duration - mae) / avg_duration * 100
        print(f"\nPotential improvement: {improvement:.1f}%")
        
        # Feature importance
        print("\nTop 10 most important features:")
        feature_importance = pd.DataFrame({
            'feature': X.columns,
            'importance': model.feature_importances_
        }).sort_values('importance', ascending=False)
        print(feature_importance.head(10))
        
        # Save model
        version = datetime.now().strftime('%Y%m%d_%H%M%S')
        model_file = f'models/sync_optimizer_{version}.pkl'
        joblib.dump(model, model_file)
        
        # Upload to S3
        s3_key = f'ml-models/sync_optimizer/{version}/model.pkl'
        self.s3_client.upload_file(model_file, self.s3_bucket, s3_key)
        
        # Save metadata
        metadata = {
            'model_name': 'sync_optimizer',
            'version': version,
            'mse': mse,
            'mae': mae,
            'rmse': rmse,
            'improvement_potential': improvement,
            'trained_at': datetime.now().isoformat(),
            'training_samples': len(df),
            'feature_names': list(X.columns),
            's3_path': f's3://{self.s3_bucket}/{s3_key}'
        }
        
        with open(f'models/sync_optimizer_{version}_metadata.json', 'w') as f:
            json.dump(metadata, f, indent=2)
        
        # Upload metadata
        self.s3_client.upload_file(
            f'models/sync_optimizer_{version}_metadata.json',
            self.s3_bucket,
            f'ml-models/sync_optimizer/{version}/metadata.json'
        )
        
        print(f"\nModel saved: {s3_key}")
        print(f"Model version: {version}")
        print(f"MAE: {mae:.2f} seconds")
        print(f"Potential improvement: {improvement:.1f}%")
        
        return model, version, mae, improvement
    
    def run_training_pipeline(self):
        """Run complete training pipeline"""
        print("=" * 60)
        print("Sync Optimization Model Training Pipeline")
        print("=" * 60)
        print(f"Timestamp: {datetime.now()}")
        print()
        
        result = self.train_model()
        
        if result:
            model, version, mae, improvement = result
            if improvement > 10:  # At least 10% improvement
                print("\n✅ Training successful!")
                print(f"Model version {version} ready for deployment")
                print(f"Expected cost savings: {improvement:.1f}%")
                return version
            else:
                print("\n⚠️  Improvement too low, manual review recommended")
                return None
        else:
            print("\n❌ Training failed")
            return None

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--s3-bucket', required=True, help='S3 bucket for model storage')
    parser.add_argument('--db-url', required=True, help='Database connection URL')
    args = parser.parse_args()
    
    trainer = SyncOptimizerTrainer(
        s3_bucket=args.s3_bucket,
        db_connection=args.db_url
    )
    
    version = trainer.run_training_pipeline()
    
    if version:
        print(f"\n🎉 Model trained successfully: {version}")
        exit(0)
    else:
        print("\n💥 Training failed")
        exit(1)
