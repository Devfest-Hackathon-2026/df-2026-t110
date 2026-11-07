# IAP keeps the frontend and API private to iap_members until public release.
#
# One-time manual setup (the project has no organization, so Terraform cannot create the client):
#   1. Google Auth Platform > Branding / Audience: configure the consent screen as "External".
#      While it is in "Testing", add each user as a test user too (plain email, no "user:").
#   2. Security > Identity-Aware Proxy > the server service > Generate credentials. IAP creates a
#      client with its redirect URI registered.
#   3. Put the client ID and its enabled secret into iap_oauth_client_id / iap_oauth_client_secret
#      in terraform.tfvars.

resource "google_project_service_identity" "iap" {
  provider = google-beta

  service = "iap.googleapis.com"

  depends_on = [google_project_service.required]
}

data "google_project" "this" {}

# Set on the Cloud Run service itself, as the IAP docs do for Cloud Run.
resource "google_iap_settings" "server" {
  count = var.iap_oauth_client_id == null ? 0 : 1

  name = "projects/${data.google_project.this.number}/iap_web/cloud_run-${var.region}/services/${google_cloud_run_v2_service.server.name}"

  access_settings {
    oauth_settings {
      client_id     = var.iap_oauth_client_id
      client_secret = var.iap_oauth_client_secret
    }
  }

  depends_on = [google_project_service.required]
}

# IAP forwards authenticated requests to Cloud Run as its service agent.
resource "google_cloud_run_v2_service_iam_member" "server_iap_invoker" {
  count = var.iap_enabled ? 1 : 0

  name     = google_cloud_run_v2_service.server.name
  location = google_cloud_run_v2_service.server.location
  role     = "roles/run.invoker"
  member   = google_project_service_identity.iap.member
}

resource "google_iap_web_cloud_run_service_iam_member" "server_access" {
  for_each = var.iap_enabled ? toset(var.iap_members) : toset([])

  location               = google_cloud_run_v2_service.server.location
  cloud_run_service_name = google_cloud_run_v2_service.server.name
  role                   = "roles/iap.httpsResourceAccessor"
  member                 = each.value
}
