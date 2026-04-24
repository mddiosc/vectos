VERSION ?= dev
DIST_DIR ?= dist

.PHONY: build release-snapshot release-layout clean

build:
	go build -o vectos ./cmd/vectos

release-snapshot:
	goreleaser release --snapshot --clean

release-layout:
	mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(DIST_DIR)/vectos-darwin-arm64 ./cmd/vectos
	@printf '%s\n' "Cross-platform release builds currently require platform-specific ONNX runtime handling; use GoReleaser or native per-platform CI for non-host targets."

clean:
	rm -rf $(DIST_DIR)
