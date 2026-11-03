function Get-SuiningSdkLock {
  $lock = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'sdk-lock.json') -Raw | ConvertFrom-Json
  if ($lock.schema -ne 1 -or $lock.archiveSha256 -cnotmatch '^[a-f0-9]{64}$' -or $lock.archiveBytes -le 0) {
    throw 'Invalid pinned watch SDK manifest'
  }
  return $lock
}

function Assert-SuiningSdk {
  param([Parameter(Mandatory)][string]$SdkHome)
  $lock = Get-SuiningSdkLock
  $metadataPath = Join-Path $SdkHome 'default/openharmony/ets/oh-uni-package.json'
  $metadata = Get-Content -LiteralPath $metadataPath -Raw -ErrorAction Stop | ConvertFrom-Json
  if ($metadata.apiVersion -cne $lock.sdkApiVersion -or $metadata.version -cne $lock.etsVersion -or $metadata.releaseType -cne 'Release') {
    throw "Watch SDK differs from the lock: expected API $($lock.sdkApiVersion), ETS $($lock.etsVersion) Release"
  }
  return $metadata
}

function Assert-SuiningSdkArchive {
  param([Parameter(Mandatory)][string]$Archive, [string]$ExpectedSha256 = '')
  $lock = Get-SuiningSdkLock
  if ($ExpectedSha256 -and $ExpectedSha256 -ine $lock.archiveSha256) {
    throw 'SDK checksum override differs from the reviewed lock; update the manifest through a reviewed PR'
  }
  if ((Get-Item -LiteralPath $Archive -ErrorAction Stop).Length -ne $lock.archiveBytes) {
    throw 'SDK archive size differs from the lock; it will not be extracted or executed'
  }
  if ((Get-FileHash -LiteralPath $Archive -Algorithm SHA256 -ErrorAction Stop).Hash -ine $lock.archiveSha256) {
    throw 'SDK checksum mismatch; it will not be extracted or executed'
  }
}
