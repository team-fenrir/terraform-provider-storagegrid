resource "storagegrid_group" "example_resource" {
  group_name = "my-new_group2"
  policies = {
    s3 = file("policies/example.json")
    management = {
      manage_all_containers        = true,
      manage_endpoints             = true,
      manage_own_container_objects = true,
      manage_own_s3_credentials    = true,
      root_access                  = true,
      view_all_containers          = true
    }
  }
}
