# CoralMux Relay

NAT-transparent WebSocket relay for AI agents. Connect your phone to your own AI server without port forwarding.

```
📱 Phone ──WSS──▶ CoralMux Relay ◀──WSS── 🤖 Agent
   (outbound)         (cloud)         (outbound)
```

Both sides make **outbound** connections. Works behind any NAT/firewall.

## Works With

| Agent Frameworks | Mobile Clients |
|------------------|----------------|
| [OpenClaw](https://github.com/openclaw/openclaw) | Android / iOS |
| LangChain | React Native |
| AutoGPT | Flutter |
| CrewAI | Web |
| Your own bot | Any WebSocket client |

Protocol-based — if it speaks WebSocket + JSON, it works.

## Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/openclaw/coralmux/main/install.sh | sh
```

Or download from [Releases](https://github.com/openclaw/coralmux/releases).

## Features

- 🔐 **End-to-end encryption** — X25519 + AES-256-GCM (relay cannot read messages)
- 🌐 **Zero-config NAT traversal** — outbound WebSocket from both ends
- ⚡ **Streaming** — real-time token-by-token response delivery
- 📊 **Rate limiting** — per-connection + daily/monthly bandwidth quota
- 📎 **Multimodal** — image attachments up to 5MB
- 🔒 **Auto TLS** — Let's Encrypt integration
- 📦 **Single binary** — no dependencies

## Usage

### Development (no TLS)
```bash
coralmux-relay -addr :8080 -admin-key my-secret
```

### Production (auto TLS)
```bash
coralmux-relay -domain relay.example.com -admin-key $(openssl rand -hex 16)
```

### Create Pairing Token
```bash
curl -X POST http://localhost:8080/api/v1/pair \
  -H "X-Admin-Key: my-secret"

# Response: {"token": "oc_pair_a1b2c3d4..."}
```

Share this token with your phone app and agent to pair them.

## Architecture

```
┌──────────────┐         ┌─────────────────┐         ┌──────────────┐
│ 📱 Phone App  │──WSS──▶│  CoralMux Relay │◀──WSS──│ 🤖 AI Agent  │
│              │         │                 │         │              │
│  E2E encrypt │         │  Forward only   │         │  E2E decrypt │
│  ───────────▶│         │  (can't read)   │         │◀─────────────│
└──────────────┘         └─────────────────┘         └──────────────┘
```

The relay only sees encrypted blobs. It forwards messages but cannot decrypt them.

## E2E Encryption

When both peers connect:

1. Exchange X25519 public keys via `key_exchange` message
2. Derive shared secret using ECDH + HKDF-SHA256
3. Encrypt all chat messages with AES-256-GCM

```json
{
  "type": "chat.send",
  "payload": {
    "enc": true,
    "ciphertext": "base64...",
    "nonce": "base64..."
  }
}
```

See [PROTOCOL.md](PROTOCOL.md) for full protocol specification.

## Rate Limits

| Limit | Value |
|-------|-------|
| Max message size | 5 MB |
| Phone messages/min | 30 |
| Agent messages/min | 120 |
| Daily bandwidth/token | 500 MB |
| Monthly bandwidth/token | 10 GB |

## Build from Source

```bash
git clone https://github.com/openclaw/coralmux.git
cd coralmux
make build          # Native binary
make build-all      # All platforms (dist/)
```

## Deployment

### systemd
```bash
sudo cp coralmux-relay /usr/local/bin/
sudo cp deploy/systemd/coralmux-relay.service /etc/systemd/system/
sudo systemctl enable --now coralmux-relay
```

### Docker
```bash
docker run -p 443:443 ghcr.io/openclaw/coralmux \
  -domain relay.example.com -admin-key your-secret
```

### Oracle Cloud Free Tier (Recommended)
ARM Ampere A1 (4 cores, 24GB RAM) is **free forever**. Perfect for running your own relay.

## Public Relay

Don't want to host your own? Use the public relay:

```
wss://relay.coralmux.com/ws
```

Rate limits apply. For production use, we recommend self-hosting.

## License

MIT

---

Made with ☕ by [OpenClaw](https://github.com/openclaw)
