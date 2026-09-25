# Shum Push Server

A small APNs gateway for the [Shum](https://github.com/hiTechTeam/Shum-iOS) decentralized messenger.

The server stores the mapping between a Shum identity and its APNs device token. It never receives message text, attachments, contact names, or encryption keys. Device registration and notification requests are signed by the sender's Shum identity.

## HTTP API

- `GET /healthz` checks the process.
- `GET /readyz` checks whether APNs credentials are configured.
- `POST /v1/devices` registers an APNs token.
- `DELETE /v1/devices` removes an APNs token.
- `POST /v1/notifications` sends a generic notification without message content.

Write requests use Ed25519 headers `X-Shum-Public-Key`, `X-Shum-Timestamp`, `X-Shum-Nonce`, and `X-Shum-Signature`. The signature covers the method, path, timestamp, nonce, and SHA-256 hash of the exact request body.

## Configuration

Copy `deploy/server.env.example` to `/etc/shum-push/server.env`. APNs delivery uses separate topic-specific `.p8` keys for Sandbox and Production, along with their Key IDs, the Apple Developer Team ID, and the app bundle identifier.

The public endpoint should terminate HTTPS before forwarding traffic to this service. A Yandex API Gateway service domain can be used without buying a custom domain.

## Development

```bash
go test ./...
go run ./cmd/shum-push-server
```

## License

MIT
