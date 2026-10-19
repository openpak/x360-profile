# Changelog

- 2026-10-19: v0.9.4: use the account-core avatar as the primary 64px gamer picture, with a 32px compatibility tile and durable opaque handles.

- 2026-10-19: v0.9.3: restore the first built-in dashboard picture through the gamer-picture
  setting. Xenon recovery does not run the new-account picture fallback.

- 2026-10-19: v0.9.2: serve HTTPS with XAM-compatible TLS 1.2 RSA suites; XAM omits Authorization
  for HTTP LiveHive URLs. Accept decimal XUID paths sent by the dashboard.

- 2026-10-18: v0.9.0 first release: empty-list replies for the resources the dashboard recovers after a sign-in,
  behind the XSTS token.
