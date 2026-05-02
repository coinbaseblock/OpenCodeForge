<#
.SYNOPSIS
  Probe every OpenCodeForge service and print a one-line status for each.
#>
[CmdletBinding()]
param(
  [int]$OllamaPort = 11434,
  [int]$WebUIPort  = 3000,
  [int]$ToolsPort  = 8088
)

$ErrorActionPreference = 'Continue'

function Probe {
  param(
    [string]$Name,
    [string]$Url
  )
  try {
    $resp = Invoke-WebRequest -Uri $Url -TimeoutSec 5 -UseBasicParsing
    if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 400) {
      Write-Host ("{0,-12} ok ({1})" -f $Name, $Url) -ForegroundColor Green
    } else {
      Write-Host ("{0,-12} bad status {1} ({2})" -f $Name, $resp.StatusCode, $Url) -ForegroundColor Yellow
    }
  } catch {
    Write-Host ("{0,-12} down ({1})" -f $Name, $Url) -ForegroundColor Red
  }
}

Probe 'ollama'    "http://localhost:$OllamaPort/api/tags"
Probe 'open-webui' "http://localhost:$WebUIPort/"
Probe 'tools-api' "http://localhost:$ToolsPort/health"
