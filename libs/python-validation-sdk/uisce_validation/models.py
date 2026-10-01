from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional
from datetime import datetime

@dataclass
class ViolationRecord:
    rule_id: str
    rule_key: str
    rule_name: str
    severity: str  # "BLOCK" | "WARN"
    message: str
    rule_version: str = "1"
    bo_key: str = ""
    record_id: Optional[str] = None
    fields: List[str] = field(default_factory=list)
    context: Dict[str, Any] = field(default_factory=dict)
    write_blocked: bool = False
    rule_error: bool = False

@dataclass
class RuleErrorEntry:
    rule_id: str
    rule_key: str
    rule_name: str
    error: str
    rule_version: str = "1"

@dataclass
class RecordEvaluationResult:
    passed: bool
    violations: List[ViolationRecord] = field(default_factory=list)
    rule_errors: List[RuleErrorEntry] = field(default_factory=list)
    rules_evaluated: int = 0
    snapshot_id: str = ""
    context_required: bool = False
    missing_fields: List[str] = field(default_factory=list)

@dataclass
class BatchRecordResult:
    index: int
    record_id: str
    passed: bool
    violations: List[ViolationRecord] = field(default_factory=list)
    rule_errors: List[RuleErrorEntry] = field(default_factory=list)
    context_required: bool = False
    missing_fields: List[str] = field(default_factory=list)

@dataclass
class BatchEvaluationReport:
    total_records: int
    passed_records: int
    failed_records: int
    blocked_records: int
    snapshot_id: str
    results: List[BatchRecordResult] = field(default_factory=list)

@dataclass
class ChainVerificationResult:
    valid: bool
    tenant_id: str
    seq_from: int
    seq_to: int
    total_rows: int
    anchors_checked: int
    errors: List[str] = field(default_factory=list)
