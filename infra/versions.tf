terraform {
  required_version = "~> 1.17"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 8.10"
    }
    github = {
      source  = "integrations/github"
      version = "~> 6.13"
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

# Authenticates with the GITHUB_TOKEN environment variable (e.g. `GITHUB_TOKEN=$(gh auth token)`),
# which needs admin access to the repository.
provider "github" {
  owner = var.github_owner
}
