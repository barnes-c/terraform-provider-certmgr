# Terraform Provider CertMgr

This is a terraform provider for [certMgr](https://gitlab.cern.ch/ai-config-team/ai-tools/-/blob/master/aitools/certmgr.py?ref_type=heads). certMgr is the certificate management tool used at CERN to automate the handling of X.509 certificates—typically for machines and services.

For more information about certMgr see the [GitLab Repo](https://gitlab.cern.ch/ai-config-team/ai-tools/-/blob/master/aitools/certmgr.py?ref_type=heads)

## Provider usage

To use the provider you just have to declare a provider block:

```terraform
provider "certmgr" {
  host = "<YOUR-CERTMGR-SERVER>"
  port = 8008
}
```

To be able to use the Provider valid Kerberos tickets must also be present

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

```shell
make testacc
```
