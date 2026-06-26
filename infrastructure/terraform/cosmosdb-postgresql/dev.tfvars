# Development configuration for Cosmos DB for PostgreSQL
environment = "dev"
location    = "eastus"

# Smaller coordinator for dev
coordinator_vcores     = 2
coordinator_storage_gb = 128

# Minimal worker nodes for dev
worker_node_count = 2
worker_vcores     = 2
worker_storage_gb = 128

# PostgreSQL version
postgresql_version = "16"

# Admin credentials
administrator_login = "semlayeradmin"
# administrator_password = "<set-in-env>"

# Network access
aks_subnet_ids = []

# Allow dev IPs
allowed_ip_ranges = [
  # Add your dev machine IP/range
]

tags = {
  Application = "Semlayer"
  Component   = "Database"
  Environment = "Development"
  ManagedBy   = "Terraform"
}
