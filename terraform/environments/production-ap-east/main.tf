# =============================================================================
# Uisce Semantic OS — Terraform: Production (AP East 1)
# =============================================================================
# Mirrors production-eu-central/main.tf for ap-east-1.
# See production-eu-central/main.tf for full documentation.
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

  backend "s3" {
    bucket         = "uisce-terraform-state"
    key            = "infrastructure/ap-east/terraform.tfstate"
    region         = "ap-east-1"
    encrypt        = true
    dynamodb_table = "uisce-terraform-locks"
  }
}

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

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "ap-east-1"
}

variable "eks_cluster_name" {
  description = "EKS cluster name in ap-east-1"
  type        = string
  default     = ""
}

variable "infisical_project_id" {
  description = "Infisical project ID for secret injection"
  type        = string
  default     = ""
}

data "aws_eks_cluster" "eks" {
  count = var.eks_cluster_name != "" ? 1 : 0
  name  = var.eks_cluster_name
}

data "aws_eks_cluster_auth" "eks" {
  count = var.eks_cluster_name != "" ? 1 : 0
  name  = var.eks_cluster_name
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.eks[0].endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks[0].certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.eks[0].token
}

resource "helm_release" "uisce_ap_east" {
  name       = "uisce-ap-east"
  chart      = "${path.module}/../../deploy/helm"
  namespace  = "uisce"
  create_namespace = true

  values = [
    file("${path.module}/../../deploy/helm/values-prod-ap-east.yaml"),
  ]

  set {
    name  = "global.region"
    value = "ap-east-1"
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
    value = "uisce-node-ap-east-1"
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
    value = "uisce-infisical-secrets-ap-east"
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

output "helm_release_name" {
  description = "Helm release name for AP East"
  value       = resource.helm_release.uisce_ap_east.name
}
