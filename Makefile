.PHONY: build build-dev clean release-dry-run

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

clean:
	rm -f vectos
