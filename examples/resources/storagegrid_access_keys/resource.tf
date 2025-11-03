# Create access keys for a user with expiration
resource "storagegrid_access_keys" "developer_keys" {
  user_name    = storagegrid_user.developer.user_name
  created_date = "2024-01-01"
  expires      = "2025-12-31T23:59:59.000Z"
}

# Create permanent access keys (no expiration)
resource "storagegrid_access_keys" "service_account_keys" {
  user_name    = storagegrid_user.service_account.user_name
  created_date = "2024-01-01"
  # expires not specified = permanent keys
}

# Output the access key information (secret will be sensitive)
output "developer_access_key_id" {
  value = storagegrid_access_keys.developer_keys.access_key
}

output "developer_secret_key" {
  value     = storagegrid_access_keys.developer_keys.secret_access_key
  sensitive = true
}
