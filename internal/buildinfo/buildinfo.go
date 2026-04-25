// Package buildinfo exposes release metadata injected at build time via ldflags.
//
// Use the following build flags to inject values:
//
//	go build \
//	  -ldflags "-X vectos/internal/buildinfo.Version=v0.1.0 \
//	             -X vectos/internal/buildinfo.Commit=abc1234 \
//	             -X vectos/internal/buildinfo.Date=2026-04-25" \
//	  ./cmd/vectos
//
// When building without those flags the sentinel values below are used so the
// binary still compiles and reports something useful during local development.
package buildinfo

// Version is the SemVer release tag (e.g. "v0.1.0").
// Injected at build time; defaults to "dev" for local builds.
var Version = "dev"

// Commit is the short git commit SHA of the build.
// Injected at build time; defaults to "unknown".
var Commit = "unknown"

// Date is the ISO-8601 build date (e.g. "2026-04-25").
// Injected at build time; defaults to "unknown".
var Date = "unknown"
