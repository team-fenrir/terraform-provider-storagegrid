# Create a user and assign to a group
resource "storagegrid_user" "developer" {
  user_name = "john-doe"
  full_name = "John Doe"
  member_of = [storagegrid_group.developers.group_name]
  disable   = false
}

# Create a user with password assigned to multiple groups
resource "storagegrid_user" "admin" {
  user_name = "admin-user"
  full_name = "Admin User"
  member_of = [
    storagegrid_group.admin.group_name,
    storagegrid_group.developers.group_name
  ]
  password = "SecurePassword123!"
  disable  = false
}

# Create a disabled user
resource "storagegrid_user" "disabled" {
  user_name = "disabled-user"
  member_of = []
  disable   = true
}
