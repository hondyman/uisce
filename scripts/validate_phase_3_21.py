#!/usr/bin/env python3
"""
Phase 3.21 Schema & Component Validation Script
Comprehensive validation of all Phase 3.21 infrastructure components.
"""

import psycopg2
import psycopg2.extras
import sys
import logging
from typing import Dict, List, Tuple
from datetime import datetime

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class Phase321Validator:
    """Validates Phase 3.21 schema and components"""
    
    def __init__(self, host: str, port: int, user: str, password: str, database: str):
        self.conn_string = f"postgresql://{user}:{password}@{host}:{port}/{database}"
        self.conn = None
        self.results = {}
    
    def connect(self):
        """Connect to PostgreSQL"""
        try:
            self.conn = psycopg2.connect(self.conn_string)
            logger.info("✓ Connected to PostgreSQL")
            self.results['database_connection'] = True
        except Exception as e:
            logger.error(f"✗ Failed to connect: {str(e)}")
            self.results['database_connection'] = False
            return False
        return True
    
    def check_tables(self) -> bool:
        """Verify all 10 core tables exist"""
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
        
        cur = self.conn.cursor()
        found_tables = []
        
        for table in expected_tables:
            cur.execute(f"""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_name = '{table}'
                )
            """)
            exists = cur.fetchone()[0]
            if exists:
                found_tables.append(table)
                logger.info(f"✓ Table {table} exists")
            else:
                logger.error(f"✗ Table {table} missing")
        
        self.results['tables'] = {
            'total': len(expected_tables),
            'found': len(found_tables),
            'status': len(found_tables) == len(expected_tables)
        }
        
        return len(found_tables) == len(expected_tables)
    
    def check_indexes(self) -> bool:
        """Verify indexes are created"""
        cur = self.conn.cursor()
        cur.execute("""
            SELECT COUNT(*) FROM pg_indexes 
            WHERE schemaname = 'public' AND indexname LIKE 'idx_%'
        """)
        count = cur.fetchone()[0]
        
        self.results['indexes'] = {
            'count': count,
            'status': count >= 30
        }
        
        logger.info(f"{'✓' if count >= 30 else '✗'} Found {count} indexes (target: 30+)")
        return count >= 30
    
    def check_views(self) -> bool:
        """Verify views are created"""
        expected_views = [
            'feature_catalog_active',
            'quality_check_failures',
            'failing_tests',
            'pending_approvals',
            'active_drifts',
            'top_features_by_model',
            'feature_lineage_ancestors'
        ]
        
        cur = self.conn.cursor(cursor_factory=psycopg2.extras.DictCursor)
        found_views = []
        
        for view in expected_views:
            cur.execute(f"""
                SELECT EXISTS (
                    SELECT FROM information_schema.views 
                    WHERE table_name = '{view}'
                )
            """)
            exists = cur.fetchone()[0]
            if exists:
                found_views.append(view)
                logger.info(f"✓ View {view} exists")
            else:
                logger.error(f"✗ View {view} missing")
        
        self.results['views'] = {
            'total': len(expected_views),
            'found': len(found_views),
            'status': len(found_views) == len(expected_views)
        }
        
        return len(found_views) == len(expected_views)
    
    def check_sample_data(self) -> bool:
        """Verify sample data is loaded"""
        cur = self.conn.cursor()
        
        queries = {
            'features': 'SELECT COUNT(*) FROM feature_catalog',
            'watermarks': 'SELECT COUNT(*) FROM feature_watermarks',
            'drifts': 'SELECT COUNT(*) FROM feature_drift_metrics',
            'importance': 'SELECT COUNT(*) FROM feature_importance',
            'tests': 'SELECT COUNT(*) FROM feature_test_cases'
        }
        
        data_counts = {}
        for name, query in queries.items():
            cur.execute(query)
            count = cur.fetchone()[0]
            data_counts[name] = count
            status = "✓" if count > 0 else "✗"
            logger.info(f"{status} {name}: {count} records")
        
        self.results['sample_data'] = data_counts
        return all(count > 0 for count in data_counts.values())
    
    def check_constraints(self) -> bool:
        """Verify constraints are defined"""
        cur = self.conn.cursor()
        cur.execute("""
            SELECT COUNT(*) FROM information_schema.table_constraints 
            WHERE table_schema = 'public' 
            AND constraint_type IN ('FOREIGN KEY', 'PRIMARY KEY', 'UNIQUE', 'CHECK')
        """)
        count = cur.fetchone()[0]
        
        self.results['constraints'] = {
            'count': count,
            'status': count >= 20
        }
        
        logger.info(f"{'✓' if count >= 20 else '✗'} Found {count} constraints (target: 20+)")
        return count >= 20
    
    def check_functions(self) -> bool:
        """Verify helper functions exist"""
        expected_functions = [
            'get_feature_health',
            'get_feature_ancestors',
            'update_feature_catalog_timestamp',
            'update_test_cases_timestamp'
        ]
        
        cur = self.conn.cursor()
        found_functions = []
        
        for func in expected_functions:
            cur.execute(f"""
                SELECT EXISTS (
                    SELECT FROM information_schema.routines 
                    WHERE routine_name = '{func}' AND routine_schema = 'public'
                )
            """)
            exists = cur.fetchone()[0]
            if exists:
                found_functions.append(func)
                logger.info(f"✓ Function {func} exists")
            else:
                logger.error(f"✗ Function {func} missing")
        
        self.results['functions'] = {
            'total': len(expected_functions),
            'found': len(found_functions),
            'status': len(found_functions) == len(expected_functions)
        }
        
        return len(found_functions) == len(expected_functions)
    
    def check_feature_health(self) -> bool:
        """Check health of sample features"""
        cur = self.conn.cursor(cursor_factory=psycopg2.extras.DictCursor)
        cur.execute("""
            SELECT 
                fc.feature_id,
                fc.name,
                fw.last_processed,
                fw.watermark_age_seconds,
                (SELECT COUNT(*) FROM feature_drift_metrics WHERE feature_id = fc.feature_id AND is_drifted) as active_drifts
            FROM feature_catalog fc
            LEFT JOIN feature_watermarks fw ON fc.feature_id = fw.feature_id
            LIMIT 5
        """)
        
        features = cur.fetchall()
        health_data = []
        
        for feat in features:
            health_data.append({
                'feature_id': feat['feature_id'],
                'name': feat['name'],
                'last_processed': feat['last_processed'],
                'age_seconds': feat['watermark_age_seconds'],
                'active_drifts': feat['active_drifts']
            })
            logger.info(f"✓ Feature {feat['feature_id']}: age={feat['watermark_age_seconds']}s, drifts={feat['active_drifts']}")
        
        self.results['feature_health'] = health_data
        return len(features) > 0
    
    def run_all_checks(self) -> Dict:
        """Run all validation checks"""
        logger.info("=" * 70)
        logger.info("Phase 3.21 Schema & Component Validation")
        logger.info("=" * 70)
        
        if not self.connect():
            return self.results
        
        checks = [
            ('Tables', self.check_tables),
            ('Indexes', self.check_indexes),
            ('Views', self.check_views),
            ('Sample Data', self.check_sample_data),
            ('Constraints', self.check_constraints),
            ('Helper Functions', self.check_functions),
            ('Feature Health', self.check_feature_health)
        ]
        
        for check_name, check_func in checks:
            try:
                logger.info(f"\nRunning: {check_name}")
                result = check_func()
                self.results[f'{check_name}_passed'] = result
            except Exception as e:
                logger.error(f"✗ {check_name} check failed: {str(e)}")
                self.results[f'{check_name}_passed'] = False
        
        self.conn.close()
        self.print_summary()
        
        return self.results
    
    def print_summary(self):
        """Print validation summary"""
        logger.info("\n" + "=" * 70)
        passed_checks = sum(1 for k, v in self.results.items() if k.endswith('_passed') and v)
        total_checks = sum(1 for k in self.results.keys() if k.endswith('_passed'))
        
        logger.info(f"VALIDATION SUMMARY: {passed_checks}/{total_checks} checks passed")
        
        if passed_checks == total_checks:
            logger.info("✓ All Phase 3.21 components validated successfully!")
            logger.info("Ready for production deployment.")
        else:
            logger.warning("✗ Some validations failed. Review logs above.")
        
        logger.info("=" * 70)

if __name__ == '__main__':
    import argparse
    
    parser = argparse.ArgumentParser(description='Validate Phase 3.21 schema')
    parser.add_argument('--host', default='localhost', help='PostgreSQL host')
    parser.add_argument('--port', type=int, default=5432, help='PostgreSQL port')
    parser.add_argument('--user', default='postgres', help='PostgreSQL user')
    parser.add_argument('--password', default='secret', help='PostgreSQL password')
    parser.add_argument('--database', default='semlayer', help='Database name')
    
    args = parser.parse_args()
    
    validator = Phase321Validator(args.host, args.port, args.user, args.password, args.database)
    results = validator.run_all_checks()
    
    # Exit with error if any checks failed
    if not all(v for k, v in results.items() if k.endswith('_passed')):
        sys.exit(1)
