# Deploy: x360-profile

Part of OpenPak's Xbox 360 stack: one repository and one image per service around the shared Go library
[`x360-common`](../../x360-common) (`replace` + committed `vendor/`; after changing it run `go mod vendor` here and commit).

## Configuration

`X360_PROFILE_ADDR` (default `:10042`), `X360_PROFILE_XSTS_KEY` (the token signing key file, read-only). See `deploy/`.

## Build and test

`go test -mod=vendor ./...`. Tag `vX.Y.Z` and CI publishes `ghcr.io/openpak/x360-profile:latest` (amd64 + arm64).
The response shapes come from the XAM analysis in the Wharf notes (xrl-schema-opcodes.md section 12). Licensed AGPL-3.0, see LICENSE.

## Account erasure

`DELETE /internal/accounts/{account_id}/data` with `Authorization: Bearer $ACCOUNT_ERASE_KEY` on a separate listener `ERASE_ADDR` (default `:9099`, container network only, never published; not served while the key is unset). Reply `{"service":"x360-profile","rows":N}`; one `account erased` log line per erasure.

Website step name: `x360-profile`.

Deletes the account's rows of `pictures` (opaque avatar handles). Avatar pixels live in the account core. Nothing else per account is stored. No event consumer.
