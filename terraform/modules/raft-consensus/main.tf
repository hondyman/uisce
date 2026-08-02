# =============================================================================
# Terraform Module: raft-consensus
# Provisions the networking infrastructure required for a multi-region
# HashiCorp Raft consensus cluster:
#   - Dedicated subnet for Raft traffic (TCP 7946)
#   - Security group allowing TCP 7946 between Raft peers only
#   - IAM role + policy for EBS-backed StatefulSet persistence
# =============================================================================

variable "region" {
  description = "AWS region for this Raft node"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID to deploy Raft subnet into"
  type        = string
}

variable "cluster_peers" {
  description = "List of Raft peer addresses (FQDN:7946)"
  type        = list(string)
  default     = []
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}

# ----------------------------------------------------------------------------
# Data sources
# ----------------------------------------------------------------------------
data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_caller_identity" "current" {}

# ----------------------------------------------------------------------------
# Raft subnet — isolated subnet for consensus traffic
# ----------------------------------------------------------------------------
module "raft_subnet" {
  source = "terraform-aws-modules/vpc/aws//modules/subnets"

  name = "uisce-raft-${var.region}"
  vpc_id = var.vpc_id

  subnets = {
    for i, az in slice(data.aws_availability_zones.available.names, 0, 1) :
    "uisce-raft-${var.region}-${az}" => {
      cidr              = cidrsubnet(var.vpc_id == "" ? "10.0.0.0/16" : "10.1.0.0/16", 4, i + 10)
      availability_zone = az
    }
  }

  tags = merge(var.tags, {
    Name        = "uisce-raft-subnet-${var.region}"
    Component   = "raft-consensus"
    Environment = var.region
  })
}

# ----------------------------------------------------------------------------
# Security group — TCP 7946 only between Raft peers
# ----------------------------------------------------------------------------
resource "aws_security_group" "raft_consensus" {
  name        = "uisce-raft-consensus-${var.region}"
  description = "Allow Raft consensus traffic (TCP 7946) between Uisce peers"
  vpc_id      = var.vpc_id

  tags = merge(var.tags, {
    Name        = "uisce-raft-sg-${var.region}"
    Component   = "raft-consensus"
    Environment = var.region
  })
}

resource "aws_vpc_security_group_ingress_rule" "raft_peers" {
  for_each = toset(var.cluster_peers)

  security_group_id = aws_security_group.raft_consensus.id
  description       = "Raft peer: ${each.value}"
  from_port         = 7946
  ip_protocol       = "tcp"
  to_port           = 7946
  cidr_ipv4         = split("/", each.value)[0] # extract CIDR from FQDN (best-effort)
}

resource "aws_vpc_security_group_egress_rule" "raft_peers" {
  for_each = toset(var.cluster_peers)

  security_group_id = aws_security_group.raft_consensus.id
  description        = "Raft peer: ${each.value}"
  from_port          = 7946
  ip_protocol        = "tcp"
  to_port            = 7946
  cidr_ipv4          = split("/", each.value)[0]
}

resource "aws_vpc_security_group_ingress_rule" "raft_self" {
  security_group_id = aws_security_group.raft_consensus.id
  description       = "Self-referential for Raft intra-node communication"
  from_port         = 7946
  ip_protocol       = "tcp"
  to_port           = 7946
  referenced_group_id = aws_security_group.raft_consensus.id
}

# ----------------------------------------------------------------------------
# IAM role for EBS CSI driver (StatefulSet persistent volumes)
# ----------------------------------------------------------------------------
data "aws_iam_policy_document" "raft_ebs_csi_assume_role" {
  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "raft_ebs_csi" {
  name               = "uisce-raft-ebs-csi-${var.region}"
  assume_role_policy = data.aws_iam_policy_document.raft_ebs_csi_assume_role.json
  tags               = var.tags
}

resource "aws_iam_policy" "raft_ebs_csi_policy" {
  name = "uisce-raft-ebs-csi-policy-${var.region}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["ec2:CreateVolume", "ec2:DeleteVolume", "ec2:AttachVolume", "ec2:DetachVolume"]
        Resource = "arn:aws:ec2:*:*:volume/*"
      },
      {
        Effect   = "Allow"
        Action   = ["ec2:DescribeVolumes"]
        Resource = "*"
      }
    ]
  })
  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "raft_ebs_csi_attach" {
  role       = aws_iam_role.raft_ebs_csi.name
  policy_arn = aws_iam_policy.raft_ebs_csi_policy.arn
}

# ----------------------------------------------------------------------------
# Outputs
# ----------------------------------------------------------------------------
output "raft_subnet_ids" {
  description = "Subnet IDs for Raft traffic"
  value       = values(module.raft_subnet.subnets)[*].id
}

output "raft_security_group_id" {
  description = "Security group ID for Raft consensus"
  value       = aws_security_group.raft_consensus.id
}

output "raft_ebs_csi_role_arn" {
  description = "IAM role ARN for EBS CSI driver"
  value       = aws_iam_role.raft_ebs_csi.arn
}
