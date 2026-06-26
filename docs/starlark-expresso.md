# Starlark “Expresso-ish” Rules

This repo supports an authoring style where a Starlark script simply sets:

- `ok = <bool>` (required)
- `message = "..."` (optional)

The runtime injects a dynamic `ctx` object plus helper builtins (no imports required).

## Context model

- `ctx.page`: scalar values (top-level non-object fields)
- `ctx.<object>`: any top-level object in the input record

Example input:

```json
{
  "account": {"account_type": "ADVISORY", "aum": 150},
  "country": "US"
}
```

Then:

- `ctx.account.account_type` exists (via `get/field/F`)
- `ctx.page.country` exists

## Common pattern

```starlark
ok = eq(field("account", "account_type"), "ADVISORY") and gt(num_field("account", "aum"), 100)
message = "Account did not meet threshold"
```

## Helper builtins (high level)

### Accessors

- `field(bo, key, default=None)`
- `num_field(bo, key, default=None)`
- `bool_field(bo, key, default=None)`
- `F("bo.key")` or `F("bo", "key")`

### Generic ctx-aware lookups

- `get(object, field, default=None)`
- `exists(object, field)`
- `get_path(object, path, default=None)` (dot path; supports list indexes like `items.0.name`)
- `exists_path(object, path)`

### Conversions

- `to_string(x)`
- `to_number(x)`
- `to_bool(x)`
- `coalesce(a, b, c, ...)`

### Comparisons

- `eq/ne/gt/ge/lt/le`

### Strings

- `contains/startswith/endswith/is_blank`
- `lower/upper/trim`
- `regex_match(pattern, s)`

### Dates

- `today()` -> `YYYY-MM-DD`
- `date_before(a, b)` / `date_after(a, b)` using `YYYY-MM-DD`

## VS Code authoring tips

- Install the recommended Starlark extension (see the workspace’s recommended extensions).
- Use the provided snippets:
  - type `semlayer-ok` then tab
  - type `semlayer-field` / `semlayer-path` then tab

## Inline testcases (fast feedback)

You can embed tiny testcases directly in a `.star` file as comment lines containing JSON objects.
Each testcase is one line starting with `# { ... }`.

Example:

```starlark
ok = eq(field("account", "account_type"), "ADVISORY")

# {"name":"advisory","record":{"account":{"account_type":"ADVISORY"}},"expect":true}
# {"name":"brokerage","record":{"account":{"account_type":"BROKERAGE"}},"expect":false}
```

Run them locally:

`cd backend && go run ./cmd/starlarktest -file /absolute/path/to/rule.star`
