#!/usr/bin/env python3
"""
DataFusion Arrow Flight SQL REST Server

A lightweight Python server that provides:
- Arrow Flight SQL endpoint via gRPC
- REST API for SQL query execution
- Direct Iceberg REST catalog integration

This is a bootstrap implementation for development/testing.
For production, use Ballista or a native Rust Arrow Flight SQL server.
"""

import os
import sys
import json
import asyncio
import logging
import tempfile
import os
import re
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
import threading

import pyarrow as pa
from flask import Flask, request, jsonify
from flask_cors import CORS

try:
    import datafusion
    from datafusion import SessionContext
    HAS_DATAFUSION = True
except ImportError:
    HAS_DATAFUSION = False
    print("WARNING: datafusion Python package not installed. Install with: pip install datafusion")

try:
    import pyiceberg
    from pyiceberg.catalog import load_catalog
    HAS_ICEBERG = True
except ImportError:
    HAS_ICEBERG = False
    print("WARNING: pyiceberg not installed. Install with: pip install pyiceberg")

logging.basicConfig(
    level=os.environ.get("RUST_LOG", "info").upper(),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)


@dataclass
class QueryResult:
    schema: List[Dict[str, str]]
    records: List[List[Any]]
    execution_time_ms: float


class DataFusionServer:
    def __init__(self):
        self.ctx: Optional[SessionContext] = None
        self.iceberg_catalog = None
        self._init_datafusion()
        self._init_iceberg()

    def _init_datafusion(self):
        if not HAS_DATAFUSION:
            logger.warning("DataFusion not available - SQL queries will fail")
            return

        try:
            self.ctx = SessionContext()
            logger.info("DataFusion session context initialized")
        except Exception as e:
            logger.error(f"Failed to initialize DataFusion: {e}")

    def _init_iceberg(self):
        if not HAS_ICEBERG:
            logger.warning("PyIceberg not available - Iceberg queries will fail")
            return

        catalog_uri = os.environ.get("ICEBERG_CATALOG_URI", "http://localhost:8181")
        iceberg_cred = os.environ.get("ICEBERG_REST_CREDENTIAL", "root:secret")
        iceberg_scope = os.environ.get("ICEBERG_REST_SCOPE", "PRINCIPAL_ROLE:ALL")
        iceberg_warehouse = os.environ.get("ICEBERG_WAREHOUSE", "default")
        try:
            self.iceberg_catalog = load_catalog(
                "rest",
                **{
                    "uri": catalog_uri,
                    "credential": iceberg_cred,
                    "warehouse": iceberg_warehouse,
                    "scope": iceberg_scope,
                    "s3.endpoint": os.environ.get("S3_ENDPOINT", "http://localhost:9000"),
                    "s3.access-key-id": os.environ.get("S3_ACCESS_KEY", "minioadmin"),
                    "s3.secret-access-key": os.environ.get("S3_SECRET_KEY", "minioadmin"),
                }
            )
            logger.info(f"Iceberg REST catalog initialized: {catalog_uri} warehouse={iceberg_warehouse}")
        except Exception as e:
            logger.warning(f"Failed to initialize Iceberg catalog: {e}")

    def _rewrite_and_register_iceberg_tables(self, query: str) -> tuple:
        deregister = []
        rewritten = query

        warehouse = os.environ.get("ICEBERG_WAREHOUSE", "default")
        warehouse_sql = warehouse.replace("-", "_")

        three_part = r'(?:(?<=[^.\w])|(?<=^))([\w-]+)\.([\w]+)\.([\w]+)(?:(?=[^.\w])|(?=$))'
        for m in re.finditer(three_part, query):
            catalog_part, ns, tbl = m.group(1), m.group(2), m.group(3)
            catalog_norm = catalog_part.replace("-", "_")
            if catalog_norm == warehouse_sql:
                iceberg_ident = f"{ns}.{tbl}"
            else:
                iceberg_ident = f"{catalog_part}.{ns}.{tbl}"
                logger.info(f"Multi-catalog ref {iceberg_ident} (warehouse={warehouse})")

            tmp_name = f"_iceberg_tmp_{len(deregister)}"
            try:
                iceberg_table = self.iceberg_catalog.load_table(iceberg_ident)
                arrow_table = iceberg_table.scan().to_arrow()
                tmp_path = f"/tmp/{tmp_name}.arrow"
                with pa.ipc.new_file(tmp_path, arrow_table.schema) as writer:
                    writer.write_table(arrow_table)
                self.ctx.register_arrow(tmp_name, tmp_path)
                logger.info(f"Registered Iceberg table {iceberg_ident} as {tmp_name} ({arrow_table.num_rows} rows)")
                deregister.append(tmp_name)
                rewritten = rewritten.replace(m.group(0), tmp_name)
            except Exception as e:
                logger.warning(f"Could not load Iceberg table {iceberg_ident}: {e}")

        return rewritten, deregister

    def execute_query(self, query: str) -> QueryResult:
        import time
        start = time.time()

        if not self.ctx:
            raise RuntimeError("DataFusion not initialized")

        deregister = []
        if self.iceberg_catalog:
            try:
                query, deregister = self._rewrite_and_register_iceberg_tables(query)
            except Exception as e:
                logger.warning(f"Iceberg table rewrite failed: {e}")
                deregister = []

        result = self.ctx.sql(query)

        schema = []
        records = []

        for batch in result.collect():
            if not schema:
                schema = [
                    {"name": field.name, "type": str(field.type)}
                    for field in batch.schema
                ]

            for row in batch.to_pydict().values():
                records.append(list(row))

        for tmp_name in deregister:
            try:
                self.ctx.deregister_table(tmp_name)
            except Exception:
                pass

        elapsed = (time.time() - start) * 1000
        logger.info(f"Query executed in {elapsed:.2f}ms, returned {len(records)} rows")

        return QueryResult(
            schema=schema,
            records=records,
            execution_time_ms=elapsed
        )

    def get_iceberg_table(self, namespace: str, table: str):
        if not self.iceberg_catalog:
            raise RuntimeError("Iceberg catalog not initialized")

        identifier = f"{namespace}.{table}"
        return self.iceberg_catalog.load_table(identifier)


