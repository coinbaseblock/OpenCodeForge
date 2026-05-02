<#
.SYNOPSIS
  Build the `opencodeforge-coder` model inside the Ollama container from
  ollama/Modelfile.coder. Pulls the FROM base model first if it's missing.

.PARAMETER ModelName
  Name to give the resulting model. Default: opencodeforge-coder.

.PARAMETER ModelfilePath
  Path on the host to the Modelfile (used to detect the FROM base).
  Default: ollama\Modelfile.coder relative to the repo root.

.PARAMETER ModelfileInContainer
  Path inside the container where the Modelfile is mounted.
  Default: /modelfiles/Modelfile.coder.

.EXAMPLE
  .\scripts\build-coder.ps1
  .\scripts\build-coder.ps1 -ModelName my-coder
#>
[CmdletBinding()]
param(
  [string]$Container = 'opencodeforge-ollama',
  [string]$ModelName = 'opencodeforge-coder',
  [string]$ModelfilePath = (Join-Path $PSScriptRoot '..\ollama\Modelfile.coder'),
  [string]$ModelfileInContainer = '/modelfiles/Modelfile.coder'
)

$ErrorActionPreference = 'Stop'

& docker inspect $Container *> $null
if ($LASTEXITCODE -ne 0) {
  throw "container '$Container' not running. Run 'docker compose up -d' first."
}

if (-not (Test-Path $ModelfilePath)) {
  throw "Modelfile not found on host at: $ModelfilePath"
}

$fromLine = Select-String -Path $ModelfilePath -Pattern '^\s*FROM\s+(\S+)' | Select-Object -First 1
if (-not $fromLine) {
  throw "could not parse FROM line in $ModelfilePath"
}
$base = $fromLine.Matches[0].Groups[1].Value

$listOutput = & docker exec $Container ollama list 2>&1 | Out-String
if ($listOutput -notmatch [regex]::Escape($base)) {
  Write-Host "base model $base missing - pulling first ..."
  & docker exec $Container ollama pull $base
  if ($LASTEXITCODE -ne 0) { throw "ollama pull $base failed" }
}

Write-Host "creating $ModelName from $ModelfileInContainer (base: $base) ..."
& docker exec $Container ollama create $ModelName -f $ModelfileInContainer
if ($LASTEXITCODE -ne 0) { throw "ollama create $ModelName failed" }

Write-Host "done."
& docker exec $Container ollama list
