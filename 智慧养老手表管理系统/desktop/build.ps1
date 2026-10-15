$ErrorActionPreference='Stop'
$project=Split-Path $PSScriptRoot -Parent
Push-Location $project
try {
  npm ci --prefix oldIot
  if ($LASTEXITCODE -ne 0) { throw 'Frontend dependency install failed' }
  npm --prefix oldIot run build
  if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed' }
  mvn -f careold/pom.xml '-Dnet.bytebuddy.experimental=true' clean package
  if ($LASTEXITCODE -ne 0) { throw 'Backend tests/build failed' }
  npm ci --prefix desktop/build-tools
  if ($LASTEXITCODE -ne 0) { throw 'Packaging tools install failed' }
  python desktop/download-runtimes.py
  if ($LASTEXITCODE -ne 0) { throw 'Runtime download/check failed' }
  node desktop/prepare-package.cjs
  if ($LASTEXITCODE -ne 0) { throw 'Desktop package failed' }
  python desktop/prepare-notices.py
  if ($LASTEXITCODE -ne 0) { throw 'License collection failed' }
  & desktop/sign.ps1 -FilePath (Join-Path $project 'outputs/desktop-release-20261015/release/Suining/Suining.exe')
  & desktop/build-installer.ps1
} finally { Pop-Location }
