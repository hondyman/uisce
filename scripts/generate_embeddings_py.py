#!/usr/bin/env python3
"""
Lightweight embedding generator that mirrors backend Go logic.
Usage:
  ./scripts/generate_embeddings_py.py --tenant=<TENANT_ID> --datasource=<DATASOURCE_ID> [--db='postgres://...'] [--api-key=KEY]

This script uses the Gemini embeddings endpoint used by the Go provider.
"""
import argparse
import os
import time
import json
import sys
from typing import List

try:
    import requests
    import psycopg2
    import psycopg2.extras
except Exception as e:
    print("Missing dependency; please run: pip install requests psycopg2-binary")
    raise

EMBEDDING_MODEL = "text-embedding-004"
GEMINI_EMBED_URL = "https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent"

NODE_TYPES = ('calculation', 'metric', 'measure', 'dimension', 'view', 'semantic_model', 'table')


def embed_text(api_key: str, text: str) -> List[float]:
    url = GEMINI_EMBED_URL % EMBEDDING_MODEL
    params = { 'key': api_key }
    body = {
        "content": {
            "parts": [ { "text": text } ]
        }
    }
    headers = { 'Content-Type': 'application/json' }
    resp = requests.post(url, params=params, json=body, headers=headers, timeout=60)
    if resp.status_code != 200:
        raise RuntimeError(f"Gemini embed API error {resp.status_code}: {resp.text}")
    j = resp.json()
    embedding = j.get('embedding')
    if not embedding:
        raise RuntimeError(f"No embedding in response: {j}")
    vals = embedding.get('values')
    if not isinstance(vals, list):
        raise RuntimeError(f"Invalid embedding values: {j}")
    return [float(x) for x in vals]


def build_text(row: dict) -> str:
    parts = []
    parts.append(f"Type: {row.get('node_type')}")
    parts.append(f"Name: {row.get('node_name')}")
    parts.append(f"Path: {row.get('qualified_path')}")
    desc = row.get('description')
    if desc:
        parts.append(f"Description: {desc}")
    props = row.get('properties')
    if props:
        props_str = props if isinstance(props, str) else json.dumps(props)
        if len(props_str) > 500:
            props_str = props_str[:500] + '...'
        parts.append(f"Properties: {props_str}")
    return "\n".join(parts)


def vector_to_postgres(vec: List[float]) -> str:
    return '[' + ','.join(f"{v:.6f}" for v in vec) + ']'


def process(tenant: str, datasource: str, db_url: str, api_key: str):
    conn = psycopg2.connect(db_url)
    cur = conn.cursor(cursor_factory=psycopg2.extras.DictCursor)

    select_q = f"""
        SELECT cn.id, cn.node_name, cn.qualified_path, cn.description, cnt.catalog_type_name as node_type, cn.properties::text
        FROM catalog_node cn
        LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
        WHERE cn.tenant_id = %s
          AND cn.tenant_datasource_id = %s
          AND cn.embedding IS NULL
        ORDER BY cn.created_at DESC
    """

    cur.execute(select_q, (tenant, datasource))
    rows = cur.fetchall()
    print(f"Found {len(rows)} nodes requiring embeddings for tenant={tenant} datasource={datasource}")

    batch_size = 10
    success = 0
    errors = 0
    for i in range(0, len(rows), batch_size):
        batch = rows[i:i+batch_size]
        for r in batch:
            try:
                row = dict(r)
                text = build_text(row)
                emb = embed_text(api_key, text)
                emb_str = vector_to_postgres(emb)
                cur.execute("UPDATE catalog_node SET embedding = %s::vector, updated_at = now() WHERE id = %s", (emb_str, row['id']))
                conn.commit()
                success += 1
                print(f"Saved embedding for node {row['id']}")
            except Exception as e:
                conn.rollback()
                errors += 1
                print(f"ERROR generating/saving embedding for node {r['id']}: {e}", file=sys.stderr)
        time.sleep(1)

    print(f"Complete: {success} successful, {errors} errors")
    cur.close()
    conn.close()


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('--tenant', required=True)
    p.add_argument('--datasource', required=True)
    p.add_argument('--db', default=os.getenv('DATABASE_URL'))
    p.add_argument('--api-key', default=os.getenv('GEMINI_API_KEY'))
    args = p.parse_args()

    if not args.db:
        print('DATABASE_URL not provided (env or --db).', file=sys.stderr)
        sys.exit(1)
    if not args.api_key:
        print('GEMINI_API_KEY not provided (env or --api-key).', file=sys.stderr)
        sys.exit(1)

    process(args.tenant, args.datasource, args.db, args.api_key)
