resource "storagegrid_user" "example_resource" {
  user_name = "Test2"
  member_of = [
    "my-new_group2",
  ]
}

