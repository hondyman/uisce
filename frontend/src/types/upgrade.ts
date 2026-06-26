// types/upgrade.ts
// This file now imports from the generated types to ensure schema consistency

import type {
  UpgradeArtifacts,
  DiffReport,
  DiffSummary,
  CubeDiff,
  ViewDiff,
  GovernanceDiff,
  PreAggregationDiff,
  Change,
  AliasMap,
  AliasEntry,
} from './upgrade-generated';

export type {
  UpgradeArtifacts,
  DiffReport,
  DiffSummary,
  CubeDiff,
  ViewDiff,
  GovernanceDiff,
  PreAggregationDiff,
  Change,
  AliasMap,
  AliasEntry,
};

// Re-export common types for backward compatibility
export type DiffStatus = "added" | "removed" | "changed";

/** --------------------
 *  Golden Query Schema
 *  -------------------- */

// Golden Query definitions aligned with UI usage and backend API
export interface GoldenQuery {
  name: string;
  query?: string; // SQL text
  sql?: string;   // backward compatibility
  description?: string;
  tags?: string[];
}

export interface GoldenQueryResult {
  query_name: string;
  old_result: { rows?: unknown[]; execution_time_ms: number; error?: string };
  new_result: { rows?: unknown[]; execution_time_ms: number; error?: string };
  diff_analysis: {
    row_count_diff: number;
    execution_time_diff_ms: number;
    breaking_changes?: boolean;
    data_differences: unknown[];
  };
}

/** --------------------
 *  Extension Fix Schema
 *  -------------------- */

export interface ExtensionFix {
  file_path: string;
  fixes: ExtensionFixEntry[];
}

export interface ExtensionFixEntry {
  line_number: number;
  old_code: string;
  new_code: string;
  alias_used?: string;
  confidence: 'high' | 'medium' | 'low';
}

/** --------------------
 *  API Response Helpers
 *  -------------------- */

export interface UpgradeStatusResponse {
  core_version: string;
  status: "pending" | "ready" | "canary" | "active" | "rolled_back";
  warnings: string[];
  blockers: string[];
}

export interface DiffResponse {
  schema_version: string;
  changelog?: Array<{
    version: string;
    date: string;
    description: string;
    [k: string]: unknown;
  }>;
  report: DiffReport;
  aliases: AliasMap;
}

export interface UpgradeOverviewResponse {
  schema_version: string;
  changelog?: Array<{
    version: string;
    date: string;
    description: string;
  }>;
  report: DiffReport;
  aliases: AliasMap;
  status: UpgradeStatusResponse;
  ui_hints?: {
    needs_diff_review: boolean;
    needs_extension_fix: boolean;
    needs_query_run: boolean;
  };
}

export interface MultiUpgradeOverviewResponse {
  versions: UpgradeOverviewResponse[];
}
