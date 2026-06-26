# Semlayer Infrastructure - Azure Configuration
# Multi-region enterprise deployment on Azure

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.85"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.47"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.25"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  # Remote state in Azure Storage
  backend "azurerm" {
    resource_group_name  = "semlayer-tfstate-rg"
    storage_account_name = "semlayertfstate"
    container_name       = "tfstate"
    key                  = "production-us/terraform.tfstate"
  }
}

# =============================================================================
# Provider Configuration
# =============================================================================

provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy    = false
      recover_soft_deleted_key_vaults = true
    }
    resource_group {
      prevent_deletion_if_contains_resources = true
    }
  }
}

provider "azuread" {}

# =============================================================================
# Variables
# =============================================================================

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"

  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "Environment must be staging or production."
  }
}

variable "location" {
  description = "Primary Azure region"
  type        = string
  default     = "eastus"
}

variable "location_secondary" {
  description = "Secondary Azure region for DR"
  type        = string
  default     = "westus2"
}

variable "cluster_name" {
  description = "AKS cluster name"
  type        = string
  default     = "semlayer-prod"
}

variable "kubernetes_version" {
  description = "Kubernetes version for AKS"
  type        = string
  default     = "1.29"
}

variable "vnet_address_space" {
  description = "VNet address space"
  type        = list(string)
  default     = ["10.0.0.0/16"]
}

variable "postgresql_sku" {
  description = "PostgreSQL Flexible Server SKU"
  type        = string
  default     = "GP_Standard_D4s_v3"
}

variable "redis_sku" {
  description = "Redis Cache SKU"
  type        = string
  default     = "Premium"
}

variable "redis_capacity" {
  description = "Redis Cache capacity"
  type        = number
  default     = 1
}

# =============================================================================
# Data Sources
# =============================================================================

data "azurerm_client_config" "current" {}

data "azuread_client_config" "current" {}

# =============================================================================
# Resource Group
# =============================================================================

resource "azurerm_resource_group" "main" {
  name     = "${var.cluster_name}-rg"
  location = var.location

  tags = {
    Environment = var.environment
    Project     = "semlayer"
    ManagedBy   = "terraform"
  }
}

# =============================================================================
# Virtual Network
# =============================================================================

resource "azurerm_virtual_network" "main" {
  name                = "${var.cluster_name}-vnet"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  address_space       = var.vnet_address_space

  tags = {
    Environment = var.environment
  }
}

# AKS Subnet
resource "azurerm_subnet" "aks" {
  name                 = "aks-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.0.0/20"]

  service_endpoints = [
    "Microsoft.Storage",
    "Microsoft.Sql",
    "Microsoft.KeyVault",
    "Microsoft.ServiceBus"
  ]
}

# PostgreSQL Subnet
resource "azurerm_subnet" "postgresql" {
  name                 = "postgresql-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.16.0/24"]

  delegation {
    name = "postgresql-delegation"
    service_delegation {
      name = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = [
        "Microsoft.Network/virtualNetworks/subnets/join/action"
      ]
    }
  }

  service_endpoints = ["Microsoft.Storage"]
}

# Redis Subnet
resource "azurerm_subnet" "redis" {
  name                 = "redis-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.17.0/24"]
}

# Private Endpoints Subnet
resource "azurerm_subnet" "private_endpoints" {
  name                 = "private-endpoints-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.18.0/24"]

  private_endpoint_network_policies_enabled = false
}

# Application Gateway Subnet
resource "azurerm_subnet" "appgw" {
  name                 = "appgw-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.19.0/24"]
}

# =============================================================================
# Azure Kubernetes Service (AKS)
# =============================================================================

