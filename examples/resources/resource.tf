# Copyright (c) HashiCorp, Inc.

resource "certmgr_certificate" "my_cert" {
  hostname = "myhostname.cern.ch"
}
