package semlayer.governance.semantic

import future.keywords.in

default allow = false

# Allow if no denial reasons
allow {
    count(deny) == 0
}

# Deny if description is too short (if present)
deny[msg] {
    is_string(input.description)
    count(input.description) > 0
    count(input.description) < 5
    msg := "Description must be at least 5 characters long if provided"
}

# Deny if recursion depth is too high (simulated check)
# In reality, the helper logic would be passed in input, here we check 'complexity_score' if available
deny[msg] {
    input.complexity_score > 5
    msg := sprintf("Complexity score %d exceeds limit of 5", [input.complexity_score])
}

# Deny if using restricted tables (unless authorized)
# referencing physical columns in 'restricted_column_list'
deny[msg] {
    some col in input.referenced_columns
    col in data.restricted_columns.list
    not input.user_has_pii_access
    msg := sprintf("Access to restricted column '%s' denied", [col])
}

# Deny dangerous characters in NodeName
deny[msg] {
    not regex.match("^[a-zA-Z0-9_]+$", input.node_name)
    msg := "Node name must contain only alphanumeric characters and underscores"
}
