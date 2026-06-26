#!/usr/bin/env python3
"""
Smoke test for semantic similarity search using Gemini embeddings + pgvector.
Usage:
  source scripts/load-local-env.sh
  python3 scripts/smoke_similarity_test.py --tenant-datasource <ID> --query "Customer name"

This script embeds the query using the same Gemini embedding model and queries Postgres for nearest nodes.
"""
import os
import argparse
import requests
import json
import psycopg2
import psycopg2.extras
import sys

EMBEDDING_MODEL = "text-embedding-004"
GEMINI_EMBED_URL = "https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent"


def embed_text(api_key: str, text: str):
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


def vector_to_postgres(vec):
    return '[' + ','.join(f"{v:.6f}" for v in vec) + ']'


def run_smoke(db_url, api_key, tenant_datasource_id, query_text, topk=5):
    print(f"Embedding query: '{query_text}' (tenant_datasource={tenant_datasource_id})")
    emb = embed_text(api_key, query_text)
    vec_str = vector_to_postgres(emb)

    conn = psycopg2.connect(db_url)
    cur = conn.cursor(cursor_factory=psycopg2.extras.DictCursor)

    sql = """
    SELECT id, node_name, substring(embedding::text for 200) as embedding_preview, (embedding <=> %s::vector) as distance
    FROM catalog_node
    WHERE tenant_datasource_id = %s AND embedding IS NOT NULL
    ORDER BY embedding <=> %s::vector
    LIMIT %s
    """

    cur.execute(sql, (vec_str, tenant_datasource_id, vec_str, topk))
    rows = cur.fetchall()
    if not rows:
        print("No results returned.")
        return

    print(f"Top {len(rows)} results:")
    for r in rows:
        print(f"- id={r['id']} distance={r['distance']:.6f} name={r['node_name']} preview={r['embedding_preview']}")

    cur.close()
    conn.close()


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('--tenant-datasource', required=True)
    p.add_argument('--query', required=True)
    p.add_argument('--db', default=os.getenv('DATABASE_URL'))
    p.add_argument('--api-key', default=os.getenv('GEMINI_API_KEY'))
    args = p.parse_args()

    if not args.db:
        print('DATABASE_URL not provided (env or --db).', file=sys.stderr)
        sys.exit(1)
    if not args.api_key:
        print('GEMINI_API_KEY not provided (env or --api-key).', file=sys.stderr)
        sys.exit(1)

    run_smoke(args.db, args.api_key, args.tenant_datasource, args.query)
