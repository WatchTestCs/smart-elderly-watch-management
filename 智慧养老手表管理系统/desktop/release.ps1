param(
  [string]$OutputDir,
  [switch]$SkipBuild
)

$ErrorActionPreference='Stop'
$project=Split-Path $PSScriptRoot -Parent
$metadata=Get-Content (Join-Path $PSScriptRoot 'version.json') -Raw | ConvertFrom-Json
$version=[string]$metadata.version
$channel=[string]$metadata.channel

if (-not $OutputDir) { $OutputDir=Join-Path $project ("outputs/v{0}-{1}" -f $version,$channel) }
$OutputDir=[IO.Path]::GetFullPath($OutputDir)

if (-not $SkipBuild) {
  & (Join-Path $PSScriptRoot 'build.ps1') -OutputDir $OutputDir
  if (-not $?) { throw 'Build stage failed' }
}

$unsigned=Join-Path $OutputDir 'unsigned/Suining'
$signed=Join-Path $OutputDir 'signed/Suining'
$installerDir=Join-Path $OutputDir 'installer'
$evidenceDir=Join-Path $OutputDir 'evidence'

if (-not (Test-Path (Join-Path $unsigned 'Suining.exe'))) {
  throw "Unsigned build is missing: $unsigned"
}

Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $signed
New-Item -ItemType Directory -Force -Path $signed,$installerDir,$evidenceDir | Out-Null
Get-ChildItem -LiteralPath $unsigned -Force | Copy-Item -Destination $signed -Recurse -Force

& (Join-Path $PSScriptRoot 'sign.ps1') -FilePath (Join-Path $signed 'Suining.exe')
if (-not $?) { throw 'Application signing failed' }

& (Join-Path $PSScriptRoot 'package.ps1') -OutputDir $OutputDir -AppDir $signed -Sign
if (-not $?) { throw 'Signed installer packaging failed' }

$portable=Join-Path $installerDir ("岁宁-Portable-{0}.zip" -f $version)
Remove-Item -Force -ErrorAction SilentlyContinue $portable
Compress-Archive -Path (Join-Path $signed '*') -DestinationPath $portable -CompressionLevel Optimal

$installer=Join-Path $installerDir ("岁宁-Setup-{0}.exe" -f $version)
$releaseFiles=@(
  (Join-Path $signed 'Suining.exe'),
  $installer,
  $portable
)

foreach ($file in $releaseFiles) {
  if (-not (Test-Path $file)) { throw "Release artifact missing: $file" }
}

$checksums=$releaseFiles | ForEach-Object {
  $hash=Get-FileHash -LiteralPath $_ -Algorithm SHA256
  "{0}  {1}" -f $hash.Hash.ToLowerInvariant(),(Split-Path $_ -Leaf)
}
[IO.File]::WriteAllLines((Join-Path $evidenceDir 'SHA256SUMS.txt'),$checksums,[Text.UTF8Encoding]::new($false))

$commit='unknown'
try {
  Push-Location $project
  $candidate=(git rev-parse HEAD 2>$null)
  if ($LASTEXITCODE -eq 0 -and $candidate) { $commit=$candidate.Trim() }
} finally {
  Pop-Location
}

$manifest=[ordered]@{
  product='岁宁'
  version=$version
  channel=$channel
  commit=$commit
  generatedAtUtc=(Get-Date).ToUniversalTime().ToString('o')
  artifacts=$releaseFiles | ForEach-Object {
    $hash=Get-FileHash -LiteralPath $_ -Algorithm SHA256
    [ordered]@{
      file=(Split-Path $_ -Leaf)
      sha256=$hash.Hash.ToLowerInvariant()
      size=(Get-Item -LiteralPath $_).Length
    }
  }
}
$manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $evidenceDir 'release-manifest.json') -Encoding utf8

Write-Host "Release candidate prepared in: $OutputDir"
Write-Host "Evidence: $evidenceDir"
