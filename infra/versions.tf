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

  # State is kept locally for now; move to a GCS backend once the team shares applies.
  backend "local" {}
}

provider "google" {
  project = var.project_id
  region  = var.region
}
