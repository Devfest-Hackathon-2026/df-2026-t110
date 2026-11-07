locals {
  required_services = [
    "artifactregistry.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "sqladmin.googleapis.com",
    "sts.googleapis.com",
  ]
}

resource "google_project_service" "required" {
  for_each = toset(local.required_services)

  service            = each.value
  disable_on_destroy = false
}
