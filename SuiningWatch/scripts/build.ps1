param([string]$DevEcoHome = 'D:\Huawei\DevEco Studio')
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
$nodeExe = Join-Path $DevEcoHome 'tools\node\node.exe'
$hvigorExe = Join-Path $DevEcoHome 'tools\hvigor\hvigor\bin\hvigor.js'
if (!(Test-Path -LiteralPath $nodeExe)) { throw "未找到 DevEco Node：$nodeExe，请传入 -DevEcoHome" }
$moduleRoot = Join-Path $projectRoot 'node_modules\@ohos'
New-Item -ItemType Directory -Path $moduleRoot -Force | Out-Null
foreach ($package in @('hvigor', 'hvigor-ohos-plugin')) {
  $link = Join-Path $moduleRoot $package
  $target = Join-Path $DevEcoHome "tools\hvigor\$package"
  if (!(Test-Path -LiteralPath $link)) {
    New-Item -ItemType Junction -Path $link -Target $target | Out-Null
  }
}
$env:NODE_PATH = Join-Path $projectRoot 'node_modules'
$env:DEVECO_SDK_HOME = Join-Path $DevEcoHome 'sdk'
$env:JAVA_HOME = Join-Path $DevEcoHome 'jbr'
Push-Location $projectRoot
try {
  & $nodeExe $hvigorExe assembleHap --mode module -p module=entry@default -p product=default -p buildMode=debug --no-daemon
  if ($LASTEXITCODE -ne 0) { throw "Hvigor 编译失败（$LASTEXITCODE）" }
  $hap = Join-Path $projectRoot 'entry\build\default\outputs\default\entry-default-unsigned.hap'
  if (!(Test-Path -LiteralPath $hap)) { throw '编译未生成预期 HAP' }
  Write-Output "模拟器调试包：$hap"
} finally { Pop-Location }