resource "azurerm_user_assigned_identity" "aks" {
  name                = "${var.cluster_name}-identity"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_kubernetes_cluster" "main" {
  name                = var.cluster_name
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  dns_prefix          = var.cluster_name
  kubernetes_version  = var.kubernetes_version

  # Use User Assigned Identity
  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.aks.id]
  }

  # Default node pool (system)
  default_node_pool {
    name                         = "system"
    vm_size                      = "Standard_D4s_v3"
    node_count                   = 3
    min_count                    = 3
    max_count                    = 5
    enable_auto_scaling          = true
    vnet_subnet_id               = azurerm_subnet.aks.id
    only_critical_addons_enabled = true
    os_disk_size_gb              = 100
    os_disk_type                 = "Managed"
    
    node_labels = {
      "workload" = "system"
    }

    upgrade_settings {
      max_surge = "33%"
    }

    zones = ["1", "2", "3"]
  }

  # Network configuration
  network_profile {
    network_plugin      = "azure"
    network_plugin_mode = "overlay"
    network_policy      = "calico"
    load_balancer_sku   = "standard"
    outbound_type       = "loadBalancer"
    service_cidr        = "10.1.0.0/16"
    dns_service_ip      = "10.1.0.10"
  }

  # Azure AD integration
  azure_active_directory_role_based_access_control {
    managed                = true
    azure_rbac_enabled     = true
    admin_group_object_ids = [azuread_group.aks_admins.object_id]
  }

  # Enable features
  oidc_issuer_enabled       = true
  workload_identity_enabled = true
  
  # Key Vault secrets provider
  key_vault_secrets_provider {
    secret_rotation_enabled  = true
    secret_rotation_interval = "2m"
  }

  # Azure Monitor
  oms_agent {
    log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  }

  # Microsoft Defender
  microsoft_defender {
    log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  }

  # Maintenance window
  maintenance_window {
    allowed {
      day   = "Sunday"
      hours = [2, 3, 4]
    }
  }

  auto_scaler_profile {
    balance_similar_node_groups      = true
    expander                         = "least-waste"
    max_graceful_termination_sec     = 600
    max_node_provisioning_time       = "15m"
    max_unready_nodes                = 3
    max_unready_percentage           = 45
    new_pod_scale_up_delay           = "10s"
    scale_down_delay_after_add       = "10m"
    scale_down_delay_after_delete    = "10s"
    scale_down_delay_after_failure   = "3m"
    scan_interval                    = "10s"
    scale_down_unneeded              = "10m"
    scale_down_unready               = "20m"
    scale_down_utilization_threshold = 0.5
    empty_bulk_delete_max            = 10
    skip_nodes_with_local_storage    = false
    skip_nodes_with_system_pods      = true
  }

  tags = {
    Environment = var.environment
  }
}

# API Node Pool
resource "azurerm_kubernetes_cluster_node_pool" "api" {
  name                  = "api"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.main.id
  vm_size               = "Standard_D4s_v3"
  node_count            = 3
  min_count             = 3
  max_count             = 50
  enable_auto_scaling   = true
  vnet_subnet_id        = azurerm_subnet.aks.id
  os_disk_size_gb       = 100
  os_disk_type          = "Managed"
  
  node_labels = {
    "workload" = "api"
  }

  upgrade_settings {
    max_surge = "25%"
  }

  zones = ["1", "2", "3"]

  tags = {
    Environment = var.environment
  }
}

# Cube Workers Node Pool (Spot instances for cost optimization)
resource "azurerm_kubernetes_cluster_node_pool" "cube" {
  name                  = "cube"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.main.id
  vm_size               = "Standard_E8s_v3"  # Memory-optimized
  node_count            = 5
  min_count             = 3
  max_count             = 100
  enable_auto_scaling   = true
  vnet_subnet_id        = azurerm_subnet.aks.id
  os_disk_size_gb       = 128
  os_disk_type          = "Managed"
  priority              = "Spot"
  eviction_policy       = "Delete"
  spot_max_price        = -1  # Pay up to on-demand price
  
  node_labels = {
    "workload"                                = "cube"
    "kubernetes.azure.com/scalesetpriority"  = "spot"
  }

  node_taints = [
    "kubernetes.azure.com/scalesetpriority=spot:NoSchedule"
  ]

  upgrade_settings {
    max_surge = "50%"
  }

  zones = ["1", "2", "3"]

  tags = {
    Environment = var.environment
  }
}

