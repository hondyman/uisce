schema "public" {}

table "tenant" {
  schema = schema.public
  column "id" {
    type = text
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "created_at" {
    type = timestamp
    null = false
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
}

table "app_user" {
  schema = schema.public
  column "id" {
    type = text
    null = false
  }
  column "email" {
    type = varchar(320)
    null = false
  }
  column "display_name" {
    type = text
  }
  column "created_at" {
    type = timestamp
    null = false
    default = sql("now()")
  }
  column "is_active" {
    type = boolean
    null = false
    default = true
  }
  primary_key {
    columns = [column.id]
  }
  index "ux_app_user_email" {
    unique = true
    columns = [column.email]
  }
}

table "user_tenant" {
  schema = schema.public
  column "user_id" {
    type = text
    null = false
  }
  column "tenant_id" {
    type = text
    null = false
  }
  primary_key {
    columns = [column.user_id, column.tenant_id]
  }
  foreign_key "user_tenant_user_id_fkey" {
    columns = [column.user_id]
    ref_columns = [table.app_user.column.id]
    on_delete = CASCADE
  }
  foreign_key "user_tenant_tenant_id_fkey" {
    columns = [column.tenant_id]
    ref_columns = [table.tenant.column.id]
    on_delete = CASCADE
  }
}

table "asset" {
  schema = schema.public
  column "id" {
    type = uuid
    null = false
  }
  column "tenant_id" {
    type = text
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "asset_type" {
    type = text
    null = false
  }
  column "domain" {
    type = text
    null = false
  }
  column "certified" {
    type = boolean
    null = false
    default = false
  }
  column "sensitivity" {
    type = text
    null = false
    default = sql("'medium'")
  }
  column "created_at" {
    type = timestamp
    null = false
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "asset_tenant_id_fkey" {
    columns = [column.tenant_id]
    ref_columns = [table.tenant.column.id]
    on_delete = CASCADE
  }
  index "idx_asset_tenant_domain" {
    columns = [column.tenant_id, column.domain]
  }
  index "idx_asset_cert" {
    columns = [column.tenant_id, column.certified]
  }
}

# Add remaining tables here (role, claim, policy, etc.) following the same pattern
