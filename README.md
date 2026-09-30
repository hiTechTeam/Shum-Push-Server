# Shum Push Server

A small APNs gateway for the [Shum](https://github.com/hiTechTeam/Shum-iOS) decentralized messenger.

The server stores only the mapping between a Shum identity and its APNs device tokens. Device registration and notification requests are signed by the sender's Shum identity.

## Data handling

- Every write request includes the sender's signed Shum contact card: display name, profile bio, and public keys. The card is used to verify the request signature and is not stored.
- A notification request also includes the recipient identity, an event identifier, and the event kind (`message`, `invitation` or `reaction`). While handling the request, the server therefore knows which identity notified which. This relationship is not stored or logged.
- The server never receives message text, attachments, photos, address books, or private keys.
- The client IP address is kept in process memory only for rate limiting and is never written to disk or logs.
- APNs payloads contain the sender's display name from the signed card as the title, a generic localized body, the event identifier, and the event kind. They never carry message content or the chosen reaction and never change the app icon badge. The name is cleaned of control and text-direction characters and cut to 64 characters.
- Device tokens are removed when the app deletes its profile or when APNs reports them as invalid.

## HTTP API

- `GET /healthz` checks the process.
- `GET /readyz` checks whether APNs credentials are configured.
- `GET /invite` serves a public contact invitation page.
- `POST /v1/devices` registers an APNs token.
- `DELETE /v1/devices` removes an APNs token.
- `POST /v1/notifications` sends a generic notification without message content.

Write requests use Ed25519 headers `X-Shum-Public-Key`, `X-Shum-Timestamp`, `X-Shum-Nonce`, and `X-Shum-Signature`. The signature covers the method, path, timestamp, nonce, and SHA-256 hash of the exact request body.

## Contact links

The iOS share action uses `https://<gateway-host>/invite#c4/<signed-profile>`.
The page opens the self-contained `shum://c4/<signed-profile>` link when the visitor
taps **Open in Shum**, and offers the existing TestFlight invitation for installation.
This works with the gateway's HTTPS hostname; a separate domain is not required.
It is a browser landing page, not an iOS Universal Link.

The public profile stays in the URL fragment and is not sent to the server in HTTP
requests. The page has no analytics or external scripts. Its script accepts only
bounded base64url data for supported invitation versions and never redirects to a supplied arbitrary URL.
The app verifies the profile signature and renders the avatar locally from its seed.
Legacy c2 locator, c and c3 invitations remain supported. The shared link and
the QR code on the iOS profile screen both use c4: they carry the signed profile,
keys, name and avatar seed, so the contact is added without any network lookup.
New c4 invitations require an iOS build with self-contained profile support.

Forward `GET /invite` through the HTTPS gateway to this service. The existing
`/{path+}` HTTP integration already covers it. Deploy the server before distributing
an iOS build that generates the new links. Previously shared `shum://` links do not
change automatically. Some embedded browsers may require opening the page in Safari.

## Configuration

Copy `deploy/server.env.example` to `/etc/shum-push/server.env`. APNs delivery uses separate topic-specific `.p8` keys for Sandbox and Production, along with their Key IDs, the Apple Developer Team ID, and the app bundle identifier.

The public endpoint should terminate HTTPS before forwarding traffic to this service, for example with a reverse proxy or a cloud API gateway.

## Development

```bash
go test ./...
node --test internal/api/invitation.test.cjs
go run ./cmd/shum-push-server
```

## License

MIT
