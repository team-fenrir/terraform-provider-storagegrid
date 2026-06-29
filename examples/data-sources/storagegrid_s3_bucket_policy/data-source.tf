# Look up the policy for the logs bucket
data "storagegrid_s3_bucket_policy" "logs" {
  bucket_name = "logs-bucket"
}

output "logs_bucket_policy" {
  value = data.storagegrid_s3_bucket_policy.logs.policy
}
