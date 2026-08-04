"""
Tests for customization_clusters ETL job.
Run with: pytest tests/ -v
"""

import json
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).parent.parent))
from jobs.customization_clusters import (
    _permissions_to_text,
    compute_pattern_hash,
    normalize_text_fields,
)


class TestPermissionsToText:
    def test_json_string_array(self):
        assert _permissions_to_text('["read", "write"]') == "read write"

    def test_already_list(self):
        assert _permissions_to_text(["read", "write"]) == "read write"

    def test_empty(self):
        assert _permissions_to_text("[]") == ""

    def test_invalid_json(self):
        assert _permissions_to_text("not json") == ""

    def test_mixed_types_in_json(self):
        assert _permissions_to_text('["read", 123]') == "read"


class TestComputePatternHash:
    def test_deterministic(self):
        h1 = compute_pattern_hash("role_viewer", "A viewer role", "read write")
        h2 = compute_pattern_hash("role_viewer", "A viewer role", "read write")
        assert h1 == h2

    def test_different_inputs_different_hash(self):
        h1 = compute_pattern_hash("role_viewer", "A viewer role", "read write")
        h2 = compute_pattern_hash("role_editor", "An editor role", "read write delete")
        assert h1 != h2

    def test_hash_length(self):
        h = compute_pattern_hash("role_viewer", "desc", "read write")
        assert len(h) == 32  # SHA-256 truncated to 32 chars


class TestNormalizeTextFields:
    def test_combined_text_lowercases(self):
        import pandas as pd
        df = pd.DataFrame([
            {
                "tenant_id": "t1",
                "entity_id": "e1",
                "action": "CREATE",
                "entity_type": "bp_roles",
                "name": "ROLE_VIEWER",
                "description": "A VIEWER ROLE",
                "permissions": '["READ", "WRITE"]',
                "created_at": "2024-01-01",
            }
        ])
        result = normalize_text_fields(df)
        assert result["combined_text"].iloc[0] == "role_viewer a viewer role read write"

    def test_handles_missing_name(self):
        import pandas as pd
        df = pd.DataFrame([
            {
                "tenant_id": "t1",
                "entity_id": "e1",
                "action": "CREATE",
                "entity_type": "bp_roles",
                "name": None,
                "description": "desc",
                "permissions": "[]",
                "created_at": "2024-01-01",
            }
        ])
        result = normalize_text_fields(df)
        assert "role_viewer" not in result["combined_text"].iloc[0]

    def test_permissions_text_sorted(self):
        import pandas as pd
        df = pd.DataFrame([
            {
                "tenant_id": "t1",
                "entity_id": "e1",
                "action": "CREATE",
                "entity_type": "bp_roles",
                "name": "role",
                "description": "",
                "permissions": '["z", "a", "m"]',
                "created_at": "2024-01-01",
            }
        ])
        result = normalize_text_fields(df)
        assert result["permissions_text"].iloc[0] == "a m z"
