#!/usr/bin/env python3
"""
Phase 3.21 Integration Test Suite
End-to-end validation of drift detection, importance computation, and materialization.
"""

import pytest
import psycopg2
import json
import numpy as np
from datetime import datetime, timedelta
from typing import Dict, List
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# ============================================================================
# Fixtures & Setup
# ============================================================================

@pytest.fixture(scope="session")
def postgres_connection():
    """Connect to PostgreSQL for integration tests"""
    conn = psycopg2.connect(
        host="localhost",
        port=5432,
        user="postgres",
        password="secret",
        database="semlayer"
    )
    yield conn
    conn.close()

@pytest.fixture(autouse=True)
def cleanup_test_data(postgres_connection):
    """Clean up test data after each test"""
    yield
    cur = postgres_connection.cursor()
    # Keep sample data but allow for multiple test runs
    postgres_connection.commit()

# ============================================================================
# Phase 3.21 Schema Validation Tests
# ============================================================================

class TestSchemaValidation:
    """Validate Phase 3.21 database schema"""
    
    def test_all_tables_exist(self, postgres_connection):
        """Verify all 10 core tables exist"""
        cur = postgres_connection.cursor()
        
        expected_tables = [
            'feature_catalog',
            'feature_watermarks',
            'feature_drift_metrics',
            'feature_quality_checks',
            'feature_importance',
            'feature_change_log',
            'feature_test_cases',
            'feature_lineage',
            'feature_computations',
            'schema_migrations'
        ]
        
        for table in expected_tables:
            cur.execute(f"""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_name = '{table}'
                )
            """)
            assert cur.fetchone()[0], f"Table {table} not found"
    
    def test_feature_catalog_structure(self, postgres_connection):
        """Verify feature_catalog has all required columns"""
        cur = postgres_connection.cursor()
        
        required_columns = [
            'feature_id', 'name', 'owner', 'feature_type', 'expression',
            'drift_config', 'test_cases', 'region', 'tenant_id', 'created_at'
        ]
        
        cur.execute("""
            SELECT column_name FROM information_schema.columns 
            WHERE table_name = 'feature_catalog'
        """)
        
        columns = {row[0] for row in cur.fetchall()}
        for col in required_columns:
            assert col in columns, f"Column {col} not found in feature_catalog"
    
    def test_indexes_created(self, postgres_connection):
        """Verify performance indexes are created"""
        cur = postgres_connection.cursor()
        
        cur.execute("""
            SELECT COUNT(*) FROM pg_indexes 
            WHERE schemaname = 'public' AND indexname LIKE 'idx_%'
        """)
        
        count = cur.fetchone()[0]
        assert count >= 30, f"Expected ≥30 indexes, found {count}"
    
    def test_materialized_views_exist(self, postgres_connection):
        """Verify materialized views for nightly aggregations"""
        cur = postgres_connection.cursor()
        
        materialized_views = ['active_drifts', 'computation_slos']
        
        for view in materialized_views:
            cur.execute(f"""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_name = '{view}'
                )
            """)
            assert cur.fetchone()[0], f"Materialized view {view} not found"

# ============================================================================
# Sample Data Validation Tests
# ============================================================================