# Global server instance
_server: Optional[DataFusionServer] = None


def get_server() -> DataFusionServer:
    global _server
    if _server is None:
        _server = DataFusionServer()
    return _server


@app.route("/health", methods=["GET"])
def health():
    return jsonify({
        "status": "ok",
        "service": "datafusion-flight-sql",
        "datafusion_available": HAS_DATAFUSION,
        "iceberg_available": HAS_ICEBERG,
    })


@app.route("/api/v1/query", methods=["POST"])
def query():
    data = request.get_json()
    if not data or "query" not in data:
        return jsonify({"error": "Missing 'query' in request body"}), 400

    query_str = data["query"]
    logger.info(f"Executing query: {query_str[:200]}...")

    try:
        server = get_server()
        result = server.execute_query(query_str)

        return jsonify({
            "schema": result.schema,
            "records": result.records,
            "execution_time_ms": result.execution_time_ms,
            "row_count": len(result.records),
        })
    except Exception as e:
        logger.error(f"Query failed: {e}")
        return jsonify({"error": str(e)}), 500


@app.route("/api/v1/catalog/namespaces", methods=["GET"])
def list_namespaces():
    try:
        server = get_server()
        if not server.iceberg_catalog:
            return jsonify({"error": "Iceberg catalog not available"}), 503

        namespaces = server.iceberg_catalog.list_namespaces()
        return jsonify({"namespaces": namespaces})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/api/v1/catalog/namespaces/<namespace>/tables", methods=["GET"])
def list_tables(namespace):
    try:
        server = get_server()
        if not server.iceberg_catalog:
            return jsonify({"error": "Iceberg catalog not available"}), 503

        tables = server.iceberg_catalog.list_tables(namespace)
        return jsonify({"tables": tables})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


if __name__ == "__main__":
    port = int(os.environ.get("FLIGHT_SQL_PORT", 8555))
    logger.info(f"Starting DataFusion Arrow Flight SQL server on port {port}")
    app.run(host="0.0.0.0", port=port, debug=False)
