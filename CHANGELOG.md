# Changelog

- 2026-11-04: v0.11.0: Account erasure (card #120). `DELETE /internal/accounts/{account_id}/data` with `Authorization: Bearer $ACCOUNT_ERASE_KEY` on a separate listener `ERASE_ADDR` (default `:9099`, container network only, never published; not served while the key is unset). Reply `{"service":"x360-profile","rows":N}`; one `account erased` log line per erasure. Deletes the account's picture handles (table `pictures`).

- 2026-11-03: v0.10.2: No secrets in logs: the `request` line logs the path only, not the query string. Test added.

- 2026-11-03: v0.10.1: Every request logs one stderr line, `http <method> <path> -> <status> <duration>ms` (x360-common v0.5.1 `httplog`): no query, headers, cookies, tokens or bodies; `/healthz` skipped.

- 2026-11-03: v0.10.0: picture handles move to Postgres (schema `x360_profile`, table `pictures`, `X360_PROFILE_DATABASE_URL`,
  required with account media: the service refuses to start without it). Handle files in `X360_PROFILE_STATE` are imported
  once and left in place.

- 2026-10-19: v0.9.4: use the account-core avatar as the primary 64px gamer picture, with a 32px compatibility tile and durable opaque handles.

- 2026-10-19: v0.9.3: restore the first built-in dashboard picture through the gamer-picture
  setting. Xenon recovery does not run the new-account picture fallback.

- 2026-10-19: v0.9.2: serve HTTPS with XAM-compatible TLS 1.2 RSA suites; XAM omits Authorization
  for HTTP LiveHive URLs. Accept decimal XUID paths sent by the dashboard.

- 2026-10-18: v0.9.0 first release: empty-list replies for the resources the dashboard recovers after a sign-in,
  behind the XSTS token.
