
resource "storagegrid_access_keys" "example" {
  user_name = "<my-username>"
  # This field is mandatory. It is a free string field that can be used to trigger new access_keys when changed
  created_date = "2025-07-21"
  # OPTIONAL
  expires = "2028-09-04T00:00:00.000Z"
}