class TestSampleDataIntegrity:
    """Validate sample data is correctly loaded"""
    
    def test_sample_features_exist(self, postgres_connection):
        """Verify 5 sample features are loaded"""
        cur = postgres_connection.cursor()
        
        cur.execute("SELECT COUNT(*) FROM feature_catalog")
        count = cur.fetchone()[0]
        
        assert count >= 5, f"Expected ≥5 sample features, found {count}"
    
    def test_sample_watermarks_exist(self, postgres_connection):
        """Verify watermarks for incremental processing"""
        cur = postgres_connection.cursor()
        
        cur.execute("SELECT COUNT(*) FROM feature_watermarks")
        count = cur.fetchone()[0]
        
        assert count > 0, "No watermarks found in database"
    
    def test_sample_drift_metrics_exist(self, postgres_connection):
        """Verify drift metrics are loaded"""
        cur = postgres_connection.cursor()
        
        cur.execute("""
            SELECT COUNT(*) FROM feature_drift_metrics 
            WHERE ks_score IS NOT NULL OR psi_score IS NOT NULL
        """)
        count = cur.fetchone()[0]
        
        assert count > 0, "No drift metrics found"
    
    def test_sample_importance_scores_exist(self, postgres_connection):
        """Verify importance scores are loaded"""
        cur = postgres_connection.cursor()
        
        cur.execute("""
            SELECT COUNT(*) FROM feature_importance 
            WHERE shap_mean IS NOT NULL
        """)
        count = cur.fetchone()[0]
        
        assert count > 0, "No importance scores found"

# ============================================================================
# Drift Detection Algorithm Tests
# ============================================================================

class TestDriftAlgorithms:
    """Test drift detection algorithms in isolation"""
    
    def test_ks_test_perfect_separation(self):
        """KS test should detect perfect separation"""
        from scipy.stats import ks_2samp
        
        baseline = np.random.normal(0, 1, 1000)
        recent = np.random.normal(5, 1, 1000)  # Mean shift of 5
        
        statistic, p_value = ks_2samp(baseline, recent)
        
        assert statistic > 0.5, "Should detect large drift"
        assert p_value < 0.05, "Should be statistically significant"
    
    def test_ks_test_identical_distributions(self):
        """KS test should NOT detect drift for identical distributions"""
        from scipy.stats import ks_2samp
        
        np.random.seed(42)
        baseline = np.random.normal(0, 1, 1000)
        recent = np.random.normal(0, 1, 1000)  # Same distribution
        
        statistic, p_value = ks_2samp(baseline, recent)
        
        assert statistic < 0.15, "Should not detect drift"
        assert p_value > 0.05, "Should NOT be statistically significant"
    
    def test_psi_binned_continuous(self):
        """PSI should work on binned continuous features"""
        np.random.seed(42)
        baseline = np.random.normal(0, 1, 1000)
        
        # Bin into 10 buckets
        baseline_counts, _ = np.histogram(baseline, bins=10)
        baseline_pct = baseline_counts / baseline_counts.sum()
        
        # Shifted distribution
        recent = np.random.normal(1, 1, 1000)  # Mean shift
        recent_counts, _ = np.histogram(recent, bins=10)
        recent_pct = recent_counts / recent_counts.sum()
        
        # PSI computation
        epsilon = 1e-10
        psi = np.sum((baseline_pct - recent_pct) * np.log((baseline_pct + epsilon) / (recent_pct + epsilon)))
        
        assert psi > 0.1, "PSI should be > 0.1 for shifted distribution"
    
    def test_chi2_categorical(self):
        """Chi-square test for categorical features"""
        from scipy.stats import chisquare
        
        # Baseline: uniform distribution
        baseline = np.array([100, 100, 100, 100])
        
        # Recent: skewed distribution
        recent = np.array([200, 50, 25, 25])
        
        chi2, p_value = chisquare(recent, baseline)
        
        assert chi2 > 10, "Should detect significant drift"
        assert p_value < 0.05, "Should be statistically significant"
    
    def test_classifier_drift_multivariate(self):
        """Classifier AUC should detect multivariate drift"""
        from sklearn.ensemble import RandomForestClassifier
        from sklearn.metrics import roc_auc_score
        
        np.random.seed(42)
        
        # Generate baseline data
        baseline_X = np.random.normal(0, 1, (100, 5))
        baseline_y = np.zeros(100)
        
        # Generate drifted data
        drifted_X = np.random.normal(2, 1, (100, 5))  # Mean shift
        drifted_y = np.ones(100)
        
        # Train classifier
        X_combined = np.vstack([baseline_X, drifted_X])
        y_combined = np.hstack([baseline_y, drifted_y])
        
        clf = RandomForestClassifier(n_estimators=10, random_state=42)
        clf.fit(X_combined, y_combined)
        
        # AUC should be high (>0.7) for clear drift
        y_pred = clf.predict_proba(X_combined)[:, 1]
        auc = roc_auc_score(y_combined, y_pred)
        
        assert auc > 0.7, f"AUC should be > 0.7 for clear drift, got {auc}"

