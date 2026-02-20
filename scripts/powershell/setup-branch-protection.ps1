# GitHub Branch Protection Setup Script
# Run this script with a GitHub personal access token to configure branch protection

# Usage:
# $env:GITHUB_TOKEN = "<YOUR_GITHUB_TOKEN>"
# .\setup-branch-protection.ps1 -RepoOwner "amaranth494" -RepoName "mudpuppy"

param(
    [Parameter(Mandatory=$true)]
    [string]$RepoOwner,
    
    [Parameter(Mandatory=$true)]
    [string]$RepoName,
    
    [string]$BranchName = "main",
    [string]$Token = $env:GITHUB_TOKEN
)

if (-not $Token) {
    Write-Error "GitHub token not provided. Set $env:GITHUB_TOKEN or pass -Token parameter"
    exit 1
}

$headers = @{
    "Authorization" = "Bearer $Token"
    "Accept" = "application/vnd.github+json"
    "X-GitHub-Api-Version" = "2022-11-28"
}

$url = "https://api.github.com/repos/$RepoOwner/$RepoName/branches/$BranchName/protection"

$body = @{
    required_status_checks = @{
        strict = $true
        contexts = @("CI")
    }
    required_reviews = @{
        dismiss_stale_reviews = $true
        require_code_owner_reviews = $false
        required_approving_review_count = 1
    }
    enforce_admins = $false
    allow_force_pushes = $false
    allow_deletions = $false
    require_conversation_resolution = $true
} | ConvertTo-Json

Write-Host "Configuring branch protection for $BranchName..."

try {
    $null = Invoke-RestMethod -Uri $url -Method PUT -Headers $headers -Body $body -ContentType "application/json"
    Write-Host "Branch protection enabled successfully!" -ForegroundColor Green
    Write-Host "- Require status checks to pass: Yes"
    Write-Host "- Require code reviews: Yes (1 approval)"
    Write-Host "- Include administrators: No"
} catch {
    Write-Error "Failed to configure branch protection: $_"
    exit 1
}

# Also protect 'master' branch if it exists
$masterUrl = "https://api.github.com/repos/$RepoOwner/$RepoName/branches/master/protection"
Write-Host "`nChecking for master branch..."

try {
    $masterCheck = Invoke-RestMethod -Uri "https://api.github.com/repos/$RepoOwner/$RepoName/branches/master" -Method GET -Headers $headers
    if ($masterCheck) {
        Write-Host "Master branch exists, applying protection..."
        $null = Invoke-RestMethod -Uri $masterUrl -Method PUT -Headers $headers -Body $body -ContentType "application/json"
        Write-Host "Master branch protection enabled!" -ForegroundColor Green
    }
} catch {
    Write-Host "No master branch found (using main only)" -ForegroundColor Yellow
}

Write-Host "`nBranch protection setup complete!"
