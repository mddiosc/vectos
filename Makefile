.PHONY: build build-dev clean release-dry-run test lint check

# Development build (no version injection).
build:
	go build -o vectos ./cmd/vectos

# Build with version metadata injected from git.
# Usage: make build-dev  (uses current git describe + commit + date)
build-dev:
	go build \
	  -ldflags "-X vectos/internal/buildinfo.Version=$$(git describe --tags --always --dirty) \
	             -X vectos/internal/buildinfo.Commit=$$(git rev-parse --short HEAD) \
	             -X vectos/internal/buildinfo.Date=$$(date -u +%Y-%m-%d)" \
	  -o vectos ./cmd/vectos

# Dry-run of the GoReleaser pipeline (no publish).
# Requires goreleaser to be installed: https://goreleaser.com/install/
release-dry-run:
	goreleaser release --snapshot --clean

test:
	go test ./... -count=1

lint:
	go vet ./...
	@command -v staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed; skipping"

check: lint test

clean:
	rm -f vectos
