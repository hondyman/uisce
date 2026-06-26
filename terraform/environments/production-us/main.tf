# Semlayer Infrastructure - Terraform Configuration
# Multi-cloud, multi-region enterprise deployment

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
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  # Remote state configuration - use S3 for production
  backend "s3" {
    bucket         = "semlayer-terraform-state"
    key            = "infrastructure/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "semlayer-terraform-locks"
  }
}

# =============================================================================
# Provider Configuration
# =============================================================================

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "semlayer"
      Environment = var.environment
      ManagedBy   = "terraform"
      Team        = "platform"
    }
  }
}

provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}

provider "aws" {
  alias  = "us_west_2"
  region = "us-west-2"
}

provider "aws" {
  alias  = "eu_west_1"
  region = "eu-west-1"
}

provider "aws" {
  alias  = "ap_southeast_1"
  region = "ap-southeast-1"
}

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

variable "aws_region" {
  description = "Primary AWS region"
  type        = string
  default     = "us-east-1"
}

variable "cluster_name" {
  description = "EKS cluster name"
  type        = string
  default     = "semlayer-prod"
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "kubernetes_version" {
  description = "Kubernetes version for EKS"
  type        = string
  default     = "1.29"
}

variable "enable_multi_region" {
  description = "Enable multi-region deployment"
  type        = bool
  default     = false
}

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.r6g.xlarge"
}

variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.r6g.large"
}

# =============================================================================
# Data Sources
# =============================================================================

data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

# =============================================================================
# VPC Module
# =============================================================================

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${var.cluster_name}-vpc"
  cidr = var.vpc_cidr

  azs             = slice(data.aws_availability_zones.available.names, 0, 3)
  private_subnets = [for k, v in slice(data.aws_availability_zones.available.names, 0, 3) : cidrsubnet(var.vpc_cidr, 4, k)]
  public_subnets  = [for k, v in slice(data.aws_availability_zones.available.names, 0, 3) : cidrsubnet(var.vpc_cidr, 4, k + 4)]
  intra_subnets   = [for k, v in slice(data.aws_availability_zones.available.names, 0, 3) : cidrsubnet(var.vpc_cidr, 4, k + 8)]

  enable_nat_gateway     = true
  single_nat_gateway     = var.environment == "staging"
  one_nat_gateway_per_az = var.environment == "production"

  enable_dns_hostnames = true
  enable_dns_support   = true

  # VPC Flow Logs
  enable_flow_log                      = true
  create_flow_log_cloudwatch_log_group = true
  create_flow_log_cloudwatch_iam_role  = true
  flow_log_max_aggregation_interval    = 60

  # Tags for EKS
  public_subnet_tags = {
    "kubernetes.io/role/elb"                    = 1
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb"           = 1
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "karpenter.sh/discovery"                    = var.cluster_name
  }

  tags = {
    Environment = var.environment
    Terraform   = "true"
  }
}

# =============================================================================
# EKS Cluster Module
# =============================================================================

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = var.cluster_name
  cluster_version = var.kubernetes_version

  cluster_endpoint_public_access  = true
  cluster_endpoint_private_access = true

  # Cluster access entry
  enable_cluster_creator_admin_permissions = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  # Cluster addons
  cluster_addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
      configuration_values = jsonencode({
        env = {
          ENABLE_PREFIX_DELEGATION          = "true"
          WARM_PREFIX_TARGET                = "1"
          ENI_CONFIG_LABEL_DEF              = "topology.kubernetes.io/zone"
          AWS_VPC_K8S_CNI_CUSTOM_NETWORK_CFG = "true"
        }
      })
    }
    aws-ebs-csi-driver = {
      most_recent              = true
      service_account_role_arn = module.ebs_csi_irsa.iam_role_arn
    }
  }

  # Node groups
  eks_managed_node_groups = {
    # System node group
    system = {
      name           = "system"
      instance_types = ["m6i.xlarge"]
      capacity_type  = "ON_DEMAND"

      min_size     = 3
      max_size     = 5
      desired_size = 3

      labels = {
        workload = "system"
      }

      taints = [{
        key    = "dedicated"
        value  = "system"
        effect = "NO_SCHEDULE"
      }]

      update_config = {
        max_unavailable_percentage = 33
      }
    }

    # API node group
    api = {
      name           = "api"
      instance_types = ["c6i.2xlarge"]
      capacity_type  = "ON_DEMAND"

      min_size     = 3
      max_size     = 50
      desired_size = 5

      labels = {
        workload = "api"
      }

      update_config = {
        max_unavailable_percentage = 25
      }
    }

    # Cube workers node group (Spot for cost optimization)
    cube-workers = {
      name           = "cube-workers"
      instance_types = ["r6i.2xlarge", "r6i.4xlarge", "r5.2xlarge"]
      capacity_type  = "SPOT"

      min_size     = 3
      max_size     = 100
      desired_size = 10

      labels = {
        workload = "cube"
      }

      update_config = {
        max_unavailable_percentage = 50
      }
    }

    # Data services node group
    data = {
      name           = "data"
      instance_types = ["r6i.4xlarge"]
      capacity_type  = "ON_DEMAND"

      min_size     = 3
      max_size     = 20
      desired_size = 3

      labels = {
        workload = "data"
      }

      update_config = {
        max_unavailable_percentage = 33
      }

      # Use gp3 EBS for better performance
      block_device_mappings = {
        xvda = {
          device_name = "/dev/xvda"
          ebs = {
            volume_size           = 100
            volume_type           = "gp3"
            iops                  = 3000
            throughput            = 125
            encrypted             = true
            delete_on_termination = true
          }
        }
      }
    }
  }

  # Fargate profiles for specific workloads (optional)
  fargate_profiles = var.environment == "production" ? {
    karpenter = {
      name = "karpenter"
      selectors = [
        { namespace = "karpenter" }
      ]
    }
  } : {}

  tags = {
    Environment                                 = var.environment
    "karpenter.sh/discovery"                    = var.cluster_name
  }
}

