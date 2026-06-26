import re
import os

# Try a list of likely DDL files so the script is resilient when files move.
ddl_candidates = [
    "backend/totalddl.sql",
    "backend/migration_to_ddl_schema.sql",
    "backend/init-db.sql",
    "migration_to_ddl_schema.sql",
    "init-db.sql",
    "totalddl.sql",
]

out_file = "api-gateway/graph/schema.graphqls"

def find_ddl_file():
    for p in ddl_candidates:
        if os.path.isfile(p):
            return p
    return None

ddl_file = find_ddl_file()
if ddl_file:
    print(f"[generate_graphql] Using DDL file: {ddl_file}")
else:
    print('[generate_graphql] No DDL file found among candidates; continuing with empty DDL (will generate empty schema).')

# Match CREATE TABLE with body between parentheses (non-greedy)
table_re = re.compile(r"CREATE\s+TABLE\s+public\.(?P<name>\"?[A-Za-z_][A-Za-z0-9_]*\"?)\s*\((?P<body>.*?)\)\s*;", re.IGNORECASE | re.DOTALL)

CONSTRAINT_TOKENS = {
    'DEFAULT', 'NOT', 'NULL', 'CONSTRAINT', 'PRIMARY', 'UNIQUE', 'REFERENCES', 'CHECK', 'KEY', 'COLLATE',
    'GENERATED', 'BY', 'AS', 'IDENTITY', 'ON', 'UPDATE', 'SET', 'CASCADE', 'DEFERRABLE', 'INITIALLY',
    'DEFERRED', 'INDEX', 'COMMENT', 'TRIGGER', 'BEFORE', 'AFTER', 'EACH', 'ROW', 'EXECUTE', 'FUNCTION',
    'PARTITION', 'VALUES', 'FROM', 'TO', 'WHERE', 'WITH', 'WITHOUT', 'STORAGE', 'OWNER', 'AUTHORIZATION',
}

SKIP_LINE_PREFIXES = (
    'CONSTRAINT', 'PRIMARY KEY', 'UNIQUE', 'FOREIGN KEY', 'CREATE INDEX', 'COMMENT', 'CHECK', 'TRIGGER'
)

def to_camel(s: str) -> str:
    s = s.replace('"', '')
    return ''.join([p.capitalize() for p in s.split('_') if p])

def sanitize_name(s: str) -> str:
    """Return a sanitized identifier (keeps underscores) for SQL names."""
    s = s.replace('"', '')
    s = re.sub(r"[^A-Za-z0-9_]", "_", s)
    return s


def to_lower_camel(s: str) -> str:
    """Convert snake_case (or other sanitized name) to lowerCamelCase for GraphQL fields."""
    s = s.replace('"', '')
    s = re.sub(r"[^A-Za-z0-9_]", "_", s)
    parts = [p for p in s.split('_') if p]
    if not parts:
        return s
    first = parts[0].lower()
    rest = ''.join(p.capitalize() for p in parts[1:])
    name = first + rest
    # Ensure it doesn't start with a digit
    if not re.match(r'^[A-Za-z_]', name):
        name = '_' + name
    # Collapse leading double-underscores
    name = re.sub(r'^__+', '_', name)
    return name

def split_columns(body: str):
    cols = []
    depth = 0
    current = []
    for ch in body:
        if ch == '(':
            depth += 1
        elif ch == ')':
            depth -= 1
        if ch == ',' and depth == 0:
            seg = ''.join(current).strip()
            if seg:
                cols.append(seg)
            current = []
        else:
            current.append(ch)
    tail = ''.join(current).strip()
    if tail:
        cols.append(tail)
    return cols

def parse_column(segment: str):
    line = segment.strip()
    up = line.upper()
    first_word = up.split(None, 1)[0] if ' ' in up else up

    if any(up.startswith(p) for p in SKIP_LINE_PREFIXES) or first_word in CONSTRAINT_TOKENS:
        return None

    # quoted or unquoted column name
    name = None
    rest = ''
    if line.startswith('"'):
        endq = line.find('"', 1)
        if endq == -1:
            return None # Unmatched quote
        name = line[1:endq]
        rest = line[endq+1:].strip()
    else:
        parts = re.split(r'\s+', line, maxsplit=1)
        if len(parts) < 2:
            return None
        name, rest = parts[0], parts[1]
    # collect type tokens until a constraint token
    type_tokens = []
    for tok in re.split(r"\s+", rest):
        if tok.upper() in CONSTRAINT_TOKENS:
            break
        type_tokens.append(tok)
    if not type_tokens:
        return None
    sql_type = ' '.join(type_tokens)
    not_null = 'NOT NULL' in up
    return name, sql_type, not_null

def sql_to_graphql_type(sql_type: str) -> str:
    t = sql_type.lower()
    if 'uuid' in t:
        return 'ID'
    if any(x in t for x in ['varchar', 'character varying', 'char', 'text', 'citext']):
        return 'String'
    if any(x in t for x in ['bigint', 'int8']):
        return 'Int'
    if any(x in t for x in ['int', 'integer', 'int4', 'smallint', 'serial', 'bigserial', 'smallserial']):
        return 'Int'
    if any(x in t for x in ['numeric', 'decimal', 'real', 'double', 'float']):
        return 'Float'
    if 'bool' in t:
        return 'Boolean'
    if any(x in t for x in ['timestamp', 'timestamptz', 'date', 'time']):
        return 'String'
    if any(x in t for x in ['json', 'jsonb', 'bytea']):
        return 'String'
    return 'String'