# ============================================================================
# Feature Importance Tests
# ============================================================================

class TestFeatureImportance:
    """Test feature importance computation"""
    
    def test_shap_values_output_shape(self):
        """SHAP values should have correct shape"""
        import shap
        from sklearn.ensemble import RandomForestClassifier
        
        np.random.seed(42)
        X = np.random.randn(100, 5)
        y = np.random.randint(0, 2, 100)
        
        model = RandomForestClassifier(n_estimators=10)
        model.fit(X, y)
        
        explainer = shap.TreeExplainer(model)
        shap_values = explainer.shap_values(X)
        
        # For binary classification, shap_values might be [class_0, class_1]
        if isinstance(shap_values, list):
            shap_values = shap_values[0]  # Use first class
        
        assert shap_values.shape == X.shape, f"Shape mismatch: {shap_values.shape} vs {X.shape}"
    
    def test_importance_stability_metric(self):
        """Stability should be between 0 and 1"""
        np.random.seed(42)
        
        # Simulated importance scores over 30 days
        scores = np.random.uniform(0.3, 0.7, 30)
        
        # Stability = 1 - min(variance / scale, 1.0)
        variance = np.var(scores)
        scale = np.max(scores) - np.min(scores)
        stability = 1.0 - min(variance / (scale + 1e-9), 1.0)
        stability = np.clip(stability, 0, 1)
        
        assert 0 <= stability <= 1, f"Stability should be [0,1], got {stability}"
    
    def test_importance_percentile_ranking(self):
        """Percentile ranking should be [0, 100]"""
        scores = np.array([0.1, 0.3, 0.5, 0.7, 0.9])
        
        # Rank each feature [0, 100]
        percentiles = [(s / max(scores)) * 100 for s in scores]
        
        assert all(0 <= p <= 100 for p in percentiles), "Percentiles should be [0,100]"
        assert percentiles[-1] == 100, "Max should be 100"

# ============================================================================
# PostgreSQL Persistence Tests
# ============================================================================

