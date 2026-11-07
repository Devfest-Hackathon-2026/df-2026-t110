terraform {
  required_version = "~> 1.17"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 8.10"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.9"
    }
  }

  # The bucket depends on the project ID, so it is passed with `-backend-config=backend.hcl`
  # (see backend.hcl.example for the bootstrap steps).
  backend "gcs" {
    prefix = "infra"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
