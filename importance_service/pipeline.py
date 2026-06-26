"""
Phase 3.21: Feature Importance Pipeline
SHAP-based feature importance computation with trend analysis and stability metrics.
"""

import logging
import asyncio
from datetime import datetime
import numpy as np
import pandas as pd
import xgboost as xgb
import shap
from typing import Dict, List, Tuple, Optional

from config import settings
from storage.postgres import store_feature_importance, get_feature_data, update_stability_metrics
from storage.iceberg import load_training_dataset
from metrics.prometheus import importance_computation_duration, importance_calculation_errors

logger = logging.getLogger(__name__)

class FeatureImportanceComputer:
    """Computes SHAP-based feature importance for models"""
    
    def __init__(self, model_id: str, model_path: str):
        self.model_id = model_id
        self.model_path = model_path
        self.model = None
        self.explainer = None
    
    def load_model(self):
        """Load XGBoost model from path"""
        try:
            self.model = xgb.Booster()
            self.model.load_model(self.model_path)
            logger.info(f"Loaded model {self.model_id} from {self.model_path}")
        except Exception as e:
            logger.error(f"Failed to load model {self.model_id}: {str(e)}")
            raise
    
    def compute_shap_values(self, X: pd.DataFrame, sample_size: Optional[int] = None) -> Tuple[np.ndarray, np.ndarray]:
        """
        Compute SHAP values using TreeExplainer.
        
        Returns:
            (shap_values, base_value)
        """
        try:
            # Sample if dataset too large
            if sample_size and len(X) > sample_size:
                X_sample = X.sample(n=sample_size, random_state=42)
                logger.info(f"Sampled {sample_size} rows from {len(X)} for SHAP computation")
            else:
                X_sample = X
            
            # Create DMatrix
            dmatrix = xgb.DMatrix(X_sample)
            
            # Compute SHAP values
            logger.info(f"Computing SHAP values for {len(X_sample)} samples...")
            self.explainer = shap.TreeExplainer(self.model)
            shap_values = self.explainer.shap_values(X_sample)
            
            return shap_values, self.explainer.expected_value
        except Exception as e:
            logger.error(f"Failed to compute SHAP values: {str(e)}")
            raise
    
    async def compute_importance(self, X: pd.DataFrame, feature_names: List[str]) -> Dict:
        """
        Compute aggregate feature importance metrics.
        """
        try:
            # SHAP values
            shap_values, base_value = await asyncio.to_thread(self.compute_shap_values, X)
            
            # Mean absolute SHAP
            mean_abs_shap = np.abs(shap_values).mean(axis=0)
            
            # Permutation importance (drop-column method)
            logger.info("Computing permutation importance...")
            perm_importance = await asyncio.to_thread(
                self._compute_permutation_importance, X
            )
            
            # Gain importance (tree-based)
            gain_importance = self._compute_gain_importance()
            
            # Build result dict
            importance_dict = {
                "model_id": self.model_id,
                "computed_at": datetime.utcnow(),
                "dataset_size": len(X),
                "features": []
            }
            
            for i, fname in enumerate(feature_names):
                importance_dict["features"].append({
                    "name": fname,
                    "mean_abs_shap": float(mean_abs_shap[i]),
                    "permutation_importance": float(perm_importance[i]) if perm_importance is not None else None,
                    "gain_importance": float(gain_importance[i]) if gain_importance is not None else None,
                    "shap_values_sample": shap_values[:min(100, len(shap_values)), i].tolist()
                })
            
            return importance_dict
        except Exception as e:
            logger.error(f"Feature importance computation failed: {str(e)}")
            importance_calculation_errors.labels(model_id=self.model_id).inc()
            raise
    
    def _compute_permutation_importance(self, X: pd.DataFrame) -> np.ndarray:
        """Compute drop-column permutation importance"""
        try:
            baseline_score = self.model.predict(xgb.DMatrix(X)).mean()
            
            perm_importance = []
            for col in X.columns:
                X_permuted = X.copy()
                X_permuted[col] = np.random.permutation(X_permuted[col].values)
                
                permuted_score = self.model.predict(xgb.DMatrix(X_permuted)).mean()
                importance = baseline_score - permuted_score
                perm_importance.append(importance)
            
            return np.array(perm_importance)
        except Exception as e:
            logger.warning(f"Permutation importance computation failed: {str(e)}")
            return None
    
    def _compute_gain_importance(self) -> Optional[np.ndarray]:
        """Extract tree-based gain (feature frequency) importance"""
        try:
            importance_dict = self.model.get_score(importance_type='gain')
            # importance_dict is {feature_name: score, ...}
            logger.info(f"Extracted gain importance for {len(importance_dict)} features")
            return np.array(list(importance_dict.values()))
        except Exception as e:
            logger.warning(f"Gain importance extraction failed: {str(e)}")
            return None

