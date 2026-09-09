# Terraform Provider for Apple Ads

Manage [Apple Ads](https://ads.apple.com/) configuration as code with Terraform.

This is an open-source project from [Pi Cubed](https://github.com/picubedllc). It provides a Terraform Plugin Framework provider for Apple Ads campaigns, ad groups, and keywords — with safety guardrails that protect historical campaign identity.

**Status: early development, not yet published** to the Terraform Registry.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for building from source)

## Building the Provider

```shell
git clone https://github.com/picubedllc/terraform-provider-appleads
cd terraform-provider-appleads
make install
```

## Developing

```shell
make build    # compile
make test     # unit tests
make lint     # golangci-lint
make fmt      # gofmt
make generate # regenerate docs (tfplugindocs)
```

Acceptance tests (`make testacc`) require real Apple Ads credentials and are not run by default.
