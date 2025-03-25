terraform {
  required_providers {
    certmgr = {
      source  = "gitlab.cern.ch/ai-config-team/certmgr"
      version = "0.1.0"
    }
  }
}

provider "certmgr" {
  hostname = "hector.cern.ch"
  port     = 8008
}
