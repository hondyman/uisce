# =============================================================================
# Uisce Semantic OS — Terraform: Production (EU Central 1)
# =============================================================================
# Deploys the Uisce Semantic OS Helm chart into the EU-CENTRAL-1 region.
#
# This module assumes the EKS cluster, RDS, and ElastiCache for eu-central-1
# are already provisioned. Adjust the `eks_cluster_name`, `rds_endpoint`, and
# `redis_endpoint` variables to match your EU infrastructure.
#
# For full infrastructure provisioning (VPC, EKS, RDS, ElastiCache), use the
# production-us/ module as a reference and parameterize by region.
#
# Usage:
#   terraform init   -backend-config="key=infrastructure/eu-central/terraform.tfstate"
#   terraform plan   -var="aws_region=eu-central-1"
#   terraform apply -var="aws_region=eu-central-1"
# =============================================================================

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.25"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
  }

  # Separate state file per region for isolation
  backend "s3" {
    bucket         = "uisce-terraform-state"
    key            = "infrastructure/eu-central/terraform.tfstate"
    region         = "eu-central-1"
    encrypt        = true
    dynamodb_table = "uisce-terraform-locks"
  }
}

# =============================================================================
# Provider
# =============================================================================

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "uisce"
      Environment = var.environment
      ManagedBy   = "terraform"
      Team        = "platform"
    }
  }
}

# =============================================================================
# Variables
# =============================================================================

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "eu-central-1"
}

variable "eks_cluster_name" {
  description = "EKS cluster name in eu-central-1"
  type        = string
  default     = ""   # Set via terraform.tfvars or -var
}

variable "rds_endpoint" {
  description = "RDS PostgreSQL endpoint in eu-central-1"
  type        = string
  default     = ""   # Set via terraform.tfvars or -var
}

variable "redis_endpoint" {
  description = "ElastiCache Redis endpoint in eu-central-1"
  type        = string
  default     = ""   # Set via terraform.tfvars or -var
}

variable "kafka_brokers" {
  description = "Kafka/Redpanda broker list for eu-central-1"
  type        = string
  default     = "redpanda:9092"
}

variable "infisical_project_id" {
  description = "Infisical project ID for secret injection"
  type        = string
  default     = ""
}

# =============================================================================
# Data Sources
# =============================================================================

data "aws_eks_cluster" "eks" {
  count = var.eks_cluster_name != "" ? 1 : 0
  name  = var.eks_cluster_name
}

data "aws_eks_cluster_auth" "eks" {
  count = var.eks_cluster_name != "" ? 1 : 0
  name  = var.eks_cluster_name
}

# =============================================================================
# Kubernetes Provider
# =============================================================================

provider "kubernetes" {
  host                   = data.aws_eks_cluster.eks[0].endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks[0].certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.eks[0].token
}

# =============================================================================
# Uisce Helm Release — EU Central 1
# =============================================================================

resource "helm_release" "uisce_eu_central" {
  name       = "uisce-eu-central"
  chart      = "${path.module}/../../deploy/helm"
  namespace  = "uisce"
  create_namespace = true

  values = [
    file("${path.module}/../../deploy/helm/values-prod-eu-central.yaml"),
  ]

  set {
    name  = "global.region"
    value = "eu-central-1"
  }

  set {
    name  = "global.environment"
    value = var.environment
  }

  set {
    name  = "raftLedger.enabled"
    value = "true"
  }

  set {
    name  = "raftLedger.nodeID"
    value = "uisce-node-eu-central-1"
  }

  set {
    name  = "raftLedger.clusterPeers[0]"
    value = "uisce-node-us-east-1.uisce.svc.cluster.local:7946"
  }

  set {
    name  = "raftLedger.clusterPeers[1]"
    value = "uisce-node-eu-central-1.uisce.svc.cluster.local:7946"
  }

  set {
    name  = "raftLedger.clusterPeers[2]"
    value = "uisce-node-ap-east-1.uisce.svc.cluster.local:7946"
  }

  set {
    name  = "infisical.enabled"
    value = var.infinisical_project_id != "" ? "true" : "false"
  }

  set {
    name  = "infisical.secretRef"
    value = "uisce-infisical-secrets-eu-central"
  }

  set {
    name  = "image.repository"
    value = "uisce-core-local"
  }

  set {
    name  = "image.pullPolicy"
    value = "Never"
  }

  set {
    name  = "replicaCount"
    value = "3"
  }

  set {
    name  = "autoscaling.enabled"
    value = "true"
  }

  set {
    name  = "autoscaling.minReplicas"
    value = "3"
  }

  set {
    name  = "autoscaling.maxReplicas"
    value = "12"
  }

  set {
    name  = "podSecurityContext.runAsNonRoot"
    value = "true"
  }

  set {
    name  = "containerSecurityContext.readOnlyRootFilesystem"
    value = "true"
  }

  set {
    name  = "containerSecurityContext.allowPrivilegeEscalation"
    value = "false"
  }

  set {
    name  = "containerSecurityContext.capabilities.drop[0]"
    value = "ALL"
  }

  set {
    name  = "podDisruptionBudget.enabled"
    value = "true"
  }

  set {
    name  = "podDisruptionBudget.minAvailable"
    value = "2"
  }

  set {
    name  = "networkPolicy.enabled"
    value = "true"
  }

  depends_on = [data.aws_eks_cluster.eks]
}

# =============================================================================
# Outputs
# =============================================================================

output "helm_release_name" {
  description = "Helm release name for EU Central"
  value       = resource.helm_release.uisce_eu_central.name
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = var.eks_cluster_name
}