# =============================================================================
# EBS CSI Driver IRSA
# =============================================================================

module "ebs_csi_irsa" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.0"

  role_name             = "${var.cluster_name}-ebs-csi"
  attach_ebs_csi_policy = true

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kube-system:ebs-csi-controller-sa"]
    }
  }
}

# =============================================================================
# RDS PostgreSQL
# =============================================================================

module "rds" {
  source  = "terraform-aws-modules/rds/aws"
  version = "~> 6.0"

  identifier = "${var.cluster_name}-postgres"

  engine               = "postgres"
  engine_version       = "15.4"
  family               = "postgres15"
  major_engine_version = "15"
  instance_class       = var.db_instance_class

  allocated_storage     = 100
  max_allocated_storage = 1000
  storage_type          = "gp3"
  storage_encrypted     = true

  db_name  = "semlayer"
  username = "semlayer_admin"
  port     = 5432

  # Multi-AZ for production
  multi_az = var.environment == "production"

  # VPC configuration
  db_subnet_group_name   = module.vpc.database_subnet_group_name
  vpc_security_group_ids = [module.rds_security_group.security_group_id]

  # Maintenance
  maintenance_window              = "Mon:00:00-Mon:03:00"
  backup_window                   = "03:00-06:00"
  backup_retention_period         = var.environment == "production" ? 35 : 7
  skip_final_snapshot             = var.environment != "production"
  deletion_protection             = var.environment == "production"
  performance_insights_enabled    = true
  performance_insights_retention_period = 7
  create_monitoring_role          = true
  monitoring_interval             = 60

  # Parameter group
  parameters = [
    {
      name  = "log_min_duration_statement"
      value = "1000"  # Log queries > 1 second
    },
    {
      name  = "shared_preload_libraries"
      value = "pg_stat_statements"
    },
    {
      name  = "pg_stat_statements.track"
      value = "all"
    }
  ]

  tags = {
    Environment = var.environment
  }
}

module "rds_security_group" {
  source  = "terraform-aws-modules/security-group/aws"
  version = "~> 5.0"

  name        = "${var.cluster_name}-rds-sg"
  description = "Security group for RDS PostgreSQL"
  vpc_id      = module.vpc.vpc_id

  ingress_with_source_security_group_id = [
    {
      from_port                = 5432
      to_port                  = 5432
      protocol                 = "tcp"
      description              = "PostgreSQL access from EKS"
      source_security_group_id = module.eks.node_security_group_id
    }
  ]

  tags = {
    Environment = var.environment
  }
}

# =============================================================================
# ElastiCache Redis
# =============================================================================

module "elasticache" {
  source = "terraform-aws-modules/elasticache/aws"
  version = "~> 1.0"

  cluster_id               = "${var.cluster_name}-redis"
  create_cluster           = true
  create_replication_group = false

  engine          = "redis"
  engine_version  = "7.0"
  node_type       = var.redis_node_type
  num_cache_nodes = var.environment == "production" ? 3 : 1

  # Cluster mode
  parameter_group_family = "redis7"

  # Network
  subnet_ids         = module.vpc.private_subnets
  security_group_ids = [module.redis_security_group.security_group_id]

  # Maintenance
  maintenance_window = "sun:05:00-sun:09:00"
  snapshot_window    = "00:00-05:00"

  # Encryption
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true

  tags = {
    Environment = var.environment
  }
}

module "redis_security_group" {
  source  = "terraform-aws-modules/security-group/aws"
  version = "~> 5.0"

  name        = "${var.cluster_name}-redis-sg"
  description = "Security group for ElastiCache Redis"
  vpc_id      = module.vpc.vpc_id

