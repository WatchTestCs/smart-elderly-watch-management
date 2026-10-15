$ErrorActionPreference='Stop'
$project=Split-Path $PSScriptRoot -Parent
$output=Join-Path $project 'outputs/desktop-release-20261015'
$appDir=Join-Path $output 'release/Suining'
$installer=Join-Path $output 'release/岁宁-Setup-2.0.0.exe'
$manifest=Join-Path $output 'uninstall-files.nsh'
$lines=@()
Get-ChildItem -LiteralPath $appDir -File -Recurse | ForEach-Object {
  $relative=$_.FullName.Substring($appDir.Length+1).Replace('$','$$')
  $lines+='Delete "$INSTDIR\'+$relative+'"'
}
Get-ChildItem -LiteralPath $appDir -Directory -Recurse | Sort-Object { $_.FullName.Length } -Descending | ForEach-Object {
  $relative=$_.FullName.Substring($appDir.Length+1).Replace('$','$$')
  $lines+='RMDir "$INSTDIR\'+$relative+'"'
}
[IO.File]::WriteAllLines($manifest,$lines,[Text.UTF8Encoding]::new($true))
$compiler=Join-Path $output 'tools/nsis/nsis-3.12/makensis.exe'
& $compiler '/V3' '/INPUTCHARSET' 'UTF8' "/DAPPDIR=$appDir" "/DOUTPUT=$installer" "/DUNINSTALL_FILES=$manifest" "/DSIGN_SCRIPT=$PSScriptRoot/sign.ps1" "/DSIGN_SHELL=$((Get-Process -Id $PID).Path)" (Join-Path $PSScriptRoot 'installer.nsi')
if ($LASTEXITCODE -ne 0) { throw 'Installer compilation failed' }
& (Join-Path $PSScriptRoot 'sign.ps1') -FilePath $installer
Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert | Where-Object Subject -eq 'CN=Suining Local Build' | Select-Object -First 1 | Export-Certificate -FilePath (Join-Path $output 'release/Suining-Local-Signing.cer') | Out-Null
