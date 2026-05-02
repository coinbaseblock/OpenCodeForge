<#
.SYNOPSIS
  Pull recommended OpenCodeForge coding models into the Ollama container.

.PARAMETER Profile
  Which set of models to pull. One of: light, default, heavy, deepseek, all.

.EXAMPLE
  .\scripts\pull-models.ps1 default
  .\scripts\pull-models.ps1 all
#>
[CmdletBinding()]
param(
  [ValidateSet('light','default','heavy','deepseek','all')]
  [string]$Profile = 'default',
  [string]$Container = 'opencodeforge-ollama'
)

$ErrorActionPreference = 'Stop'

function Require-Container {
  param([string]$Name)
  $exists = docker inspect $Name 2>$null
  if (-not $exists) {
    Write-Error "container '$Name' not running. Run 'docker compose up -d' first."
  }
}

function Pull-Model {
  param([string]$Name)
  Write-Host "pulling $Name ..."
  docker exec -t $Container ollama pull $Name
  if ($LASTEXITCODE -ne 0) { throw "ollama pull $Name failed" }
}

Require-Container -Name $Container

switch ($Profile) {
  'light'    { Pull-Model 'qwen2.5-coder:7b' }
  'default'  { Pull-Model 'qwen2.5-coder:14b' }
  'heavy'    { Pull-Model 'qwen2.5-coder:32b' }
  'deepseek' { Pull-Model 'deepseek-coder-v2:lite' }
  'all' {
    Pull-Model 'qwen2.5-coder:7b'
    Pull-Model 'qwen2.5-coder:14b'
    Pull-Model 'qwen2.5-coder:32b'
    Pull-Model 'deepseek-coder-v2:lite'
  }
}

Write-Host "done."
docker exec -t $Container ollama list
