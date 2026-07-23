env "dev" {
  url = "postgres://user:pass@localhost:5432/aisvc?sslmode=disable"
  src = "file://schema.hcl"
  dev = "docker://postgres/15/dev"
}

env "prod" {
  url = "env://DATABASE_URL"
  src = "file://schema.hcl"
}

migrate {
  dir = "file://migrations"
}
