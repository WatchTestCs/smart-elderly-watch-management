param(
  [string]$OutputDir
)

$ErrorActionPreference='Stop'
$project=Split-Path $PSScriptRoot -Parent
$metadata=Get-Content (Join-Path $PSScriptRoot 'version.json') -Raw | ConvertFrom-Json
$version=[string]$metadata.version
$channel=[string]$metadata.channel
if (-not $OutputDir) { $OutputDir=Join-Path $project ("outputs/v{0}-{1}" -f $version,$channel) }
$OutputDir=[IO.Path]::GetFullPath($OutputDir)

$env:SUINING_OUTPUT_DIR=$OutputDir
$env:SUINING_VERSION=$version
$env:SUINING_CHANNEL=$channel

Push-Location $project
try {
  New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
  Remove-Item -Recurse -Force -ErrorAction SilentlyContinue (Join-Path $OutputDir 'app-staging')
  Remove-Item -Recurse -Force -ErrorAction SilentlyContinue (Join-Path $OutputDir 'unsigned/Suining')

  npm ci --prefix oldIot
  if ($LASTEXITCODE -ne 0) { throw 'Frontend dependency install failed' }

  npm --prefix oldIot run build
  if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed' }

  mvn -f careold/pom.xml '-Dnet.bytebuddy.experimental=true' clean verify
  if ($LASTEXITCODE -ne 0) { throw 'Backend tests/build failed' }

  npm ci --prefix desktop/build-tools
  if ($LASTEXITCODE -ne 0) { throw 'Packaging tools install failed' }

  python desktop/download-runtimes.py
  if ($LASTEXITCODE -ne 0) { throw 'Runtime download/check failed' }

  node desktop/prepare-package.cjs
  if ($LASTEXITCODE -ne 0) { throw 'Desktop package failed' }

  python desktop/prepare-notices.py
  if ($LASTEXITCODE -ne 0) { throw 'License collection failed' }

  Write-Host "Unsigned application prepared: $(Join-Path $OutputDir 'unsigned/Suining')"
  Write-Host "Next: ./desktop/package.ps1 -OutputDir '$OutputDir'"
  Write-Host "Release: ./desktop/release.ps1 -OutputDir '$OutputDir' -SkipBuild"
} finally {
  Pop-Location
}
