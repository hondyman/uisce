terraform {
  required_version = ">= 1.5.0"
  
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.80.0"
    }
  }
  
  backend "azurerm" {
    resource_group_name  = "semlayer-terraform-state"
    storage_account_name = "semlayertfstate"
    container_name       = "tfstate"
    key                  = "cosmosdb-postgresql.tfstate"
  }
}

provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

# Variables
variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
  default     = "production"
}

variable "location" {
  description = "Primary Azure region"
  type        = string
  default     = "eastus"
}

variable "location_secondary" {
  description = "Secondary Azure region for read replica"
  type        = string
  default     = "westeurope"
}

variable "location_tertiary" {
  description = "Tertiary Azure region for read replica"
  type        = string
  default     = "southeastasia"
}

variable "coordinator_vcores" {
  description = "vCores for coordinator node"
  type        = number
  default     = 4
}

variable "coordinator_storage_gb" {
  description = "Storage in GB for coordinator node"
  type        = number
  default     = 512
}

variable "worker_node_count" {
  description = "Number of worker nodes"
  type        = number
  default     = 3
}

variable "worker_vcores" {
  description = "vCores per worker node"
  type        = number
  default     = 4
}

variable "worker_storage_gb" {
  description = "Storage in GB per worker node"
  type        = number
  default     = 512
}

variable "postgresql_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "16"
}

variable "administrator_login" {
  description = "Administrator login name"
  type        = string
  default     = "semlayeradmin"
  sensitive   = true
}

variable "administrator_password" {
  description = "Administrator password"
  type        = string
  sensitive   = true
}

variable "aks_subnet_ids" {
  description = "List of AKS subnet IDs to allow access from"
  type        = list(string)
  default     = []
}

variable "allowed_ip_ranges" {
  description = "List of IP ranges to allow access from (CIDR notation)"
  type        = list(string)
  default     = []
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default = {
    Application = "Semlayer"
    Component   = "Database"
    ManagedBy   = "Terraform"
  }
}

# Resource Group
resource "azurerm_resource_group" "main" {
  name     = "rg-semlayer-cosmosdb-${var.environment}"
  location = var.location
  tags     = var.tags
}

# Primary Cosmos DB for PostgreSQL Cluster
resource "azurerm_cosmosdb_postgresql_cluster" "main" {
  name                            = "semlayer-citus-${var.environment}"
  resource_group_name             = azurerm_resource_group.main.name
  location                        = azurerm_resource_group.main.location
  
  # Coordinator configuration
  coordinator_server_edition      = "GeneralPurpose"
  coordinator_vcore_count         = var.coordinator_vcores
  coordinator_storage_quota_in_mb = var.coordinator_storage_gb * 1024
  coordinator_public_ip_access_enabled = false
  
  # Worker configuration
  node_count                      = var.worker_node_count
  node_server_edition             = "GeneralPurpose"
  node_vcores                     = var.worker_vcores
  node_storage_quota_in_mb        = var.worker_storage_gb * 1024
  node_public_ip_access_enabled   = false
  
  # PostgreSQL configuration
  citus_version                   = "12.1"
  sql_version                     = var.postgresql_version
  
  # Authentication
  administrator_login_password    = var.administrator_password
  
  # High availability
  ha_enabled                      = var.environment == "production" ? true : false
  
  # Maintenance window (Sunday 2-4 AM UTC)
  maintenance_window {
    day_of_week  = 0
    start_hour   = 2
    start_minute = 0
  }
  
  tags = merge(var.tags, {
    Environment = var.environment
    Region      = var.location
    Role        = "Primary"
  })
}

# Private Endpoint for Primary Cluster
resource "azurerm_private_endpoint" "main" {
  name                = "pe-semlayer-citus-${var.environment}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  subnet_id           = var.aks_subnet_ids[0]
  
  private_service_connection {
    name                           = "psc-semlayer-citus"
    private_connection_resource_id = azurerm_cosmosdb_postgresql_cluster.main.id
    is_manual_connection           = false
    subresource_names              = ["coordinator"]
  }
  
  private_dns_zone_group {
    name                 = "default"
    private_dns_zone_ids = [azurerm_private_dns_zone.postgres.id]
  }
  
  tags = var.tags
}

# Private DNS Zone
resource "azurerm_private_dns_zone" "postgres" {
  name                = "privatelink.postgres.cosmos.azure.com"
  resource_group_name = azurerm_resource_group.main.name
  tags                = var.tags
}

# DNS Zone VNet Links (add for each AKS VNet)
resource "azurerm_private_dns_zone_virtual_network_link" "postgres" {
  count                 = length(var.aks_subnet_ids) > 0 ? 1 : 0
  name                  = "link-semlayer-citus"
  resource_group_name   = azurerm_resource_group.main.name
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
  virtual_network_id    = split("/subnets/", var.aks_subnet_ids[0])[0]
  registration_enabled  = false
  tags                  = var.tags
}

# Firewall Rules for allowed IP ranges
resource "azurerm_cosmosdb_postgresql_firewall_rule" "allowed_ips" {
  count            = length(var.allowed_ip_ranges)
  name             = "allow-ip-${count.index}"
  cluster_id       = azurerm_cosmosdb_postgresql_cluster.main.id
  start_ip_address = cidrhost(var.allowed_ip_ranges[count.index], 0)
  end_ip_address   = cidrhost(var.allowed_ip_ranges[count.index], -1)
}

# Allow Azure services (for AKS without private endpoint)
resource "azurerm_cosmosdb_postgresql_firewall_rule" "azure_services" {
  name             = "allow-azure-services"
  cluster_id       = azurerm_cosmosdb_postgresql_cluster.main.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}

