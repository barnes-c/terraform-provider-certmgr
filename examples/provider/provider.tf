terraform {
  required_providers {
    certmgr = {
      source  = "barnes-c/certmgr"
      version = "1.0.0"
    }
  }
}

provider "certmgr" {
  host = "<YOUR-CERTMGR-SERVER>"
  port = 8008
}
