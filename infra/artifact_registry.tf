resource "google_artifact_registry_repository" "server" {
  repository_id = "server"
  location      = var.region
  format        = "DOCKER"
  description   = "Server container images."

  # Every deploy pushes a new commit-tagged image, so all but the newest few are deleted to keep
  # storage cost flat. KEEP takes precedence over DELETE, which leaves the serving image and a
  # couple of rollback targets.
  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 3
    }
  }

  cleanup_policies {
    id     = "delete-old"
    action = "DELETE"
    condition {
      tag_state = "ANY"
    }
  }

  depends_on = [google_project_service.required]
}

locals {
  server_image_repository = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.server.repository_id}"
}
