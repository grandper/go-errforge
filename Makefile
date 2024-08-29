PKG := github.com/grandper/go-errforge
PKGS := $(shell go list ./...)
SRCDIRS := $(shell go list -f '{{.Dir}}' $(PKGS))
GO := go

.PHONY: check test vet cover fmt gofmt lint build

# Run the full quality gate.
check: gofmt vet test

# Run the tests.
test:
	$(GO) test $(PKGS)

# Run the tests with the race detector and a coverage report.
cover:
	$(GO) test -race -coverprofile=coverage.out $(PKGS)
	$(GO) tool cover -func=coverage.out

# Report suspicious constructs.
vet:
	$(GO) vet $(PKGS)

# Rewrite the sources with gofmt.
fmt:
	gofmt -s -w $(SRCDIRS)

# Fail if any source file is not gofmt-formatted.
gofmt:
	@echo "Checking code is gofmted"
	@test -z "$(shell gofmt -s -l $(SRCDIRS) | tee /dev/stderr)"

# Run golangci-lint with the repository's configuration.
lint:
	golangci-lint run ./...

# Build every package.
build:
	$(GO) build $(PKGS)
