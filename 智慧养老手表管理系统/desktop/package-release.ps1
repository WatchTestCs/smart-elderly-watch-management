$ErrorActionPreference = 'Stop'
& (Join-Path $PSScriptRoot 'validate-version.ps1')
$project = Split-Path $PSScriptRoot -Parent
$repo = Split-Path $project -Parent
$meta = Get-Content (Join-Path $PSScriptRoot 'version.json') -Raw | ConvertFrom-Json
$version = $meta.version
$out = Join-Path $project "outputs/v$version-$($meta.channel)"
$release = Join-Path $out 'release'
$portable = Join-Path $out 'portable'
$target = Join-Path $PSScriptRoot 'target'
if ($env:CARGO_TARGET_DIR) {
    $target = if ([IO.Path]::IsPathRooted($env:CARGO_TARGET_DIR)) { $env:CARGO_TARGET_DIR } else { Join-Path $PSScriptRoot $env:CARGO_TARGET_DIR }
}
New-Item -ItemType Directory -Force $release, $portable | Out-Null
$setup = Join-Path $out "Suining-Setup-$version-win-x64.exe"
Copy-Item -LiteralPath $setup -Destination $release
Copy-Item -LiteralPath (Join-Path $target 'release/suining.exe') -Destination $portable
Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'binaries/suining-server-x86_64-pc-windows-msvc.exe') -Destination (Join-Path $portable 'suining-server.exe')
Copy-Item -LiteralPath (Join-Path $PSScriptRoot '使用说明.txt') -Destination $portable
if ((Get-Item (Join-Path $portable 'suining.exe')).VersionInfo.ProductVersion -notlike "$version*") { throw 'Application version mismatch' }
$backendHash = (Get-FileHash (Join-Path $portable 'suining-server.exe')).Hash.ToLowerInvariant()
if ($backendHash -ne (Get-Content (Join-Path $PSScriptRoot 'backend-sha256.txt') -Raw).Trim()) { throw 'Sidecar hash mismatch' }
$portableZip = Join-Path $release "Suining-Portable-$version-win-x64.zip"
Compress-Archive -Path (Join-Path $portable '*') -DestinationPath $portableZip -Force
$commit = (git -C $repo rev-parse HEAD).Trim()
$sourceZip = Join-Path $release "Suining-Source-$version.zip"
git -C $repo archive --format=zip --prefix="Suining-$version/" --output=$sourceZip $commit
if ($LASTEXITCODE -ne 0) { throw 'Source archive failed' }
Copy-Item -LiteralPath (Join-Path $repo "docs/releases/$version.md") -Destination (Join-Path $release 'Release-Notes-zh-CN.md')

