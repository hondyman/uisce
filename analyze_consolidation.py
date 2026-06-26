#!/usr/bin/env python3
"""
Schema Consolidation Analysis Tool
Analyzes the metrics_registry and dax_functions tables across domain schemas
"""

import sys
import json
from datetime import datetime
import subprocess


class SchemaAnalyzer:
    """Analyze domain schemas in PostgreSQL alpha database"""
    
    DB_CONNECTION = "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
    
    DOMAIN_SCHEMAS = [
        'banking', 'capital_markets', 'currency_fx', 'financial_services',
        'fixed_income', 'foffice', 'hdb_catalog', 'healthcare', 'hld',
        'insurance', 'investment_accounting', 'regulatory', 'report_sys',
        'retail', 'semantic_layer', 'sml', 'unified_financial_services',
        'wealth_management'
    ]
    
    def __init__(self):
        self.results = {
            'metrics_registry': {},
            'dax_functions': {},
            'summary': {}
        }
    
    def run_query(self, query: str) -> str:
        """Execute SQL query and return results"""
        try:
            result = subprocess.run(
                ['psql', '-h', 'localhost', '-U', 'postgres', '-d', 'alpha', '-c', query],
                capture_output=True,
                text=True,
                timeout=10
            )
            if result.returncode != 0:
                print(f"❌ Query failed: {result.stderr}", file=sys.stderr)
                return ""
            return result.stdout.strip()
        except Exception as e:
            print(f"❌ Error executing query: {e}", file=sys.stderr)
            return ""
    
    def analyze_metrics_registry(self):
        """Analyze metrics_registry tables across schemas"""
        print("📊 Analyzing metrics_registry tables...")
        
        query = """
        SELECT table_schema, COUNT(*) as record_count
        FROM information_schema.tables t
        WHERE table_name = 'metrics_registry'
        GROUP BY table_schema
        ORDER BY table_schema;
        """
        
        output = self.run_query(query)
        print(output)
        
    def analyze_dax_functions(self):
        """Analyze dax_functions tables across schemas"""
        print("\n📊 Analyzing dax_functions tables...")
        
        query = """
        SELECT table_schema, COUNT(*) as has_dax_functions
        FROM information_schema.tables t
        WHERE table_name = 'dax_functions'
        GROUP BY table_schema
        ORDER BY table_schema;
        """
        
        output = self.run_query(query)
        print(output)
    
    def get_consolidation_summary(self):
        """Generate consolidation summary"""
        print("\n📈 Consolidation Summary")
        print("=" * 50)
        
        # Total metrics records
        query = """
        SELECT COUNT(*) FROM banking.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM capital_markets.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM currency_fx.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM financial_services.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM fixed_income.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM healthcare.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM insurance.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM investment_accounting.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM regulatory.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM retail.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM unified_financial_services.metrics_registry
        UNION ALL
        SELECT COUNT(*) FROM wealth_management.metrics_registry;
        """
        
        output = self.run_query(query)
        print("\nMetrics Registry record counts per schema:")
        print(output)
    
    def generate_migration_report(self, output_file: str = "migration_report.json"):
        """Generate detailed migration report"""
        print(f"\n📄 Generating migration report: {output_file}")
        
        report = {
            'generated_at': datetime.now().isoformat(),
            'database': 'alpha (localhost:5432)',
            'consolidation_target': 'public schema',
            'tables_to_consolidate': {
                'metrics_registry': {
                    'source_schemas': [],
                    'total_records': 0,
                    'consolidated_to': 'public.metrics_registry',
                    'new_columns': ['schema_domain (VARCHAR 100)']
                },
                'dax_functions': {
                    'source_schemas': [],
                    'total_records': 0,
                    'consolidated_to': 'public.dax_functions',
                    'new_columns': ['schema_domain (VARCHAR 100)']
                }
            },
            'migration_steps': [
                'Create public.metrics_registry with schema_domain column',
                'Create public.dax_functions with schema_domain column',
                'Migrate all metrics_registry data with domain tracking',
                'Migrate all dax_functions data with domain tracking',
                'Create backwards-compatibility views (optional)',
                'Update application code',
                'Drop old tables after code migration'
            ],
            'benefits': [
                'Single source of truth for metrics and functions',
                'Reduced schema duplication (17→1 for these tables)',
                'Simplified maintenance and governance',
                'Better performance with consolidated indexes',
                'Backwards compatible views during transition'
            ]
        }
        
        with open(output_file, 'w') as f:
            json.dump(report, f, indent=2)
        
        print(f"✅ Report saved to {output_file}")
    
    def run_all(self):
        """Run all analyses"""
        print("\n" + "=" * 60)
        print("🚀 SCHEMA CONSOLIDATION ANALYSIS")
        print("=" * 60)
        
        self.analyze_metrics_registry()
        self.analyze_dax_functions()
        self.get_consolidation_summary()
        self.generate_migration_report()
        
        print("\n" + "=" * 60)
        print("✅ Analysis complete!")
        print("=" * 60)
        print("\n📖 Next steps:")
        print("  1. Review migration_report.json")
        print("  2. Read CONSOLIDATION_PLAN.md for detailed strategy")
        print("  3. Test: psql -h localhost -U postgres -d alpha -f migrations/consolidate_metrics_and_dax.sql")
        print("  4. Use find_schema_references.sh to identify code changes needed")
        print("")


if __name__ == '__main__':
    analyzer = SchemaAnalyzer()
    analyzer.run_all()
