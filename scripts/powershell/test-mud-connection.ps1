# SP02PH01T08 - Local Test Verification Script
# Tests backend connecting to a real MUD server (aardmud.org:23)

# Prerequisites:
# 1. PostgreSQL running with DATABASE_URL set
# 2. Redis running with REDIS_URL set
# 3. Server compiled and running

$ErrorActionPreference = "Stop"

# Configuration
$MudHost = "aardmud.org"
$MudPort = 23

Write-Host "=== SP02PH01T08: MUD Connection Test ===" -ForegroundColor Cyan
Write-Host "Target: $MudHost`:$MudPort" 
Write-Host ""

# Check if server is running
Write-Host "[1/4] Checking if server is running..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method GET -TimeoutSec 5
    if ($health.status -eq "ok") {
        Write-Host "  ✓ Server is running" -ForegroundColor Green
    }
} catch {
    Write-Host "  ✗ Server not running. Start with: go run ./cmd/server/" -ForegroundColor Red
    exit 1
}

# Note: Full test requires authenticated session
# For now, this is a placeholder for manual testing

Write-Host ""
Write-Host "=== Manual Test Steps ===" -ForegroundColor Cyan
Write-Host @"
To complete SP02PH01T08 testing:

1. Start the server:
   go run ./cmd/server/

2. Authenticate (via browser or curl):
   - Register: POST /api/v1/register with {"email": "test@example.com"}
   - Get OTP from Redis (for development)
   - Login: POST /api/v1/login with {"email": "test@example.com", "otp": "XXXXXX"}
   - Note the session token from the response

3. Test connection (requires authenticated session):
   curl -X POST http://localhost:8080/api/v1/session/connect \
     -H "Content-Type: application/json" \
     -H "Cookie: session_token=<TOKEN>" \
     -d '{"host": "aardmud.org", "port": 23}'

4. Test status:
   curl http://localhost:8080/api/v1/session/status \
     -H "Cookie: session_token=<TOKEN>"

5. Test disconnect:
   curl -X POST http://localhost:8080/api/v1/session/disconnect \
     -H "Cookie: session_token=<TOKEN>"

Expected: Connect to aardmud.org:23, receive MUD banner, send command, receive output
"@

Write-Host ""
Write-Host "Test verification requires full auth flow. See above for manual steps." -ForegroundColor Yellow
