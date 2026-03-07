# Sync-RailwayEnvironment.ps1
# Syncs application environment variables from staging to production in Railway
# Usage: .\sync-railway-env.ps1
#
# IMPORTANT: This script syncs APPLICATION variables only. Infrastructure variables
# (DATABASE_URL, REDIS_URL) are NOT synced because they use Railway reference variables
# that are automatically resolved per-environment:
#   - ${{Postgres.DATABASE_URL}}
#   - ${{Redis.REDIS_URL}}

param(
    [string]$ProjectName = "mudpuppy",
    [string]$StagingEnv = "staging",
    [string]$ProductionEnv = "production"
)

$ErrorActionPreference = "Stop"

Write-Host "=== Railway Environment Sync ===" -ForegroundColor Cyan
Write-Host "Project: $ProjectName"
Write-Host "Source: $StagingEnv -> Target: $ProductionEnv"
Write-Host ""

# Variables to EXCLUDE from sync:
# - Railway system variables
# - DATABASE_URL and REDIS_URL (use Railway reference variables ${{Postgres.DATABASE_URL}} and ${{Redis.REDIS_URL}})
$excludeVars = @(
    "RAILWAY_ENVIRONMENT",
    "RAILWAY_ENVIRONMENT_ID", 
    "RAILWAY_ENVIRONMENT_NAME",
    "RAILWAY_PRIVATE_DOMAIN",
    "RAILWAY_PROJECT_ID",
    "RAILWAY_PROJECT_NAME",
    "RAILWAY_PUBLIC_DOMAIN",
    "RAILWAY_SERVICE_ID",
    "RAILWAY_SERVICE_MUDPUPPY_URL",
    "RAILWAY_SERVICE_NAME",
    "RAILWAY_STATIC_URL",
    "DATABASE_URL",      # Use ${{Postgres.DATABASE_URL}} instead
    "REDIS_URL"          # Use ${{Redis.REDIS_URL}} instead
)

try {
    # Switch to staging and get ALL variables
    Write-Host "[1/4] Getting ALL variables from staging environment..." -ForegroundColor Yellow
    railway environment link $StagingEnv
    railway service link MudPuppy
    
    # Get all staging variables as JSON object
    $stagingJson = railway vars --json 2>$null | ConvertFrom-Json
    
    # Get all properties (user-defined variables)
    $stagingVars = $stagingJson.PSObject.Properties | Where-Object { $excludeVars -notcontains $_.Name }
    
    Write-Host "  Found $($stagingVars.Count) user-defined variables in staging" -ForegroundColor Green
    foreach ($v in $stagingVars) {
        Write-Host "    - $($v.Name)" -ForegroundColor Gray
    }
    Write-Host "  NOTE: DATABASE_URL and REDIS_URL are NOT synced (use Railway reference variables)" -ForegroundColor Cyan
    
    # Switch to production
    Write-Host "[2/4] Switching to production environment..." -ForegroundColor Yellow
    railway environment link $ProductionEnv
    railway service link MudPuppy
    
    Write-Host "[3/4] Copying variables to production..." -ForegroundColor Yellow
    
    $copied = 0
    foreach ($var in $stagingVars) {
        $value = $var.Value
        Write-Host "  Setting $($var.Name)..." -NoNewline
        railway variable set "$($var.Name)=$value"
        Write-Host " OK" -ForegroundColor Green
        $copied++
    }
    
    Write-Host "[4/4] Sync complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Copied $copied variables from staging to production." -ForegroundColor Cyan
    Write-Host "DATABASE_URL and REDIS_URL preserved (using Railway reference variables)" -ForegroundColor Cyan
    
} catch {
    Write-Host "Error: $_" -ForegroundColor Red
    throw
}

Write-Host ""
Write-Host "Done!" -ForegroundColor Green
