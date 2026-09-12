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
make test     # unit tests (no Apple credentials)
make testlive # read-only live Apple Ads tests (APPLEADS_* env vars)
make lint     # golangci-lint
make fmt      # gofmt
make generate # regenerate docs (tfplugindocs)
```

Pull request CI runs unit tests, vet, lint, and docs generation only. It never receives Apple Ads credentials.

Live integration tests run from the `Apple Ads Live Integration Tests` workflow (`workflow_dispatch` or weekly). Credentials are GitHub Environment secrets on `appleads-integration`, not repository secrets, and that workflow is not triggered by pull requests.

Local live tests (`make testlive`) require:

```
APPLEADS_ORG_ID
APPLEADS_CLIENT_ID
APPLEADS_TEAM_ID
APPLEADS_KEY_ID
APPLEADS_PRIVATE_KEY
```
