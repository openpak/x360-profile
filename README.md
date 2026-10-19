# x360-profile

The Xbox 360 profile REST service (hive root `LiveServiceUrl.profile`): the user's profile settings the dashboard recovers after a sign-in. An empty settings list for now: a new profile has defaults only.

The console reaches it through the LiveHive: `LiveServiceUrl.profile=https://profile.xboxlive.com` (x360-liveinfo) and OpenPak DNS
resolving `profile.xboxlive.com`. Requests carry `Authorization: XBL2.0 x=<token>` from [`x360-xsts`](../x360-xsts); the token is
verified with the same signing key (`x360-common/xstoken`) and may read only its own xuid.

Part of OpenPak's Xbox 360 stack: one repository and one image per service around the shared Go library
[`x360-common`](../x360-common) (`replace` + committed `vendor/`; after changing it run `go mod vendor` here and commit).

## Configuration

`X360_PROFILE_ADDR` (default `:10042`), `X360_PROFILE_XSTS_KEY` (the token signing key file, read-only). See `deploy/`.

## Build and test

`go test -mod=vendor ./...`. Tag `vX.Y.Z` and CI publishes `ghcr.io/openpak/x360-profile:latest` (amd64 + arm64).
The response shapes come from the XAM analysis in the Wharf notes (xrl-schema-opcodes.md section 12). Licensed AGPL-3.0, see LICENSE.

XAM sends Authorization only over HTTPS. Set `X360_SERVICE_TLS_ADDR`, `X360_SERVICE_TLS_CERT`, and
`X360_SERVICE_TLS_KEY` to enable the separate TLS listener (TLS 1.2, RSA AES-CBC). Use an exact-host
leaf signed by the x360-xuacs CA, so service 12 can validate it; Traefik must pass TLS through.
The XUID in console REST paths is decimal; hexadecimal diagnostic paths are also accepted.
