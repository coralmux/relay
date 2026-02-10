# CoralMux Protocol v1

A standard WebSocket protocol for AI agent communication through NAT-transparent relay.

## Overview

```
┌──────────────┐         ┌─────────────────┐         ┌──────────────┐
│   📱 Phone    │──WSS──▶│  CoralMux Relay │◀──WSS──│  🤖 Agent    │
│  (Android)   │         │                 │         │  (Python)    │
└──────────────┘         └─────────────────┘         └──────────────┘
      │                          │                          │
      │    Pairing Token         │                          │
      └──────────────────────────┴──────────────────────────┘
```

Both clients connect outbound to the relay. No port forwarding needed.

## Message Envelope

All messages use this JSON envelope:

```json
{
  "v": 1,
  "type": "chat.send",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "ts": 1707451200000,
  "payload": { ... }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `v` | int | Protocol version (currently 1) |
| `type` | string | Message type (namespace.action) |
| `id` | string | Unique message ID (UUID v4) |
| `ts` | int64 | Unix timestamp in milliseconds |
| `payload` | object | Type-specific payload |

## Namespaces

### Core
| Namespace | Description |
|-----------|-------------|
| `auth` | Authentication |
| `chat` | 1:1 conversation |
| `ping`/`pong` | Keepalive |
| `status` | Peer status |
| `key_exchange` | E2E encryption |

### Extended
| Namespace | Description |
|-----------|-------------|
| `agent` | Agent management |
| `group` | Group chat rooms |
| `system` | System monitoring |
| `memory` | Memory/knowledge search |
| `file` | File transfer |

## Message Types

### Authentication

#### `auth` (Client → Relay)
```json
{
  "type": "auth",
  "payload": {
    "token": "oc_pair_...",
    "role": "phone"  // or "agent"
  }
}
```

#### `auth.ok` (Relay → Client)
```json
{
  "type": "auth.ok",
  "payload": {
    "paired": true,
    "daily_quota_bytes": 524288000,
    "daily_used_bytes": 12345678
  }
}
```

#### `auth.fail` (Relay → Client)
```json
{
  "type": "auth.fail",
  "payload": {
    "code": "UNAUTHORIZED",
    "message": "Invalid pairing token"
  }
}
```

---

### Chat (1:1)

#### `chat.send` (Phone → Agent)
```json
{
  "type": "chat.send",
  "payload": {
    "text": "Hello, how are you?",
    "agent_id": "main",
    "attachments": [
      {
        "type": "image",
        "mime_type": "image/jpeg",
        "content_b64": "..."
      }
    ],
    "model": "claude-sonnet-4",
    "options": {}
  }
}
```

#### `chat.stream` (Agent → Phone)
```json
{
  "type": "chat.stream",
  "payload": {
    "delta": "Hello! I'm",
    "seq": 1
  }
}
```

#### `chat.done` (Agent → Phone)
```json
{
  "type": "chat.done",
  "payload": {
    "full_text": "Hello! I'm doing great, thanks for asking!",
    "attachments": [
      {
        "type": "image",
        "mime_type": "image/png",
        "url": "data:image/png;base64,..."
      }
    ]
  }
}
```

#### `chat.error` (Bidirectional)
```json
{
  "type": "chat.error",
  "payload": {
    "code": "BACKEND_ERROR",
    "message": "Failed to process message"
  }
}
```

#### `chat.tool_status` (Agent → Phone)
```json
{
  "type": "chat.tool_status",
  "payload": {
    "tool": "web_search",
    "status": "running"
  }
}
```

---

### Chat History

#### `chat.history` (Phone → Agent)
```json
{
  "type": "chat.history",
  "payload": {
    "agent_id": "main",
    "limit": 50
  }
}
```

#### `chat.history.result` (Agent → Phone)
```json
{
  "type": "chat.history.result",
  "payload": {
    "messages": [
      {
        "role": "user",
        "content": "Hello",
        "timestamp": 1707451200000
      },
      {
        "role": "assistant",
        "content": "Hi there!",
        "timestamp": 1707451201000
      }
    ]
  }
}
```

---

### Group Chat

#### `group.list` (Phone → Agent)
```json
{
  "type": "group.list",
  "payload": {}
}
```

#### `group.list.result` (Agent → Phone)
```json
{
  "type": "group.list.result",
  "payload": {
    "rooms": [
      {
        "id": "uuid",
        "name": "AI 비서 회의실",
        "participants": ["main", "hamon", "report"],
        "last_message": "안녕하세요",
        "last_message_at": 1707451200000
      }
    ]
  }
}
```

#### `group.messages` (Phone → Agent)
```json
{
  "type": "group.messages",
  "payload": {
    "room_id": "uuid",
    "limit": 50,
    "before": 1707451200000
  }
}
```

#### `group.messages.result` (Agent → Phone)
```json
{
  "type": "group.messages.result",
  "payload": {
    "room_id": "uuid",
    "messages": [
      {
        "id": "msg-uuid",
        "sender_id": "main",
        "sender_name": "아리아",
        "content": "안녕하세요!",
        "timestamp": 1707451200000
      }
    ]
  }
}
```

#### `group.send` (Phone → Agent)
```json
{
  "type": "group.send",
  "payload": {
    "room_id": "uuid",
    "text": "다들 안녕!"
  }
}
```

#### `group.message` (Agent → Phone, realtime)
```json
{
  "type": "group.message",
  "payload": {
    "room_id": "uuid",
    "message": {
      "id": "msg-uuid",
      "sender_id": "hamon",
      "sender_name": "하린",
      "content": "안녕하세요~",
      "timestamp": 1707451201000
    }
  }
}
```

---

### Agent Management

#### `agent.list` (Phone → Agent)
```json
{
  "type": "agent.list",
  "payload": {}
}
```

#### `agent.list.result` (Agent → Phone)
```json
{
  "type": "agent.list.result",
  "payload": {
    "agents": [
      {
        "id": "main",
        "name": "아리아",
        "description": "개인 AI 비서",
        "avatar_url": "https://..."
      }
    ]
  }
}
```

---

### System Monitoring

#### `system.status` (Phone → Agent)
```json
{
  "type": "system.status",
  "payload": {}
}
```

#### `system.status.result` (Agent → Phone)
```json
{
  "type": "system.status.result",
  "payload": {
    "cpu_percent": 12.5,
    "memory_percent": 45.2,
    "memory_used_gb": 16.3,
    "memory_total_gb": 36.0,
    "gpu_percent": 8.0,
    "uptime_seconds": 86400
  }
}
```

---

### Memory Search

#### `memory.search` (Phone → Agent)
```json
{
  "type": "memory.search",
  "payload": {
    "agent_id": "main",
    "query": "지난주 회의 내용",
    "limit": 20
  }
}
```

#### `memory.search.result` (Agent → Phone)
```json
{
  "type": "memory.search.result",
  "payload": {
    "results": [
      {
        "path": "memory/2026-02-01.md",
        "snippet": "## 회의 내용\n- MondrianChat 버그 수정...",
        "score": 0.85
      }
    ]
  }
}
```

---

### Keepalive

#### `ping` (Bidirectional)
```json
{
  "type": "ping",
  "payload": {}
}
```

#### `pong` (Bidirectional)
```json
{
  "type": "pong",
  "payload": {}
}
```

---

### Peer Status

#### `status` (Relay → Client)
```json
{
  "type": "status",
  "payload": {
    "peer": "online"  // or "offline"
  }
}
```

---

## Streaming Rules

1. **Sequence Numbers**: `seq` starts at 1, increments per chunk
2. **Completion**: `chat.done` signals end of stream with `full_text`
3. **Ordering**: Client should buffer and reorder by `seq` if needed
4. **Heartbeat**: Send `ping` every 30 seconds, expect `pong` within 10 seconds

## E2E Encryption

Optional X25519 + AES-256-GCM encryption:

1. Both sides exchange public keys via `key_exchange`
2. Derive shared secret using X25519 ECDH + HKDF-SHA256
3. Encrypt payloads with AES-256-GCM

Encrypted payload format:
```json
{
  "enc": true,
  "ciphertext": "base64...",
  "nonce": "base64..."
}
```

The relay cannot decrypt these payloads.

## Error Codes

| Code | Description |
|------|-------------|
| `UNAUTHORIZED` | Invalid or expired token |
| `PEER_OFFLINE` | Paired peer is not connected |
| `RATE_LIMITED` | Too many requests |
| `DAILY_QUOTA_EXCEEDED` | Daily bandwidth limit reached |
| `MONTHLY_QUOTA_EXCEEDED` | Monthly bandwidth limit reached |
| `MESSAGE_TOO_LARGE` | Message exceeds 5MB limit |
| `INVALID_MESSAGE` | Malformed message |
| `BACKEND_ERROR` | Agent backend error |

## Rate Limits

| Limit | Value |
|-------|-------|
| Max message size | 5 MB |
| Phone messages/min | 30 |
| Agent messages/min | 120 |
| Daily bandwidth/token | 500 MB |
| Monthly bandwidth/token | 10 GB |

---

*CoralMux Protocol v1 - Created 2026-02-09*
