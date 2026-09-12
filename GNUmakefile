default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

# Read-only live tests against Apple Ads. Requires APPLEADS_* credentials.
testlive:
	APPLEADS_LIVE_TEST=1 go test -p 1 -v -timeout=15m ./internal/client ./internal/provider -run 'TestLive|TestAcc'

# Full acceptance suite (live + future Terraform resource tests).
testacc:
	TF_ACC=1 APPLEADS_LIVE_TEST=1 go test -v -cover -timeout 120m ./...

.PHONY: fmt lint test testlive testacc build install generate
