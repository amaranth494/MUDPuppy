<#
.SYNOPSIS
    Starts frontend and backend development servers in independent terminal sessions.
.DESCRIPTION
    This script launches the frontend and backend development servers in separate
    terminal windows, allowing them to run concurrently without blocking.
.PARAMETER BackendPort
    The port for the backend server (default: 8080)
.PARAMETER FrontendPort
    The port for the frontend dev server (default: 3000)
.EXAMPLE
    .\start-dev-servers.ps1
.EXAMPLE
    .\start-dev-servers.ps1 -BackendPort 9000 -FrontendPort 8080
#>

param(
    [int]$BackendPort = 8080,
    [int]$FrontendPort = 3000
)

$ErrorActionPreference = "Stop"

# Get the project root directory (parent of scripts folder)
$ProjectRoot = Split-Path -Parent $PSScriptRoot

Write-Host "Starting MUDPuppy development servers..." -ForegroundColor Cyan
Write-Host "Backend port: $BackendPort" -ForegroundColor Yellow
Write-Host "Frontend port: $FrontendPort" -ForegroundColor Yellow

# Start Backend Server
$BackendPath = Join-Path $ProjectRoot "backend"
if (Test-Path $BackendPath) {
    Write-Host "`nStarting backend server..." -ForegroundColor Green
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$BackendPath'; Write-Host 'Backend running on port $BackendPort' -ForegroundColor Cyan; go run main.go" -WindowStyle Normal
} else {
    Write-Host "Backend directory not found at: $BackendPath" -ForegroundColor Red
}

# Start Frontend Server
$FrontendPath = Join-Path $ProjectRoot "frontend"
if (Test-Path $FrontendPath) {
    Write-Host "Starting frontend server..." -ForegroundColor Green
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$FrontendPath'; Write-Host 'Frontend running on port $FrontendPort' -ForegroundColor Cyan; npm run dev" -WindowStyle Normal
} else {
    Write-Host "Frontend directory not found at: $FrontendPath" -ForegroundColor Red
}

Write-Host "`nBoth servers started in separate terminal windows." -ForegroundColor Cyan
Write-Host "Backend: http://localhost:$BackendPort" -ForegroundColor White
Write-Host "Frontend: http://localhost:$FrontendPort" -ForegroundColor White
