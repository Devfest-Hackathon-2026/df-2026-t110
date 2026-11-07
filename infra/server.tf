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

resource "google_cloud_run_v2_service" "server" {
  name     = "server"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  iap_enabled = var.iap_enabled

  deletion_protection = false

  template {
    service_account = google_service_account.server.email

    scaling {
      min_instance_count = 0
      max_instance_count = var.server_max_instances
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
        name  = "DB_HOST"
        value = "/cloudsql/${google_sql_database_instance.main.connection_name}"
      }
      env {
        name  = "DB_NAME"
        value = google_sql_database.app.name
      }
      env {
        name  = "DB_USER"
        value = google_sql_user.app.name
      }
      env {
        name = "DB_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.db_app_password.secret_id
            version = "latest"
          }
        }
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