# Run against disposable directories only; no operator database or user profile.
$qaRoot = Join-Path ([IO.Path]::GetTempPath()) "suining-release-$([guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory $qaRoot | Out-Null
$env:SUINING_QA_HEADLESS = '1'
$env:SUINING_MYSQL_DSN = ''
$env:SUINING_PG_ANALYTICS_URL = ''
$env:MYSQL_DSN = ''
$env:PG_ANALYTICS_URL = ''
function Test-Application([string]$executable, [string]$label) {
    $env:SUINING_USER_DATA = Join-Path $qaRoot $label
    $application = Start-Process -FilePath $executable -PassThru -WindowStyle Hidden
    $backendPid = $null
    try {
        $discovery = Join-Path $env:SUINING_USER_DATA 'runtime/server.json'
        $deadline = [DateTime]::UtcNow.AddSeconds(45)
        while (!(Test-Path -LiteralPath $discovery)) {
            if ($application.HasExited) { throw "$label exited before ready" }
            if ([DateTime]::UtcNow -gt $deadline) { throw "$label readiness timed out" }
            Start-Sleep -Milliseconds 200
        }
        $server = Get-Content -LiteralPath $discovery -Raw | ConvertFrom-Json
        if ($server.desktopPid -ne $application.Id) { throw "$label desktop PID mismatch" }
        $backendPid = [int]$server.pid
        $page = Invoke-WebRequest "http://127.0.0.1:$($server.port)/index.html" -TimeoutSec 10
        if ($page.StatusCode -ne 200 -or $page.Content -notmatch '<html') { throw "$label embedded UI missing" }
        # Simulate parent termination and verify the sidecar's orphan watchdog.
        Stop-Process -Id $application.Id
        $application.WaitForExit()
        $deadline = [DateTime]::UtcNow.AddSeconds(20)
        while (Get-Process -Id $backendPid -ErrorAction SilentlyContinue) {
            if ([DateTime]::UtcNow -gt $deadline) { throw "$label orphan sidecar did not exit" }
            Start-Sleep -Milliseconds 200
        }
        return @{ distribution = $label; nativeStartup = $true; embeddedUI = $true; parentTerminationCleanup = $true }
    } finally {
        if (!$application.HasExited) { Stop-Process -Id $application.Id -ErrorAction SilentlyContinue }
        if ($backendPid) { Stop-Process -Id $backendPid -ErrorAction SilentlyContinue }
    }
}
$extracted = Join-Path $qaRoot 'portable-extracted'
Expand-Archive -LiteralPath $portableZip -DestinationPath $extracted
$checks = @(Test-Application (Join-Path $extracted 'suining.exe') 'portable')
$installDir = Join-Path $qaRoot 'installed'
$installer = Start-Process -FilePath $setup -ArgumentList @('/S', "/D=$installDir") -Wait -PassThru -WindowStyle Hidden
if ($installer.ExitCode -ne 0) { throw "Setup failed: $($installer.ExitCode)" }
try {
    $installedApp = Join-Path $installDir 'suining.exe'
    if ((Get-FileHash $installedApp).Hash -ne (Get-FileHash (Join-Path $portable 'suining.exe')).Hash) { throw 'Installed app differs from portable app' }
    $checks += Test-Application $installedApp 'installed'
    $sentinel = Join-Path $env:SUINING_USER_DATA 'preserve-on-uninstall.txt'
    Set-Content -LiteralPath $sentinel -Value 'test user data'
} finally {
    $uninstaller = Get-ChildItem -LiteralPath $installDir -Filter '*uninstall*.exe' | Select-Object -First 1
    if (!$uninstaller) { throw 'Uninstaller missing' }
    $uninstall = Start-Process -FilePath $uninstaller.FullName -ArgumentList @('/S', "_?=$installDir") -Wait -PassThru -WindowStyle Hidden
    if ($uninstall.ExitCode -ne 0) { throw "Uninstall failed: $($uninstall.ExitCode)" }
}
if (Test-Path -LiteralPath $installedApp) { throw 'Uninstall left application executable' }
if (!(Test-Path -LiteralPath $sentinel)) { throw 'Uninstall removed user data' }
$manifest = [ordered]@{
    version = $version; sourceCommit = $commit; branch = 'preview'; signing = 'unsigned'
    workflowRun = $env:GITHUB_RUN_ID; backendSHA256 = $backendHash
    checks = $checks; silentInstall = $true; uninstallPreservesData = $true
    limitations = @('Headless startup, not visual UI acceptance', 'No real devices or SMS gateway', 'WebView2 missing-runtime bootstrap not tested', 'No production migration or sustained HTTP load test')
}
$manifest | ConvertTo-Json -Depth 6 | Set-Content -Encoding utf8NoBOM (Join-Path $release 'release-manifest.json')
Get-ChildItem -LiteralPath $release -File | Where-Object Name -ne 'SHA256SUMS.txt' | Sort-Object Name | ForEach-Object {
    '{0}  {1}' -f (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant(), $_.Name
} | Set-Content -Encoding utf8NoBOM (Join-Path $release 'SHA256SUMS.txt')
Write-Host "Verified release files: $release (commit $commit)"
