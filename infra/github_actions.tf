data "github_repository" "this" {
  name = var.github_repository
}

resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github"
  display_name              = "GitHub Actions"

  depends_on = [google_project_service.required]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub Actions OIDC"

  # repository_id (not the name) so a renamed or re-created repository cannot reuse the trust.
  attribute_mapping = {
    "google.subject"          = "assertion.sub"
    "attribute.repository_id" = "assertion.repository_id"
    "attribute.ref"           = "assertion.ref"
  }
  attribute_condition = "assertion.repository_id == \"${data.github_repository.this.repo_id}\" && assertion.ref == \"${var.github_deploy_ref}\""

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "github_deployer" {
  account_id   = "github-deployer"
  display_name = "GitHub Actions deployer"
}

resource "google_service_account_iam_member" "github_deployer_workload_identity" {
  service_account_id = google_service_account.github_deployer.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository_id/${data.github_repository.this.repo_id}"
}

resource "google_artifact_registry_repository_iam_member" "github_deployer_writer" {
  repository = google_artifact_registry_repository.server.name
  location   = google_artifact_registry_repository.server.location
  role       = "roles/artifactregistry.writer"
  member     = google_service_account.github_deployer.member
}

resource "google_cloud_run_v2_service_iam_member" "github_deployer_developer" {
  name     = google_cloud_run_v2_service.server.name
  location = google_cloud_run_v2_service.server.location
  role     = "roles/run.developer"
  member   = google_service_account.github_deployer.member
}

# Deploying a revision that runs as the server service account requires actAs on it.
resource "google_service_account_iam_member" "github_deployer_act_as_server" {
  service_account_id = google_service_account.server.name
  role               = "roles/iam.serviceAccountUser"
  member             = google_service_account.github_deployer.member
}

# Non-secret values the deploy workflow needs; managed here so they follow the infrastructure.
locals {
  github_actions_variables = {
    GCP_WORKLOAD_IDENTITY_PROVIDER = google_iam_workload_identity_pool_provider.github.name
    GCP_DEPLOYER_SERVICE_ACCOUNT   = google_service_account.github_deployer.email
    GCP_REGION                     = var.region
    SERVER_IMAGE_REPOSITORY        = local.server_image_repository
    SERVER_CLOUD_RUN_SERVICE       = google_cloud_run_v2_service.server.name
    CLIENT_BUCKET                  = google_storage_bucket.client.name
  }
}

resource "github_actions_variable" "deploy" {
  for_each = local.github_actions_variables

  repository    = data.github_repository.this.name
  variable_name = each.key
  value         = each.value
}
