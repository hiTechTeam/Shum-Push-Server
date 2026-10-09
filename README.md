# Shum Push Server

English · [Русский](README.ru.md)

APNs notification gateway for [Shum](https://github.com/hiTechTeam/Shum-iOS). Registers device tokens and sends generic notifications without message content.

## Data handling

The server stores Shum identity to APNs token mappings. Write requests include a signed public contact card, used to verify the request and not stored.

Notification requests expose sender and recipient identities, event ID and kind during processing. This relationship is not stored or logged. IP addresses stay in memory for rate limiting. Message text, attachments and private keys are never received.

APNs carries a sanitized sender name, a generic localized body, event ID and kind. It carries no message text or chosen reaction and does not change the app badge. Invalid tokens and tokens removed by the app are deleted.

## API

| Endpoint | Purpose |
| :--- | :--- |
| `GET /healthz` | Process health |
| `GET /readyz` | APNs configuration readiness |
| `GET /invite` | Public contact invitation page |
| `POST /v1/devices` | Register token |
| `DELETE /v1/devices` | Remove token |
| `POST /v1/notifications` | Request notification |

Writes use Ed25519 headers `X-Shum-Public-Key`, `X-Shum-Timestamp`, `X-Shum-Nonce` and `X-Shum-Signature`. The signature covers method, path, timestamp, nonce and SHA-256 of the exact body. See the [protocol](https://github.com/hiTechTeam/Shum-Protocol/blob/main/spec/08-push-api.md).

## Contact links

`https://<gateway-host>/invite#c4/<signed-profile>` opens a self-contained `shum://c4/<signed-profile>` invitation. The fragment stays in the browser; the page has no analytics or external scripts. It accepts bounded invitation data and does not redirect to arbitrary URLs.

c4 cards carry keys, name and avatar seed for offline addition in compatible clients. Legacy c2, c and c3 invitations remain supported. This is a browser page, not an iOS Universal Link. Forward `GET /invite` through the HTTPS gateway.

## Run

Copy [server.env.example](deploy/server.env.example) to `/etc/shum-push/server.env`. Configure separate APNs .p8 keys and Key IDs for Sandbox and Production, Team ID and bundle ID. Terminate public HTTPS at a proxy or gateway.

```sh
go test ./...
node --test internal/api/invitation.test.cjs
go run ./cmd/shum-push-server
```

## License

[MIT](LICENSE).
