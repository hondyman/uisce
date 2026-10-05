# S3 Object Lock Cold Archive Bucket for Regulatory Compliance (SEC 17a-4 / FINRA 4511)
# NOTE: Object Lock MUST be enabled at bucket creation time with versioning enabled.

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

variable "environment" {
  type        = string
  default     = "production"
  description = "Deployment environment (dev, staging, production)"
}

variable "retention_years" {
  type        = number
  default     = 15
  description = "Default compliance retention period in years"
}

variable "bucket_name" {
  type        = string
  default     = "uisce-compliance-cold-archive"
  description = "Name of the S3 WORM archive bucket"
}

resource "aws_s3_bucket" "compliance_archive" {
  bucket        = "${var.bucket_name}-${var.environment}"
  force_destroy = false # Prevent accidental deletion

  object_lock_enabled = true

  tags = {
    Name        = "Uisce Compliance Cold Archive"
    Environment = var.environment
    Compliance  = "SEC-17a-4"
    WORM        = "true"
  }
}

resource "aws_s3_bucket_versioning" "compliance_versioning" {
  bucket = aws_s3_bucket.compliance_archive.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "compliance_encryption" {
  bucket = aws_s3_bucket.compliance_archive.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_object_lock_configuration" "compliance_lock_config" {
  bucket = aws_s3_bucket.compliance_archive.id

  rule {
    default_retention {
      mode  = "COMPLIANCE" # Strict SEC 17a-4 compliance mode (cannot be overridden or deleted)
      years = var.retention_years
    }
  }

  depends_on = [aws_s3_bucket_versioning.compliance_versioning]
}

output "compliance_bucket_arn" {
  value       = aws_s3_bucket.compliance_archive.arn
  description = "ARN of the compliance WORM cold archive bucket"
}

output "compliance_bucket_name" {
  value       = aws_s3_bucket.compliance_archive.id
  description = "ID of the compliance WORM cold archive bucket"
}
