<#
.SYNOPSIS
  Stop OpenCodeForge and remove its containers and volumes.

.DESCRIPTION
  This is destructive: it deletes Ollama's downloaded models and Open WebUI's
  user data. Workspace files on the host are NOT affected. The script prompts
  for confirmation unless -Force is supplied.

.PARAMETER Force
  Skip the confirmation prompt.

.EXAMPLE
  .\scripts\reset.ps1
  .\scripts\reset.ps1 -Force
#>
[CmdletBinding()]
param(
  [switch]$Force
)

$ErrorActionPreference = 'Stop'

if (-not $Force) {
  Write-Host "This will delete Ollama models and Open WebUI data volumes." -ForegroundColor Yellow
  $answer = Read-Host "Continue? (type 'yes' to confirm)"
  if ($answer -ne 'yes') {
    Write-Host "aborted."
    exit 1
  }
}

docker compose down -v
Write-Host "reset complete. Run 'docker compose up -d --build' to start fresh." -ForegroundColor Green
