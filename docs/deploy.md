# Deploy: x360-profile

Part of OpenPak's Xbox 360 stack: one repository and one image per service around the shared Go library
[`x360-common`](../../x360-common) (`replace` + committed `vendor/`; after changing it run `go mod vendor` here and commit).

## Configuration

`X360_PROFILE_ADDR` (default `:10042`), `X360_PROFILE_XSTS_KEY` (the token signing key file, read-only). See `deploy/`.

## Build and test

`go test -mod=vendor ./...`. Tag `vX.Y.Z` and CI publishes `ghcr.io/openpak/x360-profile:latest` (amd64 + arm64).
The response shapes come from the XAM analysis in the Wharf notes (xrl-schema-opcodes.md section 12). Licensed AGPL-3.0, see LICENSE.
