# Google sign-in of the app itself, separate from the IAP client in iap.tf.
#
# One-time manual setup (OAuth clients cannot be created with Terraform):
#   1. Google Auth Platform > Clients: create a "Web application" client and add
#      google_login_redirect_url (https://<server host>/api/login/callback) as an authorized redirect URI.
#   2. Put the client ID, its secret and the redirect URL into google_login_* in terraform.tfvars.

locals {
  google_login_enabled = var.google_login_client_id != null
}

resource "google_secret_manager_secret" "google_login_client_secret" {
  count = local.google_login_enabled ? 1 : 0

  secret_id = "google-login-client-secret"

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "google_login_client_secret" {
  count = local.google_login_enabled ? 1 : 0

  secret      = google_secret_manager_secret.google_login_client_secret[0].id
  secret_data = var.google_login_client_secret
}
