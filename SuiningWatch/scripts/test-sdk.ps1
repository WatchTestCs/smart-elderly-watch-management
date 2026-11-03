# Tests metadata/hash rejection only. These synthetic fixtures never compile a HAP.
$ErrorActionPreference = 'Stop'
$fixture = Join-Path ([IO.Path]::GetTempPath()) ('suining-sdk-contract-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $fixture | Out-Null
Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'check-sdk.ps1') -Destination $fixture
$archive = Join-Path $fixture 'sdk.zip'
[IO.File]::WriteAllText($archive, 'synthetic SDK fixture', [Text.UTF8Encoding]::new($false))
$lock = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'sdk-lock.json') -Raw | ConvertFrom-Json
$lock.archiveBytes = (Get-Item -LiteralPath $archive).Length
$lock.archiveSha256 = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
$lock | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $fixture 'sdk-lock.json')
. (Join-Path $fixture 'check-sdk.ps1')
$sdkHome = Join-Path $fixture 'sdk'
$ets = Join-Path $sdkHome 'default/openharmony/ets'
New-Item -ItemType Directory -Path $ets -Force | Out-Null
$metadataPath = Join-Path $ets 'oh-uni-package.json'
function Assert-Rejected([scriptblock]$Action, [string]$Label) {
  $rejected = $false
  try { & $Action | Out-Null } catch { $rejected = $true }
  if (!$rejected) { throw "Expected SDK rejection: $Label" }
}
function Write-Metadata([string]$Api, [string]$Version, [string]$Release) {
  @{apiVersion=$Api; version=$Version; releaseType=$Release} | ConvertTo-Json | Set-Content -LiteralPath $metadataPath
}
Write-Metadata '24' '6.1.1.125' 'Release'
$metadata = Assert-SuiningSdk -SdkHome $sdkHome
if ($metadata.apiVersion -cne '24') { throw 'Valid metadata was not returned' }
Assert-SuiningSdkArchive -Archive $archive
Assert-SuiningSdkArchive -Archive $archive -ExpectedSha256 $lock.archiveSha256
Assert-Rejected { Assert-SuiningSdkArchive -Archive $archive -ExpectedSha256 ('a' * 64) } 'unreviewed checksum override'
Write-Metadata '26' '6.1.1.125' 'Release'
Assert-Rejected { Assert-SuiningSdk -SdkHome $sdkHome } 'wrong API'
Write-Metadata '24' '6.1.1.126' 'Release'
Assert-Rejected { Assert-SuiningSdk -SdkHome $sdkHome } 'wrong ETS compiler'
Write-Metadata '24' '6.1.1.125' 'Beta'
Assert-Rejected { Assert-SuiningSdk -SdkHome $sdkHome } 'wrong release type'
'invalid JSON' | Set-Content -LiteralPath $metadataPath
Assert-Rejected { Assert-SuiningSdk -SdkHome $sdkHome } 'malformed metadata'
Remove-Item -LiteralPath $metadataPath
Assert-Rejected { Assert-SuiningSdk -SdkHome $sdkHome } 'missing metadata'
# Preserve byte length so the second rejection verifies the hash, not only size.
[IO.File]::WriteAllText($archive, 'synthetic SDK fixturE', [Text.UTF8Encoding]::new($false))
Assert-Rejected { Assert-SuiningSdkArchive -Archive $archive } 'modified archive of same size'
[IO.File]::WriteAllText($archive, 'truncated', [Text.UTF8Encoding]::new($false))
Assert-Rejected { Assert-SuiningSdkArchive -Archive $archive } 'truncated archive'
$lock.schema = 2
$lock | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $fixture 'sdk-lock.json')
Assert-Rejected { Get-SuiningSdkLock } 'unknown lock schema'
Write-Output 'SDK metadata and archive contracts passed; no native compilation was performed.'
