package portal_authz

default allow = false

# Allow titan_admin to do anything
allow {
    input.user.role == "titan_admin"
}

# Allow client_admin to manage their own tenant's resources
allow {
    input.user.role == "client_admin"
    tenant_match
    method_allowed_admin
}

# Allow client_viewer to view their own tenant's resources
allow {
    input.user.role == "client_viewer"
    tenant_match
    input.method == "GET"
}

# Helper: Ensure the user's tenant matches the requested resource's tenant
tenant_match {
    # If resource tenant is provided in the input
    input.resource.tenant_id == input.user.tenant_id
}

tenant_match {
    # If tenant is being set in the body (e.g., creation)
    input.body.tenant_id == input.user.tenant_id
}

# Helper: Allowed methods for client_admin
method_allowed_admin {
    allowed_methods := {"GET", "POST", "PUT", "DELETE"}
    allowed_methods[input.method]
}
