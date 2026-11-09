variable "project_id" {
  description = "Google Cloud project ID."
  type        = string
}

variable "region" {
  description = "Region for Cloud Run, Cloud SQL, and Artifact Registry."
  type        = string
  default     = "asia-northeast1"
}

variable "server_image" {
  description = "Container image for the server. Defaults to a public sample until the first image is pushed to Artifact Registry."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "server_max_instances" {
  description = "Upper bound of Cloud Run instances, capping cost and Cloud SQL connections."
  type        = number
  default     = 3
}

variable "iap_enabled" {
  description = "Restrict the server (frontend and API) to iap_members with IAP. Set to false to open it to everyone."
  type        = bool
  default     = true
}

variable "iap_members" {
  description = "IAM members (e.g. \"user:alice@example.com\") allowed through IAP. Kept out of the repository in terraform.tfvars."
  type        = list(string)
  default     = []
}

# The project has no organization, so IAP needs a custom OAuth client to admit outside accounts.
# It cannot be created with Terraform; see iap.tf.
variable "iap_oauth_client_id" {
  description = "Client ID of the OAuth client IAP uses to sign users in."
  type        = string
  default     = null
}

variable "iap_oauth_client_secret" {
  description = "Client secret of the OAuth client IAP uses to sign users in."
  type        = string
  default     = null
  sensitive   = true
}

variable "db_tier" {
  description = "Cloud SQL machine tier. Shared-core tiers keep idle cost low."
  type        = string
  default     = "db-f1-micro"
}

variable "db_running" {
  description = "Set to false to stop the Cloud SQL instance during idle periods; only storage is billed while stopped, and the server cannot reach the database."
  type        = bool
  default     = true
}

variable "db_deletion_protection" {
  description = "Prevents the Cloud SQL instance from being destroyed by Terraform or the API."
  type        = bool
  default     = true
}

variable "github_owner" {
  description = "GitHub organization that owns the repository."
  type        = string
  default     = "dvpico"
}

variable "github_repository" {
  description = "GitHub repository that builds and deploys the server."
  type        = string
  default     = "Programming-Learning-AR"
}

variable "github_deploy_ref" {
  description = "Git ref allowed to push images and deploy from GitHub Actions."
  type        = string
  default     = "refs/heads/main"
}

# The server cannot learn its own URL from Terraform without a cycle, so the redirect URL is given explicitly.
variable "google_login_client_id" {
  description = "Client ID of the OAuth client the app uses for Google sign-in. Leave null to keep sign-in disabled."
  type        = string
  default     = null
}

variable "google_login_client_secret" {
  description = "Client secret of the Google sign-in OAuth client."
  type        = string
  default     = null
  sensitive   = true

  validation {
    condition     = var.google_login_client_id == null || var.google_login_client_secret != null
    error_message = "google_login_client_secret is required when google_login_client_id is set."
  }
}

variable "google_login_redirect_url" {
  description = "Redirect URL registered on the Google sign-in client, e.g. https://<server host>/api/login/callback."
  type        = string
  default     = null

  validation {
    condition     = var.google_login_client_id == null || var.google_login_redirect_url != null
    error_message = "google_login_redirect_url is required when google_login_client_id is set."
  }
}
