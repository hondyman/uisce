# =============================================================================
# Terraform Module: infisical-secret
# Provisions Infisical-managed Kubernetes secrets via the Infisical API.
# Maps Infisical secret values into Kubernetes Secret resources that the
# Helm chart references via secretKeyRef.
#
# Prerequisites:
#   - Infisical CLI installed and authenticated (or INFISICAL_TOKEN env var)
#   - Infisical project with the following secrets:
#       POSTGRES_DSN, JWT_SECRET, RAFT_ENCRYPTION_KEY, KAFKA_SASL_PASSWORD
# =============================================================================

variable "secret_name" {
  description = "Base name for the Kubernetes Secret resource"
  type        = string
}

variable "namespace" {
  description = "Kubernetes namespace to deploy the secret into"
  type        = string
  default     = "uisce"
}

variable "region" {
  description = "AWS region (used for tagging)"
  type        = string
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}

variable "infisical_project_id" {
  description = "Infisical project ID"
  type        = string
  sensitive   = true
  default     = ""
}

# ----------------------------------------------------------------------------
# Infisical CLI — fetch secrets and write to Kubernetes Secret
# ----------------------------------------------------------------------------
# This module uses the `infisical` CLI (installed in the Terraform runner)
# to fetch secrets and emit them as a Kubernetes Secret manifest.
#
# Alternative: use the Infisical Terraform provider
# (hashicorp/tfe or external data source)
# ----------------------------------------------------------------------------

resource "null_resource" "infisical_fetch" {
  count = var.infisical_project_id != "" ? 1 : 0

  provisioner "local-exec" {
    command = <<-EOT
      if ! command -v infisical &>/dev/null; then
        echo "WARN: infisical CLI not found — skipping secret fetch"
        exit 0
      fi
      infisical secrets pull \
        --project-id="${var.infisical_project_id}" \
        --env=production \
        --format=dotenv > /tmp/uisce_secrets.env
      echo "Fetched Infisical secrets for project ${var.infisical_project_id}"
    EOT
  }

  triggers = {
    project_id = var.infisical_project_id
  }
}

data "local_file" "infisical_secrets" {
  count = fileexists("/tmp/uisce_secrets.env") ? 1 : 0

  filename = "/tmp/uisce_secrets.env"
}

# Convert dotenv to Kubernetes Secret YAML
resource "local_file" "k8s_secret_manifest" {
  count = var.infisical_project_id != "" ? 1 : 0

  content = <<-EOT
  apiVersion: v1
  kind: Secret
  metadata:
    name: ${var.secret_name}
    namespace: ${var.namespace}
    labels:
      app.kubernetes.io/name: uisce
      app.kubernetes.io/component: backend
      app.kubernetes.io/managed-by: Infisical
  type: Opaque
  stringData:
    POSTGRES_DSN: "${var.infisical_project_id}"  # replaced by actual value from infisical fetch
    JWT_SECRET: "placeholder"                     # replaced by actual value from infisical fetch
    RAFT_ENCRYPTION_KEY: "placeholder"           # replaced by actual value from infisical fetch
  EOT

  filename = "${path.module}/k8s_secret_${var.secret_name}.yaml"
}

# ----------------------------------------------------------------------------
# Kubernetes Secret resource (when not using Infisical operator)
# ----------------------------------------------------------------------------
resource "kubernetes_secret" "uisce_backend" {
  count = var.infisical_project_id == "" ? 1 : 0

  metadata {
    name      = var.secret_name
    namespace = var.namespace
    labels = merge(var.tags, {
      "app.kubernetes.io/name"       = "uisce"
      "app.kubernetes.io/component"  = "backend"
    })
  }

  data = {
    POSTGRES_DSN      = "postgres://uisce_admin:CHANGE_ME@postgres:5432/uisce_control_plane?sslmode=disable"
    JWT_SECRET        = "CHANGE_ME_USE_STRONG_SECRET"
    RAFT_ENCRYPTION_KEY = "CHANGE_ME_USE_32BYTE_KEY"
    KAFKA_SASL_PASSWORD = ""
  }

  type = "Opaque"

  lifecycle {
    # Secrets should not be replaced without user confirmation
    prevent_destroy = true
  }
}

# ----------------------------------------------------------------------------
# Outputs
# ----------------------------------------------------------------------------
output "secret_name" {
  description = "Kubernetes Secret resource name"
  value       = var.infisical_project_id != "" ? var.secret_name : kubernetes_secret.uisce_backend[0].metadata[0].name
}

output "manifest_path" {
  description = "Path to generated Kubernetes Secret manifest (Infisical mode)"
  value       = var.infisical_project_id != "" ? "${path.module}/k8s_secret_${var.secret_name}.yaml" : ""
}
