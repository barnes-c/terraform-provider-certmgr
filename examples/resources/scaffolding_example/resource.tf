# Copyright (c) HashiCorp, Inc.

resource "certificate" "my_cert" {
  hostname = "myhostname.cern.ch"
}
