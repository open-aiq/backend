env "defaultConfig" {
  src = "ent://internal/platform/ent/schema"
  dev = "docker://postgres/18/dev"
  url = getenv("DATABASE_URL")
  migration {
    dir = "file://internal/platform/migration/migrations"
  }
}
