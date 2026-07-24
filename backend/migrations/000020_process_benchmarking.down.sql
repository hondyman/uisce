-- Rollback script for Process Benchmarking System

DROP TRIGGER IF EXISTS update_bp_gap_analysis_updated_at ON bp_gap_analysis;
DROP TRIGGER IF EXISTS update_bp_peer_groups_updated_at ON bp_peer_groups;
DROP TRIGGER IF EXISTS update_bp_best_practices_updated_at ON bp_best_practices;
DROP TRIGGER IF EXISTS update_bp_performance_scores_updated_at ON bp_performance_scores;

DROP FUNCTION IF EXISTS update_bp_updated_at();

DROP TABLE IF EXISTS bp_gap_analysis;
DROP TABLE IF EXISTS bp_peer_group_members;
DROP TABLE IF EXISTS bp_peer_groups;
DROP TABLE IF EXISTS bp_best_practices;
DROP TABLE IF EXISTS bp_performance_scores;
DROP TABLE IF EXISTS bp_industry_benchmarks;
