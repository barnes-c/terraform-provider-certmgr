# Copyright (c) HashiCorp, Inc.

terraform {
  required_providers {
    certmgr = {
      source  = "gitlab.cern.ch/ai-config-team/certmgr"
      version = "0.1.0"
    }
  }
}

provider "certmgr" {
  host = "hector.cern.ch"
  port     = 8008
}
