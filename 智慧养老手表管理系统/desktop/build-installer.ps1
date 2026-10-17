param(
  [string]$OutputDir,
  [string]$AppDir,
  [string]$InstallerPath,
  [switch]$Sign
)

$ErrorActionPreference='Stop'
$project=Split-Path $PSScriptRoot -Parent
$metadata=Get-Content (Join-Path $PSScriptRoot 'version.json') -Raw | ConvertFrom-Json
$version=[string]$metadata.version
$channel=[string]$metadata.channel

if (-not $OutputDir) { $OutputDir=Join-Path $project ("outputs/v{0}-{1}" -f $version,$channel) }
$OutputDir=[IO.Path]::GetFullPath($OutputDir)
if (-not $AppDir) { $AppDir=Join-Path $OutputDir 'unsigned/Suining' }
$AppDir=[IO.Path]::GetFullPath($AppDir)

$installerDir=Join-Path $OutputDir 'installer'
$evidenceDir=Join-Path $OutputDir 'evidence'
New-Item -ItemType Directory -Force -Path $installerDir,$evidenceDir | Out-Null

if (-not $InstallerPath) { $InstallerPath=Join-Path $installerDir ("岁宁-Setup-{0}.exe" -f $version) }
$InstallerPath=[IO.Path]::GetFullPath($InstallerPath)

if (-not (Test-Path (Join-Path $AppDir 'Suining.exe'))) {
  throw "Application directory is not ready: $AppDir"
}

$manifest=Join-Path $evidenceDir 'uninstall-files.nsh'
$lines=@()
Get-ChildItem -LiteralPath $AppDir -File -Recurse | ForEach-Object {
  $relative=$_.FullName.Substring($AppDir.Length+1).Replace('$','$$')
  $lines+='Delete "$INSTDIR\'+$relative+'"'
}
Get-ChildItem -LiteralPath $AppDir -Directory -Recurse | Sort-Object { $_.FullName.Length } -Descending | ForEach-Object {
  $relative=$_.FullName.Substring($AppDir.Length+1).Replace('$','$$')
  $lines+='RMDir "$INSTDIR\'+$relative+'"'
}
[IO.File]::WriteAllLines($manifest,$lines,[Text.UTF8Encoding]::new($true))

$compiler=Join-Path $OutputDir 'tools/nsis/nsis-3.12/makensis.exe'
if (-not (Test-Path $compiler)) { throw "NSIS compiler not found: $compiler" }

$arguments=@(
  '/V3',
  '/INPUTCHARSET','UTF8',
  "/DAPPDIR=$AppDir",
  "/DOUTPUT=$InstallerPath",
  "/DUNINSTALL_FILES=$manifest",
  "/DAPP_VERSION=$version"
)

if ($Sign) {
  $arguments += "/DSIGN_SCRIPT=$PSScriptRoot/sign.ps1"
  $arguments += "/DSIGN_SHELL=$((Get-Process -Id $PID).Path)"
}

$arguments += (Join-Path $PSScriptRoot 'installer.nsi')
& $compiler @arguments
if ($LASTEXITCODE -ne 0) { throw 'Installer compilation failed' }

if ($Sign) {
  & (Join-Path $PSScriptRoot 'sign.ps1') -FilePath $InstallerPath
  $certificate=(Get-AuthenticodeSignature -LiteralPath $InstallerPath).SignerCertificate
  if (-not $certificate) { throw 'Installer signing certificate is missing.' }
  $certificateName=if ($env:SUINING_SIGNING_THUMBPRINT) { 'Suining-Signing.cer' } else { 'Suining-Local-Signing.cer' }
  Export-Certificate -Cert $certificate -FilePath (Join-Path $evidenceDir $certificateName) | Out-Null
}

Write-Host "Installer prepared: $InstallerPath"