# Coordinator configuration
resource "azurerm_cosmosdb_postgresql_coordinator_configuration" "shared_preload_libraries" {
  name       = "shared_preload_libraries"
  cluster_id = azurerm_cosmosdb_postgresql_cluster.main.id
  value      = "citus,pg_stat_statements,pg_cron"
}

resource "azurerm_cosmosdb_postgresql_coordinator_configuration" "max_connections" {
  name       = "max_connections"
  cluster_id = azurerm_cosmosdb_postgresql_cluster.main.id
  value      = "200"
}

resource "azurerm_cosmosdb_postgresql_coordinator_configuration" "log_statement" {
  name       = "log_statement"
  cluster_id = azurerm_cosmosdb_postgresql_cluster.main.id
  value      = "ddl"
}

resource "azurerm_cosmosdb_postgresql_coordinator_configuration" "work_mem" {
  name       = "work_mem"
  cluster_id = azurerm_cosmosdb_postgresql_cluster.main.id
  value      = "64MB"
}

# Node (worker) configuration
resource "azurerm_cosmosdb_postgresql_node_configuration" "max_connections" {
  name       = "max_connections"
  cluster_id = azurerm_cosmosdb_postgresql_cluster.main.id
  value      = "100"
}

resource "azurerm_cosmosdb_postgresql_node_configuration" "work_mem" {
  name       = "work_mem"
  cluster_id = azurerm_cosmosdb_postgresql_cluster.main.id
  value      = "64MB"
}

# Database role for application
resource "azurerm_cosmosdb_postgresql_role" "app_role" {
  name       = "semlayer_app"
  cluster_id = azurerm_cosmosdb_postgresql_cluster.main.id
  password   = var.administrator_password
}

# Read-only role for reporting
resource "azurerm_cosmosdb_postgresql_role" "readonly_role" {
  name       = "semlayer_readonly"
  cluster_id = azurerm_cosmosdb_postgresql_cluster.main.id
  password   = var.administrator_password
}

# Key Vault for storing connection strings
resource "azurerm_key_vault" "main" {
  name                = "kv-semlayer-db-${var.environment}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"
  
  purge_protection_enabled   = true
  soft_delete_retention_days = 7
  
  tags = var.tags
}

data "azurerm_client_config" "current" {}

resource "azurerm_key_vault_access_policy" "terraform" {
  key_vault_id = azurerm_key_vault.main.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = data.azurerm_client_config.current.object_id
  
  secret_permissions = [
    "Get", "List", "Set", "Delete", "Purge", "Recover"
  ]
}

# Store connection strings in Key Vault
resource "azurerm_key_vault_secret" "db_connection_string" {
  name         = "db-connection-string"
  value        = "host=${azurerm_cosmosdb_postgresql_cluster.main.coordinator_server_id}.postgres.cosmos.azure.com port=5432 dbname=citus user=${var.administrator_login} password=${var.administrator_password} sslmode=require"
  key_vault_id = azurerm_key_vault.main.id
  
  depends_on = [azurerm_key_vault_access_policy.terraform]
}

resource "azurerm_key_vault_secret" "db_host" {
  name         = "db-host"
  value        = "${azurerm_cosmosdb_postgresql_cluster.main.coordinator_server_id}.postgres.cosmos.azure.com"
  key_vault_id = azurerm_key_vault.main.id
  
  depends_on = [azurerm_key_vault_access_policy.terraform]
}

resource "azurerm_key_vault_secret" "db_password" {
  name         = "db-password"
  value        = var.administrator_password
  key_vault_id = azurerm_key_vault.main.id
  
  depends_on = [azurerm_key_vault_access_policy.terraform]
}

# Diagnostic Settings
resource "azurerm_monitor_diagnostic_setting" "cosmosdb" {
  name                       = "diag-semlayer-citus"
  target_resource_id         = azurerm_cosmosdb_postgresql_cluster.main.id
  log_analytics_workspace_id = var.log_analytics_workspace_id
  
  enabled_log {
    category = "PostgreSQLLogs"
  }
  
  metric {
    category = "AllMetrics"
    enabled  = true
  }
}

variable "log_analytics_workspace_id" {
  description = "Log Analytics Workspace ID for diagnostics"
  type        = string
  default     = ""
}

# Outputs
output "cluster_id" {
  description = "The ID of the Cosmos DB for PostgreSQL cluster"
  value       = azurerm_cosmosdb_postgresql_cluster.main.id
}

output "coordinator_hostname" {
  description = "The hostname of the coordinator node"
  value       = "${azurerm_cosmosdb_postgresql_cluster.main.coordinator_server_id}.postgres.cosmos.azure.com"
}

output "private_endpoint_ip" {
  description = "The private IP address of the coordinator endpoint"
  value       = azurerm_private_endpoint.main.private_service_connection[0].private_ip_address
}

output "connection_string" {
  description = "Connection string for the database (sensitive)"
  value       = "host=${azurerm_cosmosdb_postgresql_cluster.main.coordinator_server_id}.postgres.cosmos.azure.com port=5432 dbname=citus user=${var.administrator_login} password=<password> sslmode=require"
  sensitive   = false
}

output "key_vault_name" {
  description = "Name of the Key Vault storing credentials"
  value       = azurerm_key_vault.main.name
}

output "key_vault_uri" {
  description = "URI of the Key Vault"
  value       = azurerm_key_vault.main.vault_uri
}

output "cluster_nodes" {
  description = "Number of worker nodes"
  value       = var.worker_node_count
}