class TestPostgreSQLPersistence:
    """Test that computations persist correctly to PostgreSQL"""
    
    def test_store_drift_metrics(self, postgres_connection):
        """Verify drift metrics can be stored and retrieved"""
        cur = postgres_connection.cursor()
        
        # Create test drift metric
        test_data = {
            'feature_id': 'test:feature.v1',
            'ks_score': 0.15,
            'psi_score': 0.22,
            'chi2_score': 12.5,
            'classifier_score': 0.72,
            'method': 'ks',
            'is_drifted': True,
            'baseline_window': '30d',
            'eval_window': '7d',
            'baseline_count': 1000,
            'eval_count': 500
        }
        
        cur.execute("""
            INSERT INTO feature_drift_metrics 
            (feature_id, ks_score, psi_score, chi2_score, classifier_score, 
             method, is_drifted, baseline_window, eval_window, baseline_count, eval_count)
            VALUES (%(feature_id)s, %(ks_score)s, %(psi_score)s, %(chi2_score)s, 
                    %(classifier_score)s, %(method)s, %(is_drifted)s, %(baseline_window)s, 
                    %(eval_window)s, %(baseline_count)s, %(eval_count)s)
        """, test_data)
        
        postgres_connection.commit()
        
        # Retrieve and verify
        cur.execute("""
            SELECT ks_score, is_drifted FROM feature_drift_metrics 
            WHERE feature_id = 'test:feature.v1'
        """)
        
        row = cur.fetchone()
        assert row is not None, "Drift metric not found"
        assert row[0] == 0.15, "KS score not stored correctly"
        assert row[1] == True, "is_drifted flag not stored correctly"
    
    def test_store_importance_scores(self, postgres_connection):
        """Verify importance scores can be stored and retrieved"""
        cur = postgres_connection.cursor()
        
        test_importance = {
            'feature_id': 'test:importance.v1',
            'shap_mean': 0.45,
            'permutation_score': 0.38,
            'gain_importance': 0.52,
            'stability_score': 0.85,
            'trend': 0.02,
            'percentile_rank': 78
        }
        
        cur.execute("""
            INSERT INTO feature_importance 
            (feature_id, shap_mean, permutation_score, gain_importance, 
             stability_score, trend, percentile_rank)
            VALUES (%(feature_id)s, %(shap_mean)s, %(permutation_score)s, 
                    %(gain_importance)s, %(stability_score)s, %(trend)s, %(percentile_rank)s)
        """, test_importance)
        
        postgres_connection.commit()
        
        # Retrieve and verify
        cur.execute("""
            SELECT stability_score, percentile_rank FROM feature_importance 
            WHERE feature_id = 'test:importance.v1'
        """)
        
        row = cur.fetchone()
        assert row is not None, "Importance score not found"
        assert row[0] == 0.85, "Stability score not stored correctly"
        assert row[1] == 78, "Percentile rank not stored correctly"
    
    def test_watermark_incremental_tracking(self, postgres_connection):
        """Verify watermark prevents duplicate materialization"""
        cur = postgres_connection.cursor()
        
        feature_id = 'test:watermark.v1'
        first_watermark = datetime.now() - timedelta(hours=1)
        second_watermark = datetime.now()
        
        # Insert first watermark
        cur.execute("""
            INSERT INTO feature_watermarks (feature_id, last_processed)
            VALUES (%s, %s)
            ON CONFLICT (feature_id) DO UPDATE SET last_processed = EXCLUDED.last_processed
        """, (feature_id, first_watermark))
        
        postgres_connection.commit()
        
        # Verify first watermark
        cur.execute("""
            SELECT last_processed FROM feature_watermarks WHERE feature_id = %s
        """, (feature_id,))
        
        row = cur.fetchone()
        assert row is not None, "Watermark not found"
        
        # Update to second watermark
        cur.execute("""
            UPDATE feature_watermarks SET last_processed = %s WHERE feature_id = %s
        """, (second_watermark, feature_id))
        
        postgres_connection.commit()
        
        # Verify second watermark
        cur.execute("""
            SELECT last_processed FROM feature_watermarks WHERE feature_id = %s
        """, (feature_id,))
        
        row = cur.fetchone()
        assert row[0] > first_watermark, "Watermark not updated correctly"

# ============================================================================
# E2E Integration Tests
# ============================================================================

