# Create access keys for the foo user
resource "storagegrid_access_keys" "foo" {
  user_id      = storagegrid_user.foo.id
  created_date = "2024-01-01T00:00:00Z"
  expires      = "2025-12-31T23:59:59Z"
}

# Create access keys for the bar user
resource "storagegrid_access_keys" "bar" {
  user_id      = storagegrid_user.bar.id
  created_date = "2024-01-01T00:00:00Z"
  expires      = "2025-12-31T23:59:59Z"
}

# Access keys without expiration (permanent)
resource "storagegrid_access_keys" "baz" {
  user_id      = storagegrid_user.foo.id
  created_date = "2024-01-01T00:00:00Z"
  # expires not specified = permanent keys
}

# Reference the users created in the user example
resource "storagegrid_user" "foo" {
  full_name   = "Foo User"
  unique_name = "foo-user"
  member_of   = []
  disable     = false
}

resource "storagegrid_user" "bar" {
  full_name   = "Bar User"
  unique_name = "bar-user"
  member_of   = []
  disable     = false
}

# Output the access key information (secret will be sensitive)
output "foo_access_key_id" {
  value = storagegrid_access_keys.foo.access_key
}

output "foo_secret_key" {
  value     = storagegrid_access_keys.foo.secret_access_key
  sensitive = true
}