  ingress_with_source_security_group_id = [
    {
      from_port                = 6379
      to_port                  = 6379
      protocol                 = "tcp"
      description              = "Redis access from EKS"
      source_security_group_id = module.eks.node_security_group_id
    }
  ]

  tags = {
    Environment = var.environment
  }
}

# =============================================================================
# Amazon MQ (RabbitMQ)
# =============================================================================

resource "aws_mq_broker" "rabbitmq" {
  broker_name = "${var.cluster_name}-rabbitmq"

  engine_type        = "RabbitMQ"
  engine_version     = "3.11.28"
  host_instance_type = var.environment == "production" ? "mq.m5.large" : "mq.t3.micro"
  deployment_mode    = var.environment == "production" ? "CLUSTER_MULTI_AZ" : "SINGLE_INSTANCE"

  security_groups = [module.mq_security_group.security_group_id]
  subnet_ids      = var.environment == "production" ? slice(module.vpc.private_subnets, 0, 2) : [module.vpc.private_subnets[0]]

  user {
    username = "semlayer"
    password = random_password.mq_password.result
  }

  encryption_options {
    use_aws_owned_key = false
    kms_key_id        = aws_kms_key.main.arn
  }

  logs {
    general = true
  }

  maintenance_window_start_time {
    day_of_week = "MONDAY"
    time_of_day = "02:00"
    time_zone   = "UTC"
  }

  tags = {
    Environment = var.environment
  }
}

resource "random_password" "mq_password" {
  length  = 32
  special = false
}

module "mq_security_group" {
  source  = "terraform-aws-modules/security-group/aws"
  version = "~> 5.0"

  name        = "${var.cluster_name}-mq-sg"
  description = "Security group for Amazon MQ RabbitMQ"
  vpc_id      = module.vpc.vpc_id

  ingress_with_source_security_group_id = [
    {
      from_port                = 5671
      to_port                  = 5671
      protocol                 = "tcp"
      description              = "AMQPS access from EKS"
      source_security_group_id = module.eks.node_security_group_id
    },
    {
      from_port                = 443
      to_port                  = 443
      protocol                 = "tcp"
      description              = "Management console access"
      source_security_group_id = module.eks.node_security_group_id
    }
  ]

  tags = {
    Environment = var.environment
  }
}

# =============================================================================
# KMS Key for Encryption
# =============================================================================

resource "aws_kms_key" "main" {
  description             = "Semlayer main encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow EKS to use the key"
        Effect = "Allow"
        Principal = {
          Service = "eks.amazonaws.com"
        }
        Action = [
          "kms:Encrypt",
          "kms:Decrypt",
          "kms:ReEncrypt*",
          "kms:GenerateDataKey*",
          "kms:DescribeKey"
        ]
        Resource = "*"
      }
    ]
  })

  tags = {
    Name        = "${var.cluster_name}-main-key"
    Environment = var.environment
  }
}

resource "aws_kms_alias" "main" {
  name          = "alias/${var.cluster_name}-main"
  target_key_id = aws_kms_key.main.key_id
}

# =============================================================================
# Secrets Manager
# =============================================================================

resource "aws_secretsmanager_secret" "database" {
  name        = "${var.cluster_name}/database"
  description = "Database credentials for Semlayer"
  kms_key_id  = aws_kms_key.main.arn

  tags = {
    Environment = var.environment
  }
}

resource "aws_secretsmanager_secret_version" "database" {
  secret_id = aws_secretsmanager_secret.database.id
  secret_string = jsonencode({
    username = module.rds.db_instance_username
    password = module.rds.db_instance_password
    host     = module.rds.db_instance_address
    port     = module.rds.db_instance_port
    database = "semlayer"
  })
}

resource "aws_secretsmanager_secret" "rabbitmq" {
  name        = "${var.cluster_name}/rabbitmq"
  description = "RabbitMQ credentials for Semlayer"
  kms_key_id  = aws_kms_key.main.arn

  tags = {
    Environment = var.environment
  }
}

resource "aws_secretsmanager_secret_version" "rabbitmq" {
  secret_id = aws_secretsmanager_secret.rabbitmq.id
  secret_string = jsonencode({
    username = "semlayer"
    password = random_password.mq_password.result
    host     = aws_mq_broker.rabbitmq.instances[0].endpoints[0]
  })
}

# =============================================================================
# Outputs
# =============================================================================

output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "rds_endpoint" {
  description = "RDS endpoint"
  value       = module.rds.db_instance_endpoint
}

output "redis_endpoint" {
  description = "Redis endpoint"
  value       = module.elasticache.cluster_cache_nodes
}

output "rabbitmq_endpoint" {
  description = "RabbitMQ endpoint"
  value       = aws_mq_broker.rabbitmq.instances[0].endpoints[0]
}

output "kms_key_arn" {
  description = "KMS key ARN"
  value       = aws_kms_key.main.arn
}

output "kubeconfig_command" {
  description = "Command to configure kubectl"
  value       = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
}
