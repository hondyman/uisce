#!/usr/bin/env python3
"""
Conflict Resolution ML Model Training
"""

import pandas as pd
import numpy as np
from sklearn.ensemble import RandomForestClassifier
from sklearn.model_selection import train_test_split
from sklearn.metrics import accuracy_score, classification_report, confusion_matrix
import joblib
import boto3
import json
from datetime import datetime
import argparse

class ConflictResolutionTrainer:
    def __init__(self, s3_bucket, db_connection):
        self.s3_bucket = s3_bucket
        self.db_connection = db_connection
        self.s3_client = boto3.client('s3')
        
    def load_training_data(self):
        """Load conflict resolution data from database"""
        query = """
        SELECT 
            sc.conflict_type,
            sc.severity,
            sc.resolution_strategy as actual_outcome,
            sc.detected_at,
            sc.resolved_at,
            sc.user_id,
            EXTRACT(EPOCH FROM (sc.resolved_at - sc.detected_at)) as resolution_time,
            ge.title as google_title,
            ie.title as internal_title,
            ge.start_time as google_start,
            ie.start_time as internal_start
        FROM sync_conflicts sc
        LEFT JOIN google_events ge ON sc.google_event_id = ge.id
        LEFT JOIN internal_events ie ON sc.internal_event_id = ie.id
        WHERE sc.resolution_strategy IS NOT NULL
        AND sc.resolved_at > NOW() - INTERVAL '90 days'
        """
        
        df = pd.read_sql_query(query, self.db_connection)
        return df
    
    def engineer_features(self, df):
        """Create ML features from raw data"""
        features = pd.DataFrame()
        
        # Conflict characteristics
        features['conflict_type'] = df['conflict_type']
        features['severity'] = df['severity']
        
        # Title similarity (simplified)
        features['title_similarity'] = df.apply(
            lambda row: self.calculate_title_similarity(
                row['google_title'], row['internal_title']
            ), axis=1
        )
        
        # Time overlap
        features['time_overlap'] = df.apply(
            lambda row: self.calculate_time_overlap(
                row['google_start'], row['internal_start']
            ), axis=1
        )
        
        # Time-based features
        features['hour_of_day'] = pd.to_datetime(df['detected_at']).dt.hour
        features['day_of_week'] = pd.to_datetime(df['detected_at']).dt.dayofweek
        features['is_business_hours'] = (
            (features['hour_of_day'] >= 9) & 
            (features['hour_of_day'] <= 17) & 
            (features['day_of_week'] < 5)
        ).astype(int)
        
        # Encode categorical variables
        features = pd.get_dummies(features, columns=['conflict_type', 'severity'])
        
        return features
    
    def calculate_title_similarity(self, title1, title2):
        if not title1 or not title2:
            return 0.0
        
        title1_lower = title1.lower()
        title2_lower = title2.lower()
        
        if title1_lower == title2_lower:
            return 1.0
        
        words1 = set(title1_lower.split())
        words2 = set(title2_lower.split())
        
        if len(words1) == 0 or len(words2) == 0:
            return 0.0
        
        overlap = len(words1 & words2)
        total = len(words1 | words2)
        
        return overlap / total if total > 0 else 0.0
    
    def calculate_time_overlap(self, start1, start2):
        if not start1 or not start2:
            return 0.0
        
        try:
            t1 = pd.to_datetime(start1)
            t2 = pd.to_datetime(start2)
            
            diff_hours = abs((t1 - t2).total_seconds()) / 3600
            
            if diff_hours == 0:
                return 1.0
            elif diff_hours < 1:
                return 0.9
            elif diff_hours < 2:
                return 0.7
            elif diff_hours < 4:
                return 0.5
            else:
                return 0.2
        except:
            return 0.0
    
    def train_model(self):
        """Train conflict resolution model"""
        print("Loading training data...")
        df = self.load_training_data()
        
        if len(df) == 0:
            print("No training data available")
            return None
        
        print(f"Loaded {len(df)} conflict records")
        
        # Engineer features
        print("Engineering features...")
        X = self.engineer_features(df)
        y = df['actual_outcome']
        
        # Split data
        X_train, X_test, y_train, y_test = train_test_split(
            X, y, test_size=0.2, random_state=42, stratify=y
        )
        
        # Train model
        print("Training model...")
        model = RandomForestClassifier(
            n_estimators=100,
            max_depth=10,
            min_samples_split=5,
            min_samples_leaf=2,
            random_state=42,
            class_weight='balanced'
        )
        model.fit(X_train, y_train)
        
        # Evaluate
        print("\nEvaluating model...")
        y_pred = model.predict(X_test)
        y_pred_proba = model.predict_proba(X_test)
        
        accuracy = accuracy_score(y_test, y_pred)
        print(f"Model accuracy: {accuracy:.2f}")
        print("\nClassification report:")
        print(classification_report(y_test, y_pred))
        
        print("\nConfusion matrix:")
        print(confusion_matrix(y_test, y_pred))
        
        # Feature importance
        print("\nTop 10 most important features:")
        feature_importance = pd.DataFrame({
            'feature': X.columns,
            'importance': model.feature_importances_
        }).sort_values('importance', ascending=False)
        print(feature_importance.head(10))
        
        # Save model
        version = datetime.now().strftime('%Y%m%d_%H%M%S')
        model_file = f'models/conflict_resolution_{version}.pkl'
        joblib.dump(model, model_file)
        
        # Upload to S3
        s3_key = f'ml-models/conflict_resolution/{version}/model.pkl'
        self.s3_client.upload_file(model_file, self.s3_bucket, s3_key)
        
        # Save metadata
        metadata = {
            'model_name': 'conflict_resolution',
            'version': version,
            'accuracy': accuracy,
            'trained_at': datetime.now().isoformat(),
            'training_samples': len(df),
            'feature_names': list(X.columns),
            'classes': list(model.classes_),
            's3_path': f's3://{self.s3_bucket}/{s3_key}'
        }
        
        with open(f'models/conflict_resolution_{version}_metadata.json', 'w') as f:
            json.dump(metadata, f, indent=2)
        
        # Upload metadata
        self.s3_client.upload_file(
            f'models/conflict_resolution_{version}_metadata.json',
            self.s3_bucket,
            f'ml-models/conflict_resolution/{version}/metadata.json'
        )
        
        print(f"\nModel saved: {s3_key}")
        print(f"Model version: {version}")
        print(f"Accuracy: {accuracy:.2f}")
        
        return model, version, accuracy
    
    def run_training_pipeline(self):
        print("=" * 60)
        print("Conflict Resolution Model Training Pipeline")
        print("=" * 60)
        print(f"Timestamp: {datetime.now()}")
        print()
        
        model, version, accuracy = self.train_model()
        
        if model and accuracy > 0.7:
            print("\n✅ Training successful!")
            print(f"Model version {version} ready for deployment")
            return version
        else:
            print("\n❌ Training failed or accuracy too low")
            return None

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--s3-bucket', required=True, help='S3 bucket for model storage')
    parser.add_argument('--db-url', required=True, help='Database connection URL')
    args = parser.parse_args()
    
    trainer = ConflictResolutionTrainer(
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
