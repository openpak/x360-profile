# Xbox 360 Profile Server

OpenPak's Xbox 360 Profile Server.

| | |
| --- | --- |
| Network | Xbox 360 |
| Role | Profile REST service: the user's profile settings the dashboard recovers after a sign-in |
| Depends on | [`x360-xsts`](https://github.com/openpak/x360-xsts) (tokens), [`x360-common`](https://github.com/openpak/x360-common) (shared Go library), account core (avatars, optional) |
| Image | `ghcr.io/openpak/x360-profile` |
| Licence | AGPL-3.0-only |

## What it serves

| Endpoint | What |
| --- | --- |
| Profile REST (`LiveServiceUrl.profile`, `profile.xboxlive.com`) | Profile settings the dashboard recovers after a sign-in: 64px gamer picture and the 32px tile XAM requires |
| Public picture URLs `/global/t.<title>/tile/0/<tile>` | Native game-picture keys; expose no account credentials |

## Run

```sh
X360_PROFILE_XSTS_KEY=/path/to/signing.key go run -mod=vendor ./cmd/x360-profile
```

## Docs

- [Protocol](docs/protocol.md)
- [Evidence](docs/evidence.md)
- [Deploy](docs/deploy.md)
- [Notes](docs/notes.md)

## Licence

AGPL-3.0-only. See [LICENSE](LICENSE).
