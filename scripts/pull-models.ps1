<#
.SYNOPSIS
  Pull recommended OpenCodeForge coding models into the Ollama container.

.PARAMETER Profile
  Which set of models to pull. One of: ultralight, fast, light, default, heavy, deepseek, golang, python, all.

.PARAMETER DryRun
  Print what would be pulled without contacting Docker or downloading anything.

.EXAMPLE
  .\scripts\pull-models.ps1 default
  .\scripts\pull-models.ps1 all
  .\scripts\pull-models.ps1 golang -DryRun
#>
[CmdletBinding()]
param(
  [ValidateSet('ultralight','fast','light','default','heavy','deepseek','golang','python','all')]
  [string]$Profile = 'default',
  [string]$Container = 'opencodeforge-ollama',
  [switch]$DryRun
)

$ErrorActionPreference = 'Stop'

function Require-Container {
  param([string]$Name)
  if ($DryRun) { return }
  & docker inspect $Name *> $null
  if ($LASTEXITCODE -ne 0) {
    throw "container '$Name' not running. Run 'docker compose up -d' first."
  }
}

function Pull-Model {
  param([string]$Name)
  if ($DryRun) {
    Write-Host "dry-run would pull $Name"
    return
  }
  $listOutput = & docker exec $Container ollama list 2>&1 | Out-String
  if ($listOutput -match [regex]::Escape($Name)) {
    Write-Host "skip $Name (already installed)"
    return
  }
  Write-Host "pulling $Name ..."
  & docker exec $Container ollama pull $Name
  if ($LASTEXITCODE -ne 0) { throw "ollama pull $Name failed" }
}

Require-Container -Name $Container

switch ($Profile) {
  'ultralight' { Pull-Model 'qwen2.5-coder:1.5b' }
  'fast'     { Pull-Model 'qwen2.5-coder:3b' }
  'light'    { Pull-Model 'qwen2.5-coder:7b' }
  'default'  { Pull-Model 'qwen2.5-coder:14b' }
  'heavy'    { Pull-Model 'qwen2.5-coder:32b' }
  'deepseek' { Pull-Model 'deepseek-coder-v2:lite' }
  'golang' {
    Pull-Model 'qwen2.5-coder:3b'
    Pull-Model 'deepseek-coder-v2:lite'
  }
  'python' {
    Pull-Model 'qwen2.5-coder:7b'
    Pull-Model 'deepseek-coder-v2:lite'
  }
  'all' {
    Pull-Model 'qwen2.5-coder:1.5b'
    Pull-Model 'qwen2.5-coder:3b'
    Pull-Model 'qwen2.5-coder:7b'
    Pull-Model 'qwen2.5-coder:14b'
    Pull-Model 'qwen2.5-coder:32b'
    Pull-Model 'deepseek-coder-v2:lite'
  }
}

Write-Host "done."
if (-not $DryRun) {
  & docker exec $Container ollama list
}
