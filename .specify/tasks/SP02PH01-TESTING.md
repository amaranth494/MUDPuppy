# SP02PH01 Testing via Browser DevTools

## Test Steps

### 1. Login to get session cookie
1. Open your staging URL in the browser
2. Register/Login through the normal UI flow
3. The session cookie will be stored automatically

### 2. Open DevTools Console
- Press `F12` or `Ctrl+Shift+I` (Cmd+Option+I on Mac)
- Click on the "Console" tab

### 3. Test Connect to MUD

Run this in the console:

```javascript
// Test connecting to aardmud.org:23
fetch('/api/v1/session/connect', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ host: 'aardmud.org', port: 23 })
})
.then(r => r.json())
.then(console.log);
```

### 4. Test Status

```javascript
// Check connection status
fetch('/api/v1/session/status')
.then(r => r.json())
.then(console.log);
```

### 5. Test Disconnect

```javascript
// Disconnect from MUD
fetch('/api/v1/session/disconnect', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' }
})
.then(r => r.json())
.then(console.log);
```

### Test Error Cases

```javascript
// Test invalid port (should fail - only 23 allowed)
fetch('/api/v1/session/connect', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ host: 'aardmud.org', port: 22 })
})
.then(r => r.json())
.then(console.log);

// Test private IP (should fail)
fetch('/api/v1/session/connect', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ host: '192.168.1.1', port: 23 })
})
.then(r => r.json())
.then(console.log);

// Test localhost (should fail)
fetch('/api/v1/session/connect', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ host: 'localhost', port: 23 })
})
.then(r => r.json())
.then(console.log);
```

## Expected Results

| Test | Expected Response |
|------|------------------|
| Connect to aardmud.org:23 | `{ state: "connecting" or "connected" }` |
| Connect to port 22 | `{ error: "port 22 is not allowed..." }` |
| Connect to 192.168.x.x | `{ error: "private addresses not allowed" }` |
| Connect to localhost | `{ error: "localhost not allowed" }` |
| Status (connected) | `{ state: "connected", host: "...", port: 23 }` |
| Status (disconnected) | `{ state: "disconnected" }` |
