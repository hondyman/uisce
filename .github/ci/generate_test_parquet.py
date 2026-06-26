import pandas as pd
import pyarrow as pa
import pyarrow.parquet as pq
import sys

# Usage: python generate_test_parquet.py /tmp/test.parquet
out = sys.argv[1]
df = pd.DataFrame([{
    'incident_id': 'inc_test',
    'tenant_id': 't1',
    'region': 'us-east-1',
    'status': 'open',
    'severity': 'high',
    'created_at': pd.Timestamp.utcnow()
}])

table = pa.Table.from_pandas(df)
pq.write_table(table, out)
print('Wrote', out)
