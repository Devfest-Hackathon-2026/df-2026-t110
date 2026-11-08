resource "google_service_account" "server" {
  account_id   = "server"
  display_name = "Server (Cloud Run)"
}

resource "google_project_iam_member" "server_cloudsql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = google_service_account.server.member
}

resource "google_secret_manager_secret_iam_member" "server_db_app_password" {
  secret_id = google_secret_manager_secret.db_app_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.server.member
}

resource "google_secret_manager_secret_iam_member" "server_google_login_client_secret" {
  count = local.google_login_enabled ? 1 : 0

  secret_id = google_secret_manager_secret.google_login_client_secret[0].id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.server.member
}

resource "google_cloud_run_v2_service" "server" {
  name     = "server"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  iap_enabled = var.iap_enabled

  deletion_protection = false

  template {
    service_account = google_service_account.server.email
    # Cloud Storage volume mounts require the second generation execution environment.
    execution_environment = "EXECUTION_ENVIRONMENT_GEN2"

    scaling {
      min_instance_count = 0
      max_instance_count = var.server_max_instances
    }

    # Volumes and mounts are listed by name: Cloud Run returns them sorted, and any other order
    # shows up as a permanent diff in every plan.
    volumes {
      name = "client"
      gcs {
        bucket    = google_storage_bucket.client.name
        read_only = true
      }
    }

    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.main.connection_name]
      }
    }

    containers {
      image = var.server_image

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
        cpu_idle = true
      }

      env {
        name  = "STATIC_DIR"
        value = "/static"
      }
      # The server only reads DATABASE_URL. The password stays out of it and comes from PGPASSWORD,
      # which pgx applies to keyword/value connection strings like libpq does.
      env {
        name  = "DATABASE_URL"
        value = "host=/cloudsql/${google_sql_database_instance.main.connection_name} dbname=${google_sql_database.app.name} user=${google_sql_user.app.name}"
      }
      env {
        name = "PGPASSWORD"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.db_app_password.secret_id
            version = "latest"
          }
        }
      }

      # Without these the server starts with Google sign-in disabled and /api/login answers 501.
      dynamic "env" {
        for_each = local.google_login_enabled ? {
          GOOGLE_CLIENT_ID    = var.google_login_client_id
          GOOGLE_REDIRECT_URL = var.google_login_redirect_url
        } : {}
        content {
          name  = env.key
          value = env.value
        }
      }
      dynamic "env" {
        for_each = google_secret_manager_secret.google_login_client_secret
        content {
          name = "GOOGLE_CLIENT_SECRET"
          value_source {
            secret_key_ref {
              secret  = env.value.secret_id
              version = "latest"
            }
          }
        }
      }

      volume_mounts {
        name       = "client"
        mount_path = "/static"
      }
      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }
    }
  }

  # GitHub Actions deploys new images with gcloud; Terraform only sets the initial one.
  # client/client_version are written by gcloud on every deploy.
  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
    ]
  }

  depends_on = [
    google_project_service.required,
    google_project_iam_member.server_cloudsql_client,
    google_secret_manager_secret_iam_member.server_db_app_password,
    google_secret_manager_secret_version.db_app_password,
    google_secret_manager_secret_iam_member.server_google_login_client_secret,
    google_secret_manager_secret_version.google_login_client_secret,
    google_storage_bucket_iam_member.server_client_reader,
  ]
}

# Once IAP is off, the app authenticates users itself (Google sign-in), so the service is publicly invokable.
resource "google_cloud_run_v2_service_iam_member" "server_public" {
  count = var.iap_enabled ? 0 : 1

  name     = google_cloud_run_v2_service.server.name
  location = google_cloud_run_v2_service.server.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
