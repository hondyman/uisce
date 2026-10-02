import json
import time
import urllib.request
import urllib.error
from typing import Any, Dict, List, Optional, Union
from .models import (
    ViolationRecord,
    RuleErrorEntry,
    RecordEvaluationResult,
    BatchRecordResult,
    BatchEvaluationReport,
    ChainVerificationResult,
)

class ValidationClientError(Exception):
    def __init__(self, message: str, status_code: Optional[int] = None, response_body: Optional[str] = None):
        super().__init__(message)
        self.status_code = status_code
        self.response_body = response_body

class ValidationClient:
    """
    Official Python Client for the Uisce Centralized Validation Rules Engine.
    Supports single-record evaluation, transactional batch evaluation, bitemporal queries,
    and Glassbox audit chain verification.
    """
    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        tenant_id: str = "",
        auth_token: Optional[str] = None,
        timeout_seconds: float = 30.0,
        max_retries: int = 3,
        backoff_factor: float = 0.5,
    ):
        self.base_url = base_url.rstrip("/")
        self.tenant_id = tenant_id
        self.auth_token = auth_token
        self.timeout = timeout_seconds
        self.max_retries = max_retries
        self.backoff_factor = backoff_factor

    def _headers(self, custom_headers: Optional[Dict[str, str]] = None) -> Dict[str, str]:
        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
        }
        if self.tenant_id:
            headers["X-Tenant-ID"] = self.tenant_id
        if self.auth_token:
            headers["Authorization"] = f"Bearer {self.auth_token}"
        if custom_headers:
            headers.update(custom_headers)
        return headers

    def _request(
        self,
        method: str,
        path: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
        custom_headers: Optional[Dict[str, str]] = None,
    ) -> Dict[str, Any]:
        url = f"{self.base_url}{path}"
        if params:
            query = "&".join(f"{k}={urllib.parse.quote(str(v))}" for k, v in params.items() if v is not None)
            if query:
                url = f"{url}?{query}"

        encoded_data = json.dumps(data).encode("utf-8") if data is not None else None

        last_err = None
        for attempt in range(self.max_retries + 1):
            req = urllib.request.Request(url, data=encoded_data, headers=self._headers(custom_headers), method=method)
            try:
                with urllib.request.urlopen(req, timeout=self.timeout) as response:
                    resp_body = response.read().decode("utf-8")
                    if not resp_body:
                        return {}
                    return json.loads(resp_body)
            except urllib.error.HTTPError as e:
                resp_body = e.read().decode("utf-8")
                # 422 Unprocessable Entity (e.g. valid evaluation with violations) returns body
                if e.code == 422 and resp_body:
                    try:
                        return json.loads(resp_body)
                    except Exception:
                        pass
                if e.code < 500 or attempt == self.max_retries:
                    raise ValidationClientError(f"HTTP {e.code}: {resp_body}", status_code=e.code, response_body=resp_body)
                last_err = e
            except urllib.error.URLError as e:
                if attempt == self.max_retries:
                    raise ValidationClientError(f"Connection error: {e.reason}")
                last_err = e

            sleep_time = self.backoff_factor * (2 ** attempt)
            time.sleep(sleep_time)

        raise ValidationClientError(f"Request failed after {self.max_retries} retries: {last_err}")

    def evaluate_record(
        self,
        bo_name: str,
        record: Dict[str, Any],
        record_id: Optional[str] = None,
        domain: Optional[str] = None,
        timing: Optional[str] = None,
    ) -> RecordEvaluationResult:
        """
        Evaluates active validation rules against a single record.
        """
        payload = {
            "tenant_id": self.tenant_id,
            "bo_name": bo_name,
            "record": record,
            "record_id": record_id,
            "domain": domain,
            "timing": timing,
        }
        res = self._request("POST", "/api/validation-rule-nodes/evaluate-record", data=payload)

        violations = [
            ViolationRecord(
                rule_id=v.get("rule_id", ""),
                rule_key=v.get("rule_key", ""),
                rule_name=v.get("rule_name", ""),
                severity=v.get("severity", "WARN"),
                message=v.get("message", ""),
                rule_version=str(v.get("rule_version", "1")),
                bo_key=v.get("bo_key", bo_name),
                record_id=v.get("record_id", record_id),
                fields=v.get("fields", []),
                context=v.get("context", {}),
                write_blocked=v.get("write_blocked", False),
                rule_error=v.get("rule_error", False),
            )
            for v in res.get("violations", [])
        ]

        rule_errors = [
            RuleErrorEntry(
                rule_id=e.get("rule_id", ""),
                rule_key=e.get("rule_key", ""),
                rule_name=e.get("rule_name", ""),
                error=e.get("error", ""),
                rule_version=str(e.get("rule_version", "1")),
            )
            for e in res.get("rule_errors", [])
        ]

        return RecordEvaluationResult(
            passed=res.get("passed", False),
            violations=violations,
            rule_errors=rule_errors,
            rules_evaluated=res.get("rules_evaluated", 0),
            snapshot_id=res.get("snapshot_id", ""),
            context_required=res.get("context_required", False),
            missing_fields=res.get("missing_fields", []),
        )

    def evaluate_batch(
        self,
        bo_name: str,
        records: List[Dict[str, Any]],
        domain: Optional[str] = None,
        timing: Optional[str] = None,
        as_of: Optional[str] = None,
    ) -> BatchEvaluationReport:
        """
        Evaluates a batch of records under a single transactional RuleSnapshot.
        """
        payload = {
            "tenant_id": self.tenant_id,
            "bo_name": bo_name,
            "records": records,
            "domain": domain,
            "timing": timing,
            "as_of": as_of,
        }
        res = self._request("POST", "/api/validation-rule-nodes/evaluate-batch", data=payload)

        results = []
        for r in res.get("results", []):
            viols = [
                ViolationRecord(
                    rule_id=v.get("rule_id", ""),
                    rule_key=v.get("rule_key", ""),
                    rule_name=v.get("rule_name", ""),
                    severity=v.get("severity", "WARN"),
                    message=v.get("message", ""),
                    rule_version=str(v.get("rule_version", "1")),
                    bo_key=v.get("bo_key", bo_name),
                    record_id=v.get("record_id"),
                    fields=v.get("fields", []),
                    context=v.get("context", {}),
                    write_blocked=v.get("write_blocked", False),
                    rule_error=v.get("rule_error", False),
                )
                for v in r.get("violations", [])
            ]
            r_errs = [
                RuleErrorEntry(
                    rule_id=e.get("rule_id", ""),
                    rule_key=e.get("rule_key", ""),
                    rule_name=e.get("rule_name", ""),
                    error=e.get("error", ""),
                    rule_version=str(e.get("rule_version", "1")),
                )
                for e in r.get("rule_errors", [])
            ]
            results.append(BatchRecordResult(
                index=r.get("index", 0),
                record_id=r.get("record_id", ""),
                passed=r.get("passed", False),
                violations=viols,
                rule_errors=r_errs,
                context_required=r.get("context_required", False),
                missing_fields=r.get("missing_fields", []),
            ))

        return BatchEvaluationReport(
            total_records=res.get("total_records", len(records)),
            passed_records=res.get("passed_records", 0),
            failed_records=res.get("failed_records", 0),
            blocked_records=res.get("blocked_records", 0),
            snapshot_id=res.get("snapshot_id", ""),
            results=results,
        )

    def verify_chain(self, seq_from: int = 1, seq_to: int = 10000) -> ChainVerificationResult:
        """
        Verifies the cryptographic Glassbox audit chain against anchored root hashes.
        """
        params = {"from": seq_from, "to": seq_to}
        res = self._request("GET", "/api/validation-rule-nodes/verify-chain", params=params)
        return ChainVerificationResult(
            valid=res.get("valid", False),
            tenant_id=res.get("tenant_id", self.tenant_id),
            seq_from=res.get("seq_from", seq_from),
            seq_to=res.get("seq_to", seq_to),
            total_rows=res.get("total_rows", 0),
            anchors_checked=res.get("anchors_checked", 0),
            errors=res.get("errors", []),
        )
