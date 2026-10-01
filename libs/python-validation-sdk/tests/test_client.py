import unittest
from unittest.mock import patch, MagicMock
import io
import json
import urllib.error
from uisce_validation import ValidationClient, ValidationClientError

class TestValidationClient(unittest.TestCase):
    def setUp(self):
        self.client = ValidationClient(
            base_url="http://mock-server:8080",
            tenant_id="tenant-py-123",
            auth_token="mock-jwt-token",
        )

    @patch("urllib.request.urlopen")
    def test_evaluate_record_success(self, mock_urlopen):
        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps({
            "passed": True,
            "violations": [],
            "rule_errors": [],
            "rules_evaluated": 5,
            "snapshot_id": "snap-001",
        }).encode("utf-8")
        mock_urlopen.return_value.__enter__.return_value = mock_response

        res = self.client.evaluate_record("order", {"Qty": 100, "Price": 50.0})
        self.assertTrue(res.passed)
        self.assertEqual(res.rules_evaluated, 5)
        self.assertEqual(res.snapshot_id, "snap-001")
        self.assertEqual(len(res.violations), 0)

    @patch("urllib.request.urlopen")
    def test_evaluate_record_with_violations(self, mock_urlopen):
        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps({
            "passed": False,
            "violations": [
                {
                    "rule_id": "r-1",
                    "rule_key": "limit_check",
                    "rule_name": "Max Qty Check",
                    "rule_version": "2",
                    "severity": "BLOCK",
                    "message": "Qty exceeds 1000",
                    "write_blocked": True,
                }
            ],
            "rule_errors": [],
            "rules_evaluated": 3,
            "snapshot_id": "snap-002",
        }).encode("utf-8")
        mock_urlopen.return_value.__enter__.return_value = mock_response

        res = self.client.evaluate_record("order", {"Qty": 5000})
        self.assertFalse(res.passed)
        self.assertEqual(len(res.violations), 1)
        self.assertEqual(res.violations[0].rule_key, "limit_check")
        self.assertEqual(res.violations[0].rule_version, "2")
        self.assertTrue(res.violations[0].write_blocked)

    @patch("urllib.request.urlopen")
    def test_evaluate_batch(self, mock_urlopen):
        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps({
            "total_records": 2,
            "passed_records": 1,
            "failed_records": 1,
            "blocked_records": 1,
            "snapshot_id": "snap-batch-1",
            "results": [
                {"index": 0, "record_id": "rec-1", "passed": True, "violations": []},
                {"index": 1, "record_id": "rec-2", "passed": False, "violations": [
                    {"rule_id": "r-1", "rule_key": "k-1", "rule_name": "R1", "severity": "BLOCK", "message": "Failed"}
                ]},
            ]
        }).encode("utf-8")
        mock_urlopen.return_value.__enter__.return_value = mock_response

        report = self.client.evaluate_batch("order", [{"Qty": 10}, {"Qty": 9999}])
        self.assertEqual(report.total_records, 2)
        self.assertEqual(report.passed_records, 1)
        self.assertEqual(report.failed_records, 1)
        self.assertEqual(report.blocked_records, 1)
        self.assertEqual(len(report.results), 2)
        self.assertTrue(report.results[0].passed)
        self.assertFalse(report.results[1].passed)

    @patch("urllib.request.urlopen")
    def test_verify_chain(self, mock_urlopen):
        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps({
            "valid": True,
            "tenant_id": "tenant-py-123",
            "seq_from": 1,
            "seq_to": 100,
            "total_rows": 100,
            "anchors_checked": 2,
        }).encode("utf-8")
        mock_urlopen.return_value.__enter__.return_value = mock_response

        res = self.client.verify_chain(1, 100)
        self.assertTrue(res.valid)
        self.assertEqual(res.total_rows, 100)
        self.assertEqual(res.anchors_checked, 2)

if __name__ == "__main__":
    unittest.main()
