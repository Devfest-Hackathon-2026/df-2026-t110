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

variable "db_deletion_protection" {
  description = "Prevents the Cloud SQL instance from being destroyed by Terraform or the API."
  type        = bool
  default     = true
}