async def compute_nightly_importance(model_id: str, tenant_id: str = "default") -> None:
    """
    Nightly job to compute and persist feature importance for a model.
    """
    logger.info(f"Starting nightly importance computation for {model_id}")
    
    try:
        # Load model path from registry
        model_path = f"/models/{model_id}/model.bin"  # TODO: Fetch from model registry
        
        # Initialize computer
        computer = FeatureImportanceComputer(model_id, model_path)
        computer.load_model()
        
        # Load training dataset (last 30 days)
        X_train, y_train, feature_names = await load_training_dataset(
            days_back=30,
            tenant_id=tenant_id,
            sample_size=5000
        )
        
        if X_train is None or len(X_train) == 0:
            logger.warning(f"No training data available for {model_id}")
            return
        
        # Compute importance
        with importance_computation_duration.labels(model_id=model_id).time():
            importance_result = await computer.compute_importance(X_train, feature_names)
        
        # Compute stability (comparing to previous day)
        importance_result["stability_score"] = await compute_stability(model_id, importance_result)
        importance_result["importance_trend"] = await compute_trend(model_id, importance_result)
        importance_result["importance_percentile"] = await compute_percentiles(importance_result)
        
        # Persist to database
        await store_feature_importance(importance_result, tenant_id)
        
        logger.info(f"Nightly importance computation completed for {model_id}")
    except Exception as e:
        logger.error(f"Nightly importance computation failed for {model_id}: {str(e)}")
        raise

async def compute_stability(model_id: str, current_result: Dict) -> float:
    """
    Compute stability score: 1 - variance(importance over last N runs).
    Measures how consistent feature importance is over time.
    
    Returns [0, 1]:
    - 0.9+: Very stable
    - 0.7-0.9: Moderately stable
    - <0.7: Unstable (feature importance changing)
    """
    try:
        # Get historical importance scores for past N days
        historical_scores = await get_feature_data(
            model_id=model_id,
            table="feature_importance",
            days_back=30
        )
        
        if not historical_scores or len(historical_scores) < 2:
            return 0.5  # Default if insufficient history
        
        # Compute variance of mean_abs_shap over time
        variances = []
        for feature in current_result["features"]:
            fname = feature["name"]
            feature_scores = [score.get(fname, 0) for score in historical_scores]
            variance = np.var(feature_scores) if len(feature_scores) > 1 else 0
            variances.append(variance)
        
        avg_variance = np.mean(variances)
        # Normalize: higher variance → lower stability
        stability = 1.0 - min(avg_variance / 0.1, 1.0)  # Normalize by typical variance scale
        
        return float(max(0, min(1, stability)))
    except Exception as e:
        logger.warning(f"Stability computation failed: {str(e)}")
        return 0.5

async def compute_trend(model_id: str, current_result: Dict) -> float:
    """
    Compute importance trend: slope of importance over last N days.
    
    Returns:
    - > 0: Importance increasing (feature gaining importance)
    - < 0: Importance decreasing (feature losing importance)
    - ~0: Stable importance
    """
    try:
        historical_scores = await get_feature_data(
            model_id=model_id,
            table="feature_importance",
            days_back=30
        )
        
        if not historical_scores or len(historical_scores) < 2:
            return 0.0
        
        # Compute average importance per day
        daily_avg = []
        for score_dict in historical_scores:
            avg_importance = np.mean([f.get("mean_abs_shap", 0) for f in score_dict.get("features", [])])
            daily_avg.append(avg_importance)
        
        # Linear regression to get slope
        if len(daily_avg) >= 5:
            x = np.arange(len(daily_avg))
            coeffn = np.polyfit(x, daily_avg, 1)
            trend = float(coeffn[0])  # Slope
        else:
            trend = 0.0
        
        return trend
    except Exception as e:
        logger.warning(f"Trend computation failed: {str(e)}")
        return 0.0

async def compute_percentiles(importance_result: Dict) -> List[float]:
    """
    Compute importance percentile rank for each feature [0, 100].
    
    Features in top 10% have percentile ~90-100.
    """
    mean_shap_scores = [f["mean_abs_shap"] for f in importance_result["features"]]
    
    percentiles = []
    for i, score in enumerate(mean_shap_scores):
        percentile = (sum(1 for s in mean_shap_scores if s <= score) / len(mean_shap_scores)) * 100
        percentiles.append(float(percentile))
    
    return percentiles

async def bulk_importance_update(model_ids: List[str], tenant_id: str = "default") -> None:
    """
    Bulk update importance for multiple models (called from Temporal workflow).
    """
    logger.info(f"Starting bulk importance update for {len(model_ids)} models")
    
    tasks = [
        compute_nightly_importance(mid, tenant_id)
        for mid in model_ids
    ]
    
    results = await asyncio.gather(*tasks, return_exceptions=True)
    
    failed = sum(1 for r in results if isinstance(r, Exception))
    logger.info(f"Bulk update completed: {len(model_ids) - failed} succeeded, {failed} failed")
    
    return {"total": len(model_ids), "succeeded": len(model_ids) - failed, "failed": failed}
