param(
  [string]$DevEcoHome = $env:DEVECO_STUDIO_HOME,
  [string]$CommandLineHome = $env:HARMONY_COMMANDLINE_HOME,
  [string]$SdkHome = $env:DEVECO_SDK_HOME,
  [string]$ReportPath = ''
)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
if (!$ReportPath) { $ReportPath = Join-Path $projectRoot 'artifacts\native-build.json' }
$ReportPath = [IO.Path]::GetFullPath($ReportPath)
New-Item -ItemType Directory -Path (Split-Path $ReportPath -Parent) -Force | Out-Null
$report = [ordered]@{ status = 'blocked'; source = ''; dirty = $false; toolchain = ''; toolchainVersion = ''; sdk = ''; sdkApi = ''; etsVersion = ''; artifact = $null; error = ''; startedAt = [DateTime]::UtcNow.ToString('o') }
$savedEnvironment = @{}
foreach ($name in @('NODE_PATH', 'DEVECO_SDK_HOME', 'JAVA_HOME', 'PATH')) {
  $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
Push-Location $projectRoot
try {
  . (Join-Path $PSScriptRoot 'check-sdk.ps1')
  $sdkLock = Get-SuiningSdkLock
  $report.source = (& git rev-parse HEAD).Trim()
  $report.dirty = [bool](& git status --porcelain -- .)
  if ($CommandLineHome -and $DevEcoHome) { throw '请仅指定 CommandLineHome 或 DevEcoHome，避免混用工具链' }
  $prefix = @()
  if ($CommandLineHome) {
    $toolRoot = (Resolve-Path -LiteralPath $CommandLineHome).Path
    # The official CLI distribution owns its Node, Java and plugin resolution.
    $runner = @('bin\hvigorw.bat', 'bin\hvigorw.cmd') | ForEach-Object { Join-Path $toolRoot $_ } | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -First 1
    if (!$runner) { throw "未找到 Command Line Tools 的 bin\hvigorw.bat/cmd：$toolRoot" }
  } else {
    if (!$DevEcoHome) { $DevEcoHome = 'D:\Huawei\DevEco Studio' }
    $toolRoot = [IO.Path]::GetFullPath($DevEcoHome)
    $productInfo = Get-Content -LiteralPath (Join-Path $toolRoot 'product-info.json') -Raw | ConvertFrom-Json
    if ($productInfo.version -cne $sdkLock.toolchainVersion) { throw "DevEco version differs from the lock: $($sdkLock.toolchainVersion) required" }
    $report.toolchainVersion = $productInfo.version
    $runner = Join-Path $toolRoot 'tools\node\node.exe'
    $hvigor = Join-Path $toolRoot 'tools\hvigor\hvigor\bin\hvigor.js'
    $java = Join-Path $toolRoot 'jbr'
    foreach ($required in @($runner, $hvigor, (Join-Path $java 'bin\java.exe'))) {
      if (!(Test-Path -LiteralPath $required -PathType Leaf)) { throw "缺少原生构建工具：$required；请指定 DevEcoHome 或 CommandLineHome" }
    }
    $moduleRoot = Join-Path $projectRoot 'node_modules\@ohos'
    foreach ($package in @('hvigor', 'hvigor-ohos-plugin')) {
      $target = Join-Path $toolRoot "tools\hvigor\$package"
      if (!(Test-Path -LiteralPath $target -PathType Container)) { throw "缺少 bundled 插件：$target" }
    }
    New-Item -ItemType Directory -Path $moduleRoot -Force | Out-Null
    foreach ($package in @('hvigor', 'hvigor-ohos-plugin')) {
      $link = Join-Path $moduleRoot $package
      $target = Join-Path $toolRoot "tools\hvigor\$package"
      if (Test-Path -LiteralPath $link) {
        $item = Get-Item -LiteralPath $link
        if (!$item.Target -or [IO.Path]::GetFullPath([string](@($item.Target)[0])) -ne [IO.Path]::GetFullPath($target)) { throw "现有插件与指定工具链不一致：$link；脚本不会覆盖它" }
      } else { New-Item -ItemType Junction -Path $link -Target $target | Out-Null }
    }
    $prefix = @($hvigor)
    $env:NODE_PATH = Join-Path $projectRoot 'node_modules'
    $env:JAVA_HOME = $java
    $env:PATH = (Split-Path $runner -Parent) + [IO.Path]::PathSeparator + $env:PATH
  }
  if (!$SdkHome) { $SdkHome = Join-Path $toolRoot 'sdk' }
  if (!(Test-Path -LiteralPath (Join-Path $SdkHome 'default\openharmony\ets') -PathType Container)) { throw "缺少 SDK ETS 编译组件：$SdkHome" }
  $env:DEVECO_SDK_HOME = [IO.Path]::GetFullPath($SdkHome)
  $metadata = Assert-SuiningSdk -SdkHome $env:DEVECO_SDK_HOME
  $report.sdkApi = $metadata.apiVersion
  $report.etsVersion = $metadata.version
  $report.toolchain = $toolRoot
  $report.sdk = $env:DEVECO_SDK_HOME
  $report.status = 'failed'
  # A clean SDK build prevents an old HAP from satisfying this run's gate.
  foreach ($task in @('clean', 'assembleHap')) {
    & $runner @prefix $task --mode module -p module=entry@default -p product=default -p buildMode=debug --no-daemon 2>&1 | Tee-Object -FilePath ($ReportPath + '.log') -Append
    if ($LASTEXITCODE -ne 0) { throw "Hvigor $task 失败（$LASTEXITCODE）" }
  }
  $hap = Join-Path $projectRoot 'entry\build\default\outputs\default\entry-default-unsigned.hap'
  if (!(Test-Path -LiteralPath $hap)) { throw '编译未生成预期 HAP' }
  $report.artifact = @{ path = $hap; sha256 = (Get-FileHash -LiteralPath $hap -Algorithm SHA256).Hash; bytes = (Get-Item -LiteralPath $hap).Length; signed = $false }
  $report.status = 'passed'
  Write-Output "原生未签名调试包：$hap（不代表已安装或通过传感器验证）"
} catch {
  $report.error = $_.Exception.Message
  throw
} finally {
  foreach ($name in $savedEnvironment.Keys) {
    if ($null -eq $savedEnvironment[$name]) { Remove-Item -LiteralPath "Env:\$name" -ErrorAction SilentlyContinue }
    else { [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process') }
  }
  $report | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $ReportPath -Encoding utf8
  Pop-Location
}
