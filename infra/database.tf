resource "google_sql_database_instance" "main" {
  name             = "main"
  database_version = "POSTGRES_18"
  region           = var.region

  deletion_protection = var.db_deletion_protection

  settings {
    tier              = var.db_tier
    edition           = "ENTERPRISE"
    availability_type = "ZONAL"
    activation_policy = var.db_running ? "ALWAYS" : "NEVER"
    disk_type         = "PD_HDD" # Cheapest disk; latency is not a concern at this scale. Cannot be changed later.
    disk_size         = 10
    disk_autoresize   = true

    deletion_protection_enabled = var.db_deletion_protection

    backup_configuration {
      enabled    = true
      start_time = "18:00" # 03:00 JST
      # Daily backups alone meet the recovery goal (up to a day of data loss is acceptable);
      # PITR would add transaction log storage cost on top.
      point_in_time_recovery_enabled = false

      backup_retention_settings {
        retained_backups = 7
      }
    }

    # Public IP without authorized networks: only the Cloud SQL connector (IAM-checked) can reach it,
    # which avoids the idle cost of a VPC connector or private service access.
    ip_configuration {
      ipv4_enabled = true
      ssl_mode     = "ENCRYPTED_ONLY"
    }

    insights_config {
      query_insights_enabled = true
    }
  }

  depends_on = [google_project_service.required]
}

resource "google_sql_database" "app" {
  name     = "app"
  instance = google_sql_database_instance.main.name
}

resource "random_password" "db_app" {
  length  = 32
  special = false
}

resource "google_sql_user" "app" {
  name     = "app"
  instance = google_sql_database_instance.main.name
  password = random_password.db_app.result
}

resource "google_secret_manager_secret" "db_app_password" {
  secret_id = "db-app-password"

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "db_app_password" {
  secret      = google_secret_manager_secret.db_app_password.id
  secret_data = random_password.db_app.result
}
