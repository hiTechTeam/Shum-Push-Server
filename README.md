# Shum Push Server

A small APNs gateway for the [Shum](https://github.com/hiTechTeam/Shum-iOS) decentralized messenger.

The server stores only the mapping between a Shum identity and its APNs device tokens. Device registration and notification requests are signed by the sender's Shum identity.

## Data handling

- Every write request includes the sender's signed Shum contact card: display name, profile bio, and public keys. The card is used to verify the request signature and is not stored.
- A notification request also includes the recipient identity, an event identifier, and the event kind (`message` or `invitation`). While handling the request, the server therefore knows which identity notified which. This relationship is not stored or logged.
- The server never receives message text, attachments, photos, address books, or private keys.
- The client IP address is kept in process memory only for rate limiting and is never written to disk or logs.
- APNs payloads contain a generic localized alert, the event identifier, and the event kind, without message content.
- Device tokens are removed when the app deletes its profile or when APNs reports them as invalid.

## HTTP API

- `GET /healthz` checks the process.
- `GET /readyz` checks whether APNs credentials are configured.
- `POST /v1/devices` registers an APNs token.
- `DELETE /v1/devices` removes an APNs token.
- `POST /v1/notifications` sends a generic notification without message content.

Write requests use Ed25519 headers `X-Shum-Public-Key`, `X-Shum-Timestamp`, `X-Shum-Nonce`, and `X-Shum-Signature`. The signature covers the method, path, timestamp, nonce, and SHA-256 hash of the exact request body.

## Configuration

Copy `deploy/server.env.example` to `/etc/shum-push/server.env`. APNs delivery uses separate topic-specific `.p8` keys for Sandbox and Production, along with their Key IDs, the Apple Developer Team ID, and the app bundle identifier.

The public endpoint should terminate HTTPS before forwarding traffic to this service, for example with a reverse proxy or a cloud API gateway.

## Development

```bash
go test ./...
go run ./cmd/shum-push-server
```

## License

MIT
