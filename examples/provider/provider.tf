# Configure the StorageGrid Provider
terraform {
  required_providers {
    storagegrid = {
      source = "team-fenrir/storagegrid"
    }
  }
}

# Configure StorageGrid provider with management and S3 endpoints
provider "storagegrid" {
  endpoints {
    mgmt = "https://storagegrid.example.com:9443"
    s3   = "https://s3.storagegrid.example.com:10443" # Required for S3 operations
  }
  accountid = "12345678901234567890"
  username  = "admin"
  password  = "password"
}

# Alternative: Using environment variables
# export STORAGEGRID_ENDPOINT="https://storagegrid.example.com:9443"
# export STORAGEGRID_S3_ENDPOINT="https://s3.storagegrid.example.com:10443"
# export STORAGEGRID_ACCOUNTID="12345678901234567890"
# export STORAGEGRID_USERNAME="admin"
# export STORAGEGRID_PASSWORD="password"

provider "storagegrid" {
  # Configuration will be read from environment variables
}
