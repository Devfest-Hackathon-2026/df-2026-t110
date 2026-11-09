# Built client (SPA) files. The server mounts this bucket read-only and serves it on the same
# origin as the API, so IAP and session cookies cover both; the bucket itself stays private.
# Per-object Cache-Control is set by client-deploy.yml: hashed assets, including fonts,
# are immutable for one year; index.html and public files require revalidation.
# The server sets the same HTTP headers when serving through the Cloud Storage mount.
resource "google_storage_bucket" "client" {
  name     = "${var.project_id}-client"
  location = var.region

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
}

resource "google_storage_bucket_iam_member" "server_client_reader" {
  bucket = google_storage_bucket.client.name
  role   = "roles/storage.objectViewer"
  member = google_service_account.server.member
}

# objectUser (not objectCreator) because each deploy overwrites index.html.
resource "google_storage_bucket_iam_member" "github_deployer_client_writer" {
  bucket = google_storage_bucket.client.name
  role   = "roles/storage.objectUser"
  member = google_service_account.github_deployer.member
}

resource "google_storage_bucket_iam_member" "github_deployer_client_bucket_reader" {
  bucket = google_storage_bucket.client.name
  role   = "roles/storage.legacyBucketReader"
  member = google_service_account.github_deployer.member
}
