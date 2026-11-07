# The state bucket has to exist before `terraform init`, so it is created once with gcloud
# (see backend.hcl.example) and adopted here to keep its settings in code.
import {
  to = google_storage_bucket.tfstate
  id = "${var.project_id}-tfstate"
}

resource "google_storage_bucket" "tfstate" {
  name     = "${var.project_id}-tfstate"
  location = var.region

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  # Versioning lets a broken state be rolled back; old versions are pruned to keep storage cost flat.
  versioning {
    enabled = true
  }

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      num_newer_versions = 10
      with_state         = "ARCHIVED"
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}
