# Production configuration for Cosmos DB for PostgreSQL
environment = "production"
location    = "eastus"

# Coordinator node (handles query routing)
coordinator_vcores     = 8
coordinator_storage_gb = 1024

# Worker nodes (hold the sharded data)
worker_node_count = 5
worker_vcores     = 8
worker_storage_gb = 1024

# PostgreSQL version
postgresql_version = "16"

# Admin credentials (use Azure Key Vault references in CI/CD)
administrator_login = "semlayeradmin"
# administrator_password = "<from-key-vault>"

# Network access (populate with actual subnet IDs)
aks_subnet_ids = [
  # "/subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.Network/virtualNetworks/<vnet>/subnets/aks-subnet"
]

# Allowed IP ranges for dev/ops access
allowed_ip_ranges = [
  # "10.0.0.0/8",
  # "192.168.1.0/24"
]

# Log Analytics for diagnostics
# log_analytics_workspace_id = "/subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.OperationalInsights/workspaces/<workspace>"

tags = {
  Application  = "Semlayer"
  Component    = "Database"
  Environment  = "Production"
  ManagedBy    = "Terraform"
  CostCenter   = "Platform"
  DataClassification = "Confidential"
}
