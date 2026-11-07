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
