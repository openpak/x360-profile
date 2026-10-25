# Protocol: x360-profile

The console reaches it through the LiveHive: `LiveServiceUrl.profile=https://profile.xboxlive.com` (x360-liveinfo) and OpenPak DNS
resolving `profile.xboxlive.com`. Requests carry `Authorization: XBL2.0 x=<token>` from [`x360-xsts`](../../x360-xsts); the token is
verified with the same signing key (`x360-common/xstoken`) and may read only its own xuid.

XAM sends Authorization only over HTTPS. Set `X360_SERVICE_TLS_ADDR`, `X360_SERVICE_TLS_CERT`, and
`X360_SERVICE_TLS_KEY` to enable the separate TLS listener (TLS 1.2, RSA AES-CBC). Use an exact-host
leaf signed by the x360-xuacs CA, so service 12 can validate it; Traefik must pass TLS through.
The XUID in console REST paths is decimal; hexadecimal diagnostic paths are also accepted.

Account avatars require `X360_PROFILE_ACCOUNT_CORE_URL`, `X360_PROFILE_ACCOUNT_CORE_TOKEN`
(in the uncommitted core.env), and writable `X360_PROFILE_STATE`. The core avatar is read
through Media.GetAvatar; it is never changed. Only opaque picture-handle mappings are stored locally.
Public PNG URLs `/global/t.<title>/tile/0/<tile>` use native game-picture keys and expose no
account credentials. New core avatar versions receive new handles. The LiveHive must advertise
`StoragePictureUriRoot=profile.xboxlive.com;StoragePictureUriPort=80` (bare host, separate port).
