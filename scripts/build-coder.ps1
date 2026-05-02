<#
.SYNOPSIS
  Build the `opencodeforge-coder` model inside the Ollama container from
  ollama/Modelfile.coder. Pulls the FROM base model first if it's missing.

.PARAMETER ModelName
  Name to give the resulting model. Default: opencodeforge-coder.

.PARAMETER Modelfile
  Path inside the container to the Modelfile. Default: /modelfiles/Modelfile.coder.

.EXAMPLE
  .\scripts\build-coder.ps1
  .\scripts\build-coder.ps1 -ModelName my-coder
#>
[CmdletBinding()]
param(
  [string]$Container = 'opencodeforge-ollama',
  [string]$ModelName = 'opencodeforge-coder',
  [string]$Modelfile = '/modelfiles/Modelfile.coder'
)

$ErrorActionPreference = 'Stop'

$exists = docker inspect $Container 2>$null
if (-not $exists) {
  Write-Error "container '$Container' not running. Run 'docker compose up -d' first."
}

$base = (docker exec -t $Container sh -c "grep -E '^FROM ' '$Modelfile' | awk '{print `$2}'") -replace '\s', ''
if (-not $base) {
  Write-Error "could not parse FROM line in $Modelfile"
}

$installed = docker exec -t $Container ollama list | Select-String -SimpleMatch "$base"
if (-not $installed) {
  Write-Host "base model $base missing - pulling first ..."
  docker exec -t $Container ollama pull $base
  if ($LASTEXITCODE -ne 0) { throw "ollama pull $base failed" }
}

Write-Host "creating $ModelName from $Modelfile (base: $base) ..."
docker exec -t $Container ollama create $ModelName -f $Modelfile
if ($LASTEXITCODE -ne 0) { throw "ollama create $ModelName failed" }

Write-Host "done."
docker exec -t $Container ollama list