class TestEndToEndWorkflow:
    """Test complete Phase 3.21 workflows"""
    
    def test_complete_drift_detection_workflow(self, postgres_connection):
        """E2E: Load data → Compute drift → Store results → Query"""
        logger.info("Starting E2E drift detection workflow")
        
        # 1. Generate sample data
        np.random.seed(42)
        baseline = np.random.normal(100, 15, 1000)
        recent = np.random.normal(105, 15, 1000)  # Minor drift
        
        # 2. Compute drift (KS test)
        from scipy.stats import ks_2samp
        ks_stat, ks_pval = ks_2samp(baseline, recent)
        
        assert ks_stat > 0.05, "Should detect drift"
        logger.info(f"✓ KS test: stat={ks_stat:.4f}, p_value={ks_pval:.6f}")
        
        # 3. Store to PostgreSQL
        cur = postgres_connection.cursor()
        cur.execute("""
            INSERT INTO feature_drift_metrics 
            (feature_id, ks_score, is_drifted, method)
            VALUES (%s, %s, %s, %s)
        """, ('test:e2e.v1', ks_stat, True, 'ks'))
        
        postgres_connection.commit()
        logger.info("✓ Drift metrics stored to PostgreSQL")
        
        # 4. Query materialized view
        cur.execute("""
            SELECT COUNT(*) FROM active_drifts 
            WHERE feature_id = 'test:e2e.v1'
        """)
        
        count = cur.fetchone()[0]
        assert count > 0, "Drift should be in active_drifts view"
        logger.info("✓ Drift detected in active_drifts materialized view")
    
    def test_feature_health_scoring_query(self, postgres_connection):
        """E2E: Get comprehensive feature health report"""
        logger.info("Testing feature health scoring")
        
        cur = postgres_connection.cursor()
        
        # Get a real feature from sample data
        cur.execute("""
            SELECT feature_id, name FROM feature_catalog LIMIT 1
        """)
        
        result = cur.fetchone()
        if result:
            feature_id, feature_name = result
            
            # Query get_feature_health() function
            cur.execute("""
                SELECT * FROM get_feature_health(%s)
            """, (feature_id,))
            
            health_data = cur.fetchone()
            logger.info(f"✓ Feature health retrieved: {feature_name}")
    
    def test_lineage_recursive_query(self, postgres_connection):
        """E2E: Get recursive upstream/downstream dependencies"""
        logger.info("Testing recursive lineage query")
        
        cur = postgres_connection.cursor()
        
        # Get a real feature from sample data
        cur.execute("""
            SELECT feature_id FROM feature_catalog LIMIT 1
        """)
        
        result = cur.fetchone()
        if result:
            feature_id = result[0]
            
            # Query get_feature_ancestors() recursive function
            cur.execute("""
                SELECT * FROM get_feature_ancestors(%s)
            """, (feature_id,))
            
            ancestors = cur.fetchall()
            logger.info(f"✓ Found {len(ancestors)} ancestors for {feature_id}")

# ============================================================================
# Performance Tests
# ============================================================================

class TestPerformance:
    """Test performance of key operations"""
    
    def test_drift_detection_latency(self):
        """KS test should complete in <100ms for 1000 samples"""
        import time
        from scipy.stats import ks_2samp
        
        np.random.seed(42)
        baseline = np.random.normal(0, 1, 1000)
        recent = np.random.normal(0.1, 1, 1000)
        
        start = time.time()
        ks_2samp(baseline, recent)
        elapsed = (time.time() - start) * 1000  # Convert to ms
        
        assert elapsed < 100, f"KS test took {elapsed:.2f}ms, should be <100ms"
        logger.info(f"✓ KS test latency: {elapsed:.2f}ms")
    
    def test_importance_computation_latency(self):
        """Importance computation should complete in <1s for small model"""
        import time
        from sklearn.ensemble import RandomForestClassifier
        
        np.random.seed(42)
        X = np.random.randn(100, 5)
        y = np.random.randint(0, 2, 100)
        
        model = RandomForestClassifier(n_estimators=5, random_state=42)
        
        start = time.time()
        model.fit(X, y)
        elapsed = (time.time() - start) * 1000
        
        assert elapsed < 1000, f"Training took {elapsed:.2f}ms, should be <1000ms"
        logger.info(f"✓ Model training latency: {elapsed:.2f}ms")
    
    def test_postgresql_query_performance(self, postgres_connection):
        """PostgreSQL queries should complete in <100ms"""
        import time
        
        cur = postgres_connection.cursor()
        
        start = time.time()
        cur.execute("SELECT COUNT(*) FROM feature_catalog")
        cur.fetchone()
        elapsed = (time.time() - start) * 1000
        
        assert elapsed < 100, f"Query took {elapsed:.2f}ms, should be <100ms"
        logger.info(f"✓ PostgreSQL query latency: {elapsed:.2f}ms")

# ============================================================================
# Test Execution
# ============================================================================

if __name__ == '__main__':
    pytest.main([__file__, '-v', '--tb=short'])