if ddl_file:
    with open(ddl_file, 'r') as f:
        ddl = f.read()
else:
    ddl = ''

tables = []  # list of dicts: {name, typeName, inputName, columns: [(field, gqlType, required)]}

for m in table_re.finditer(ddl):
    raw_name = m.group('name')
    # table_identifier: sanitized snake-like name used in query/mutation field names
    table_identifier = sanitize_name(raw_name).lower()
    type_name = to_camel(table_identifier)
    input_name = type_name + 'Input'
    body = m.group('body')
    cols = []
    for seg in split_columns(body):
        parsed = parse_column(seg)
        if not parsed:
            continue
        col_name, sql_type, not_null = parsed
        gql_field = to_lower_camel(col_name)
        # Skip invalid GraphQL names
        if gql_field.startswith('__'):
            print(f'[generate_graphql] Skipping invalid field name "{col_name}" in table "{table_identifier}"')
            continue
        if not re.match(r'^[A-Za-z_][A-Za-z0-9_]*$', gql_field):
            continue
        gql_type = sql_to_graphql_type(sql_type)
        cols.append((gql_field, gql_type, not_null)) # Use original col_name for input mapping
    if cols:
        tables.append({
            'table': table_identifier,
            'type': type_name,
            'input': input_name,
            'columns': cols,
        })

query_fields = []
mutation_fields = []

lines = []
lines.append('# AUTOGENERATED GraphQL schema from DDL\n')

# Emit types and inputs
for t in tables:
    lines.append(f'type {t["type"]} {{\n')
    for field_name, gql_type, required in t['columns']:
        bang = '!' if required else ''
        lines.append(f'  {field_name}: {gql_type}{bang}\n')
    lines.append('}\n\n')

    lines.append(f'input {t["input"]} {{\n')
    for field_name, gql_type, _ in t['columns']:
        lines.append(f'  {field_name}: {gql_type}\n')
    lines.append('}\n\n')

    # Collect Query/Mutation fields
    query_fields.append(f'  all_{t["table"]}: [{t["type"]}!]!')
    query_fields.append(f'  {t["table"]}_by_id(id: ID!): {t["type"]}')

    mutation_fields.append(f'  create_{t["table"]}(input: {t["input"]}!): {t["type"]}!')
    mutation_fields.append(f'  update_{t["table"]}(id: ID!, input: {t["input"]}!): {t["type"]}!')
    mutation_fields.append(f'  delete_{t["table"]}(id: ID!): Boolean!')

# Root types
if query_fields:
    lines.append('type Query {\n')
    lines.append('\n'.join(query_fields) + '\n')
    lines.append('}\n\n')
if mutation_fields:
    lines.append('type Mutation {\n')
    lines.append('\n'.join(mutation_fields) + '\n')
    lines.append('}\n')

sdl = ''.join(lines)

def validate_sdl(s: str) -> str:
    # Balanced braces
    if s.count('{') != s.count('}'):
        return f'Unbalanced braces: {{={s.count("{")}, }}={s.count("}")}'
    # No empty root types
    if re.search(r'type\s+Query\s*\{\s*\}', s):
        return 'Empty type Query generated'
    if re.search(r'type\s+Mutation\s*\{\s*\}', s):
        return 'Empty type Mutation generated'
    # No standalone closing braces
    if re.search(r'^\s*\}\s*\}\s*$', s, re.MULTILINE):
        # might be valid but warn; treat as error to be safe
        return 'Suspicious consecutive closing braces detected'
    return ''

err = validate_sdl(sdl)
if err:
    print(f'[generate_graphql] Validation failed: {err}')
    # Still write to help inspection but exit non-zero for CI/gqlgen safety
    try:
        os.makedirs(os.path.dirname(out_file), exist_ok=True)
        with open(out_file, 'w') as out:
            out.write(sdl)
    except NotADirectoryError:
        # Parent path component is a file (e.g., an `api-gateway` binary). Fall back to repo-level `graph/`.
        fallback = 'graph/schema.graphqls'
        print(f"[generate_graphql] Warning: cannot create directory for {out_file}; falling back to {fallback}")
        os.makedirs(os.path.dirname(fallback), exist_ok=True)
        with open(fallback, 'w') as out:
            out.write(sdl)
    raise SystemExit(1)
else:
    try:
        os.makedirs(os.path.dirname(out_file), exist_ok=True)
        with open(out_file, 'w') as out:
            out.write(sdl)
    except NotADirectoryError:
        fallback = 'graph/schema.graphqls'
        print(f"[generate_graphql] Warning: cannot create directory for {out_file}; falling back to {fallback}")
        os.makedirs(os.path.dirname(fallback), exist_ok=True)
        with open(fallback, 'w') as out:
            out.write(sdl)
