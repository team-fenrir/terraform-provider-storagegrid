# Create a user and assign to the foo group
resource "storagegrid_user" "foo" {
  full_name   = "Foo User"
  unique_name = "foo-user"
  member_of   = [storagegrid_group.foo.id]
  disable     = false
}

# Create a user and assign to the bar group  
resource "storagegrid_user" "bar" {
  full_name   = "Bar User"
  unique_name = "bar-user"
  member_of   = [storagegrid_group.bar.id]
  disable     = false
}

# Create a disabled user
resource "storagegrid_user" "baz" {
  full_name   = "Baz Disabled User"
  unique_name = "baz-user"
  member_of   = []
  disable     = true
}

# Reference the groups created in the group example
resource "storagegrid_group" "foo" {
  group_name = "foo-group"

  policies {
    management = {
      manage_all_containers     = true
      manage_endpoints          = false
      manage_own_s3_credentials = true
      root_access               = false
    }
  }
}

resource "storagegrid_group" "bar" {
  group_name = "bar-readonly"

  policies {
    management = {
      manage_all_containers     = false
      manage_endpoints          = false
      manage_own_s3_credentials = true
      root_access               = false
    }
  }
}