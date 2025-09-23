# Look up an existing user by user name
data "storagegrid_user" "foo" {
  user_name = "foo-user"
}

# Look up another existing user by user name
data "storagegrid_user" "bar" {
  user_name = "bar-user"
}

# Output user information
output "foo_user_full_name" {
  value = data.storagegrid_user.foo.full_name
}

output "foo_user_groups" {
  value = data.storagegrid_user.foo.member_of
}

output "bar_user_disabled" {
  value = data.storagegrid_user.bar.disable
}
