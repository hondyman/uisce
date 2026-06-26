
import os
import re

mig_dir = '.'  # Run from backend/migrations

slug_map = {
    'aum-total-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'nav-growth-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d480',
    'inflows-net-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d481',
    'volatility-30d-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    'sharpe-ratio-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d483',
    'transaction-volume-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d484',
    'processing-time-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d485',
    'compliance-filings-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d486',
    'regulatory-fines-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d495',
    'client-satisfaction-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d493',
    'alpha-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d489',
    'beta-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d490',
    'max-drawdown-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d491',
    'tracking-error-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d492',
    'fund-expense-ratio-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d494',
    'audit-findings-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d496',
    'market-share-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d497',
    'peer-performance-rank-001': 'f47ac10b-58cc-4372-a567-0e02b2c3d498',
    'pop-cockpit-main': 'd142512a-3505-4c6e-8121-5079540b7274'
}

print(f"Checking directory: {os.getcwd()}")
files = [f for f in os.listdir(mig_dir) if f.endswith('.sql')]
print(f"Found {len(files)} SQL files")

for f in files:
    path = os.path.join(mig_dir, f)
    with open(path, 'r') as file:
        content = file.read()
    
    original_content = content
    
    # 1. Replace subqueries
    for slug, uuid in slug_map.items():
        pattern = r"\(SELECT id FROM public\.pop_metrics WHERE name = '" + re.escape(slug) + r"'\)"
        content = re.sub(pattern, f"'{uuid}'", content)
    
    # 2. Replace slugs
    for slug, uuid in slug_map.items():
        if slug in content:
            print(f"Replacing {slug} in {f}")
            content = content.replace(slug, uuid)
            
    # 3. Add ::uuid[] casting
    # Find ARRAY['uuid'...] pattern
    # We assume UUID structure or the specific UUIDs we just replaced
    # Look for: ARRAY['f47ac...'] or ['d1425...']
    # And check if followed by ::uuid[]
    
    def repl_array(m):
        full_match = m.group(0)
        if '::uuid[]' in full_match:
            return full_match
        # If it doesn't have cast, append it
        return full_match + '::uuid[]'

    # Pattern: ARRAY[ ... ] where ... contains 'f47ac' or 'd1425'
    # Use generic array matcher, then check content
    pattern = r"ARRAY\[.*?\](::uuid\[\])?"
    
    def callback(m):
        s = m.group(0)
        # Check if already cast
        if s.endswith('::uuid[]'):
            return s
        # Check if it contains our UUIDs
        if 'f47ac10b' in s or 'd142512a' in s:
            print(f"Adding cast to ARRAY in {f}")
            return s + '::uuid[]'
        return s

    content = re.sub(pattern, callback, content, flags=re.DOTALL)
    
    if content != original_content:
        print(f"Writing changes to {f}")
        with open(path, 'w') as file:
            file.write(content)
