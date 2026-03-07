# SP02PH01 Implementation Plan

## Overview
Phase 1: Backend Connection Manager - Implementing the core MUD proxy functionality.

## Architecture

### New Packages Required
```
internal/
  session/
    handler.go    - HTTP handlers for session REST API
    manager.go    - Connection manager (TCP dialer, session tracking)
    ipblock.go    - Private IP detection logic
    config.go     - Session-specific configuration
  store/
    connection.go - Saved connections DB operations
```

### Configuration (internal/config/config.go)
Add new environment variables:
- `MUD_PROXY_PORT_WHITELIST` - Comma-separated ports (default: "23")
- `IDLE_TIMEOUT_MINUTES` - Default: 30
- `HARD_SESSION_CAP_HOURS` - Default: 24
- `MAX_MESSAGE_SIZE_BYTES` - Default: 65536

### Redis Keys (internal/redis/keys.go)
Add new keys for MUD session management:
- `mud:session:{user_id}` - Active MUD session state
- `mud:session:{user_id}:timer` - Timer tracking

## Implementation Tasks

### SP02PH01T00: Create saved_connections table
- Create migration `003_create_saved_connections.up.sql`
- Create migration `003_create_saved_connections.down.sql`
- Table schema per SP02 spec section 3.3.5

### SP02PH01T01: TCP Dialer Implementation
- Create `internal/session/manager.go`
- Implement `DialMUD(host string, port int)` function
- Use `net.DialTimeout()` with 10-second default timeout
- Handle telnet IAC negotiation bytes (consume/will/wont)

### SP02PH01T02: Port Whitelist Enforcement
- Load whitelist from config
- Validate port before dial
- Return clear error for non-whitelisted ports

### SP02PH01T03: Private IP Detection
- Implement IP parsing and validation
- Block: 10.x.x.x, 172.16-31.x.x, 192.168.x.x, 127.x.x.x, localhost
- Check both resolved IP and hostname for private patterns

### SP02PH01T04: Per-User Session Binding
- Track active sessions in Redis
- Enforce one connection per user
- Store user_id with session metadata

### SP02PH01T05: Idle Timer Enforcement
- 30-minute idle timeout (configurable)
- Reset on any activity (inbound/outbound)
- Force disconnect on timeout

### SP02PH01T06: Hard Session Cap
- 24-hour absolute maximum (configurable)
- Track session start time
- Force disconnect on cap reached

### SP02PH01T07: Connection Metadata Logging
- Log: connect time, host, port, duration, disconnect reason
- No PII logged
- Use structured logging

### SP02PH01T08: Local Test Verification
- Test connection to Aardwolf (aardmud.org:23)
- Verify: connect, receive banner, send command, receive output, disconnect

## REST API Endpoints (PH01)

### POST /api/v1/session/connect
Request: `{ "host": "aardmud.org", "port": 23 }`
Response: `{ "state": "connecting", "session_id": "..." }`

### POST /api/v1/session/disconnect
Request: `{}`
Response: `{ "state": "disconnected", "reason": "user" }`

### GET /api/v1/session/status
Response: `{ "state": "connected", "connected_at": "...", "host": "...", "port": 23, "last_error": null, "disconnect_reason": null }`

## Acceptance Criteria

| Task | Criterion |
|------|-----------|
| SP02PH01T00 | Migration runs successfully, table created |
| SP02PH01T01 | Backend connects to aardmud.org:23 |
| SP02PH01T02 | Non-whitelisted ports rejected with error |
| SP02PH01T03 | Private IPs blocked with error |
| SP02PH01T04 | Only one session per user |
| SP02PH01T05 | 30-min idle → disconnect |
| SP02PH01T06 | 24-hour cap → disconnect |
| SP02PH01T07 | Metadata logged (no PII) |
| SP02PH01T08 | Full round-trip verified |

## Notes

- Saved connections table created but inert (no API/UI dependencies)
- REST endpoints implement control plane; WebSocket (data plane) in PH02
- Session state stored in Redis; saved connections in PostgreSQL
