# Look up an existing group by group name
data "storagegrid_group" "foo" {
  group_name = "foo-group"
}

data "storagegrid_group" "bar" {
  group_name = "bar-readonly"
}

# Output group information
output "foo_group_unique_name" {
  value = data.storagegrid_group.foo.unique_name
}

output "foo_group_policies" {
  value = data.storagegrid_group.foo.policies
}

output "bar_group_display_name" {
  value = data.storagegrid_group.bar.display_name
}
