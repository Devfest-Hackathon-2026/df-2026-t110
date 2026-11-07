output "server_url" {
  description = "Public URL of the server."
  value       = google_cloud_run_v2_service.server.uri
}

output "server_image_repository" {
  description = "Artifact Registry path to push server images to."
  value       = local.server_image_repository
}

output "db_connection_name" {
  description = "Cloud SQL connection name, for the Cloud SQL Auth Proxy."
  value       = google_sql_database_instance.main.connection_name
}