# Data Services Node Pool
resource "azurerm_kubernetes_cluster_node_pool" "data" {
  name                  = "data"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.main.id
  vm_size               = "Standard_E8s_v3"  # Memory-optimized
  node_count            = 3
  min_count             = 3
  max_count             = 20
  enable_auto_scaling   = true
  vnet_subnet_id        = azurerm_subnet.aks.id
  os_disk_size_gb       = 256
  os_disk_type          = "Managed"
  
  node_labels = {
    "workload" = "data"
  }

  upgrade_settings {
    max_surge = "33%"
  }

  zones = ["1", "2", "3"]

  tags = {
    Environment = var.environment
  }
}

# AI/GPU Node Pool (optional)
resource "azurerm_kubernetes_cluster_node_pool" "gpu" {
  count                 = var.environment == "production" ? 1 : 0
  name                  = "gpu"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.main.id
  vm_size               = "Standard_NC6s_v3"  # NVIDIA V100
  node_count            = 0
  min_count             = 0
  max_count             = 10
  enable_auto_scaling   = true
  vnet_subnet_id        = azurerm_subnet.aks.id
  os_disk_size_gb       = 128
  os_disk_type          = "Managed"
  
  node_labels = {
    "workload"            = "ai"
    "nvidia.com/gpu.present" = "true"
  }

  node_taints = [
    "nvidia.com/gpu=present:NoSchedule"
  ]

  upgrade_settings {
    max_surge = "100%"
  }

  tags = {
    Environment = var.environment
  }
}

# =============================================================================
# Azure AD Groups for RBAC
# =============================================================================

resource "azuread_group" "aks_admins" {
  display_name     = "${var.cluster_name}-admins"
  security_enabled = true
  description      = "AKS cluster administrators"
}

resource "azuread_group" "aks_developers" {
  display_name     = "${var.cluster_name}-developers"
  security_enabled = true
  description      = "AKS cluster developers (read-only + deploy)"
}

# =============================================================================
# Azure Container Registry
# =============================================================================

resource "azurerm_container_registry" "main" {
  name                = replace("${var.cluster_name}acr", "-", "")
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Premium"
  admin_enabled       = false

  # Geo-replication for multi-region
  georeplications {
    location                = var.location_secondary
    zone_redundancy_enabled = true
  }

  # Network rules
  network_rule_set {
    default_action = "Deny"

    virtual_network {
      action    = "Allow"
      subnet_id = azurerm_subnet.aks.id
    }
  }

  # Enable content trust
  trust_policy {
    enabled = true
  }

  # Retention policy
  retention_policy {
    days    = 30
    enabled = true
  }

  tags = {
    Environment = var.environment
  }
}

# Grant AKS access to ACR
resource "azurerm_role_assignment" "aks_acr" {
  scope                = azurerm_container_registry.main.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_kubernetes_cluster.main.kubelet_identity[0].object_id
}

# =============================================================================
# Azure Database for PostgreSQL Flexible Server
# =============================================================================

