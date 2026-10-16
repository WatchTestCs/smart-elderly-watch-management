param(
  [string]$OutputDir,
  [string]$AppDir,
  [switch]$Sign
)

$ErrorActionPreference='Stop'
$project=Split-Path $PSScriptRoot -Parent
$metadata=Get-Content (Join-Path $PSScriptRoot 'version.json') -Raw | ConvertFrom-Json
if (-not $OutputDir) { $OutputDir=Join-Path $project ("outputs/v{0}-{1}" -f $metadata.version,$metadata.channel) }
$OutputDir=[IO.Path]::GetFullPath($OutputDir)
if (-not $AppDir) { $AppDir=Join-Path $OutputDir 'unsigned/Suining' }

& (Join-Path $PSScriptRoot 'build-installer.ps1') -OutputDir $OutputDir -AppDir $AppDir -Sign:$Sign
if (-not $?) { throw 'Installer packaging failed' }
