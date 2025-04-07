terraform {
  required_providers {
    certmgr = {
      source  = "barnes-c/certmgr"
      version = "1.0.0"
    }
  }
}

provider "certmgr" {
  host = "hector.cern.ch"
  port = 8008
}

resource "certmgr_certificate" "my_cert" {
  hostname = "myhostname.cern.ch"
}