resource "azurerm_private_dns_zone" "postgresql" {
  name                = "privatelink.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_private_dns_zone_virtual_network_link" "postgresql" {
  name                  = "postgresql-vnet-link"
  private_dns_zone_name = azurerm_private_dns_zone.postgresql.name
  resource_group_name   = azurerm_resource_group.main.name
  virtual_network_id    = azurerm_virtual_network.main.id
}

resource "random_password" "postgresql" {
  length  = 32
  special = true
}

resource "azurerm_postgresql_flexible_server" "main" {
  name                          = "${var.cluster_name}-postgres"
  resource_group_name           = azurerm_resource_group.main.name
  location                      = azurerm_resource_group.main.location
  version                       = "15"
  delegated_subnet_id           = azurerm_subnet.postgresql.id
  private_dns_zone_id           = azurerm_private_dns_zone.postgresql.id
  administrator_login           = "semlayer_admin"
  administrator_password        = random_password.postgresql.result
  zone                          = "1"
  storage_mb                    = 131072  # 128 GB
  sku_name                      = var.postgresql_sku
  backup_retention_days         = var.environment == "production" ? 35 : 7
  geo_redundant_backup_enabled  = var.environment == "production"
  auto_grow_enabled             = true

  high_availability {
    mode                      = var.environment == "production" ? "ZoneRedundant" : "SameZone"
    standby_availability_zone = var.environment == "production" ? "2" : null
  }

  maintenance_window {
    day_of_week  = 0  # Sunday
    start_hour   = 2
    start_minute = 0
  }

  tags = {
    Environment = var.environment
  }

  depends_on = [azurerm_private_dns_zone_virtual_network_link.postgresql]
}

# PostgreSQL Database
resource "azurerm_postgresql_flexible_server_database" "semlayer" {
  name      = "semlayer"
  server_id = azurerm_postgresql_flexible_server.main.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

# PostgreSQL Configuration
resource "azurerm_postgresql_flexible_server_configuration" "extensions" {
  name      = "azure.extensions"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "PG_STAT_STATEMENTS,PGCRYPTO,UUID-OSSP"
}

resource "azurerm_postgresql_flexible_server_configuration" "log_min_duration" {
  name      = "log_min_duration_statement"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "1000"  # Log queries > 1 second
}

# =============================================================================
# Azure Cache for Redis
# =============================================================================

resource "azurerm_redis_cache" "main" {
  name                          = "${var.cluster_name}-redis"
  resource_group_name           = azurerm_resource_group.main.name
  location                      = azurerm_resource_group.main.location
  capacity                      = var.redis_capacity
  family                        = var.redis_sku == "Premium" ? "P" : "C"
  sku_name                      = var.redis_sku
  enable_non_ssl_port           = false
  minimum_tls_version           = "1.2"
  public_network_access_enabled = false

  redis_configuration {
    maxmemory_policy            = "volatile-lru"
    maxmemory_reserved          = 256
    maxfragmentationmemory_reserved = 256
    enable_authentication       = true
  }

  # Zone redundancy (Premium only)
  zones = var.redis_sku == "Premium" ? ["1", "2", "3"] : null

  # Geo-replication (Premium only in production)
  dynamic "patch_schedule" {
    for_each = var.redis_sku == "Premium" ? [1] : []
    content {
      day_of_week    = "Sunday"
      start_hour_utc = 2
    }
  }

  tags = {
    Environment = var.environment
  }
}

# Private endpoint for Redis
resource "azurerm_private_endpoint" "redis" {
  name                = "${var.cluster_name}-redis-pe"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  subnet_id           = azurerm_subnet.private_endpoints.id

  private_service_connection {
    name                           = "redis-connection"
    private_connection_resource_id = azurerm_redis_cache.main.id
    subresource_names              = ["redisCache"]
    is_manual_connection           = false
  }

  private_dns_zone_group {
    name                 = "redis-dns-group"
    private_dns_zone_ids = [azurerm_private_dns_zone.redis.id]
  }
}

resource "azurerm_private_dns_zone" "redis" {
  name                = "privatelink.redis.cache.windows.net"
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_private_dns_zone_virtual_network_link" "redis" {
  name                  = "redis-vnet-link"
  private_dns_zone_name = azurerm_private_dns_zone.redis.name
  resource_group_name   = azurerm_resource_group.main.name
  virtual_network_id    = azurerm_virtual_network.main.id
}

# =============================================================================
# Azure Service Bus (for message queuing)
# =============================================================================

resource "azurerm_servicebus_namespace" "main" {
  name                          = "${var.cluster_name}-servicebus"
  resource_group_name           = azurerm_resource_group.main.name
  location                      = azurerm_resource_group.main.location
  sku                           = var.environment == "production" ? "Premium" : "Standard"
  capacity                      = var.environment == "production" ? 1 : 0
  premium_messaging_partitions  = var.environment == "production" ? 1 : 0
  public_network_access_enabled = false
  minimum_tls_version           = "1.2"
  zone_redundant                = var.environment == "production"

  tags = {
    Environment = var.environment
  }
}

# Service Bus Queues
resource "azurerm_servicebus_queue" "events" {
  name         = "events"
  namespace_id = azurerm_servicebus_namespace.main.id

  enable_partitioning                     = true
  max_delivery_count                      = 10
  dead_lettering_on_message_expiration    = true
  requires_duplicate_detection            = true
  duplicate_detection_history_time_window = "PT10M"
}

resource "azurerm_servicebus_queue" "notifications" {
  name         = "notifications"
  namespace_id = azurerm_servicebus_namespace.main.id

  enable_partitioning  = true
  max_delivery_count   = 5
  default_message_ttl  = "P14D"
}

# Private endpoint for Service Bus
resource "azurerm_private_endpoint" "servicebus" {
  name                = "${var.cluster_name}-servicebus-pe"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  subnet_id           = azurerm_subnet.private_endpoints.id

  private_service_connection {
    name                           = "servicebus-connection"
    private_connection_resource_id = azurerm_servicebus_namespace.main.id
    subresource_names              = ["namespace"]
    is_manual_connection           = false
  }

  private_dns_zone_group {
    name                 = "servicebus-dns-group"
    private_dns_zone_ids = [azurerm_private_dns_zone.servicebus.id]
  }
}

resource "azurerm_private_dns_zone" "servicebus" {
  name                = "privatelink.servicebus.windows.net"
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_private_dns_zone_virtual_network_link" "servicebus" {
  name                  = "servicebus-vnet-link"
  private_dns_zone_name = azurerm_private_dns_zone.servicebus.name
  resource_group_name   = azurerm_resource_group.main.name
  virtual_network_id    = azurerm_virtual_network.main.id
}

# =============================================================================
# Azure Key Vault
# =============================================================================

resource "azurerm_key_vault" "main" {
  name                          = "${var.cluster_name}-kv"
  resource_group_name           = azurerm_resource_group.main.name
  location                      = azurerm_resource_group.main.location
  tenant_id                     = data.azurerm_client_config.current.tenant_id
  sku_name                      = "premium"
  soft_delete_retention_days    = 90
  purge_protection_enabled      = true
  enable_rbac_authorization     = true
  public_network_access_enabled = false

  network_acls {
    bypass                     = "AzureServices"
    default_action             = "Deny"
    virtual_network_subnet_ids = [azurerm_subnet.aks.id]
  }

  tags = {
    Environment = var.environment
  }
}

# Private endpoint for Key Vault
resource "azurerm_private_endpoint" "keyvault" {
  name                = "${var.cluster_name}-kv-pe"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  subnet_id           = azurerm_subnet.private_endpoints.id

  private_service_connection {
    name                           = "keyvault-connection"
    private_connection_resource_id = azurerm_key_vault.main.id
    subresource_names              = ["vault"]
    is_manual_connection           = false
  }

  private_dns_zone_group {
    name                 = "keyvault-dns-group"
    private_dns_zone_ids = [azurerm_private_dns_zone.keyvault.id]
  }
}

resource "azurerm_private_dns_zone" "keyvault" {
  name                = "privatelink.vaultcore.azure.net"
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_private_dns_zone_virtual_network_link" "keyvault" {
  name                  = "keyvault-vnet-link"
  private_dns_zone_name = azurerm_private_dns_zone.keyvault.name
  resource_group_name   = azurerm_resource_group.main.name
  virtual_network_id    = azurerm_virtual_network.main.id
}

# Grant AKS access to Key Vault
resource "azurerm_role_assignment" "aks_keyvault" {
  scope                = azurerm_key_vault.main.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_user_assigned_identity.aks.principal_id
}

# Store secrets in Key Vault
resource "azurerm_key_vault_secret" "postgresql_password" {
  name         = "postgresql-password"
  value        = random_password.postgresql.result
  key_vault_id = azurerm_key_vault.main.id

  depends_on = [azurerm_role_assignment.aks_keyvault]
}

resource "azurerm_key_vault_secret" "redis_key" {
  name         = "redis-primary-key"
  value        = azurerm_redis_cache.main.primary_access_key
  key_vault_id = azurerm_key_vault.main.id

  depends_on = [azurerm_role_assignment.aks_keyvault]
}

# =============================================================================
# Log Analytics Workspace
# =============================================================================

resource "azurerm_log_analytics_workspace" "main" {
  name                = "${var.cluster_name}-logs"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "PerGB2018"
  retention_in_days   = var.environment == "production" ? 90 : 30

  tags = {
    Environment = var.environment
  }
}

# =============================================================================
# Application Gateway (WAF + Ingress)
# =============================================================================

resource "azurerm_public_ip" "appgw" {
  name                = "${var.cluster_name}-appgw-pip"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  allocation_method   = "Static"
  sku                 = "Standard"
  zones               = ["1", "2", "3"]

  tags = {
    Environment = var.environment
  }
}

resource "azurerm_web_application_firewall_policy" "main" {
  name                = "${var.cluster_name}-waf-policy"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location

  policy_settings {
    enabled                     = true
    mode                        = "Prevention"
    request_body_check          = true
    file_upload_limit_in_mb     = 100
    max_request_body_size_in_kb = 128
  }

  managed_rules {
    managed_rule_set {
      type    = "OWASP"
      version = "3.2"
    }

    managed_rule_set {
      type    = "Microsoft_BotManagerRuleSet"
      version = "1.0"
    }
  }

  custom_rules {
    name      = "RateLimitRule"
    priority  = 1
    rule_type = "RateLimitRule"
    action    = "Block"

    rate_limit_duration_in_minutes = 1
    rate_limit_threshold           = 1000

    match_conditions {
      match_variables {
        variable_name = "RemoteAddr"
      }
      operator           = "IPMatch"
      negation_condition = true
      match_values       = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
    }
  }

  tags = {
    Environment = var.environment
  }
}

# =============================================================================
# Azure Front Door (Global Load Balancing)
# =============================================================================

resource "azurerm_cdn_frontdoor_profile" "main" {
  name                = "${var.cluster_name}-fd"
  resource_group_name = azurerm_resource_group.main.name
  sku_name            = "Premium_AzureFrontDoor"

  tags = {
    Environment = var.environment
  }
}

resource "azurerm_cdn_frontdoor_endpoint" "api" {
  name                     = "api"
  cdn_frontdoor_profile_id = azurerm_cdn_frontdoor_profile.main.id
}

# =============================================================================
# Outputs
# =============================================================================

output "resource_group_name" {
  description = "Resource group name"
  value       = azurerm_resource_group.main.name
}

output "aks_cluster_name" {
  description = "AKS cluster name"
  value       = azurerm_kubernetes_cluster.main.name
}

output "aks_cluster_fqdn" {
  description = "AKS cluster FQDN"
  value       = azurerm_kubernetes_cluster.main.fqdn
}

output "acr_login_server" {
  description = "ACR login server"
  value       = azurerm_container_registry.main.login_server
}

output "postgresql_fqdn" {
  description = "PostgreSQL server FQDN"
  value       = azurerm_postgresql_flexible_server.main.fqdn
}

output "redis_hostname" {
  description = "Redis hostname"
  value       = azurerm_redis_cache.main.hostname
}

output "servicebus_namespace" {
  description = "Service Bus namespace"
  value       = azurerm_servicebus_namespace.main.name
}

output "key_vault_uri" {
  description = "Key Vault URI"
  value       = azurerm_key_vault.main.vault_uri
}

output "front_door_endpoint" {
  description = "Front Door endpoint"
  value       = azurerm_cdn_frontdoor_endpoint.api.host_name
}

output "kubeconfig_command" {
  description = "Command to configure kubectl"
  value       = "az aks get-credentials --resource-group ${azurerm_resource_group.main.name} --name ${azurerm_kubernetes_cluster.main.name}"
}
