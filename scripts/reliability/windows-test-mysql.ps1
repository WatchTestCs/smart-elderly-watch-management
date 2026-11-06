# This helper owns a fresh data directory and loopback port, never a Windows service.
function Start-DisposableTestMySQL {
 param([string]$MySQLBin)
 $ErrorActionPreference='Stop'
 if(!$MySQLBin){
  $server=Get-Command mysqld.exe -ErrorAction SilentlyContinue
  if(!$server){throw 'MySQL server tools are required for driver contracts; provide -TestMySQLBin or an explicit TEST_PACKED_MYSQL_DSN'}
  $MySQLBin=Split-Path $server.Source -Parent
 }
 $MySQLBin=[IO.Path]::GetFullPath($MySQLBin)
 foreach($name in @('mysqld.exe','mysql.exe','mysqladmin.exe')){
  if(!(Test-Path -LiteralPath (Join-Path $MySQLBin $name))){throw "Missing MySQL tool: $name"}
 }
 $directory=Join-Path ([IO.Path]::GetTempPath()) ('suining-test-mysql-'+[guid]::NewGuid().ToString('N'))
 New-Item -ItemType Directory -Path $directory -ErrorAction Stop | Out-Null
 $data=Join-Path $directory 'data'
 $listener=[Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,0)
 $listener.Start()
 try {$port=$listener.LocalEndpoint.Port} finally {$listener.Stop()}
 $basedir=Split-Path $MySQLBin -Parent
 & (Join-Path $MySQLBin 'mysqld.exe') --no-defaults --initialize-insecure "--basedir=$basedir" "--datadir=$data" --console > (Join-Path $directory 'initialize.stdout.txt') 2> (Join-Path $directory 'initialize.stderr.txt')
 if($LASTEXITCODE){throw "Isolated MySQL initialization failed; evidence: $directory"}
 $arguments=@('--no-defaults',"--basedir=`"$basedir`"","--datadir=`"$data`"","--port=$port",'--bind-address=127.0.0.1','--mysqlx=OFF','--console')
 $process=Start-Process -FilePath (Join-Path $MySQLBin 'mysqld.exe') -ArgumentList $arguments -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $directory 'server.stdout.txt') -RedirectStandardError (Join-Path $directory 'server.stderr.txt')
 $owner=[pscustomobject]@{Directory=$directory;Data=$data;Port=$port;Bin=$MySQLBin;Process=$process;ServerPID=0;UUID='';DSN="root@tcp(127.0.0.1:$port)/suining_driver?parseTime=true"}
 try {
  $ready=$false
  for($attempt=0;$attempt -lt 100;$attempt++){
   & (Join-Path $MySQLBin 'mysqladmin.exe') --no-defaults --protocol=TCP --host=127.0.0.1 "--port=$port" --user=root --password= --connect-timeout=1 ping > $null 2> $null
   if($LASTEXITCODE -eq 0){$ready=$true;break}
   if($process.HasExited){throw 'Isolated MySQL launcher exited before readiness'}
   Start-Sleep -Milliseconds 200
  }
  if(!$ready){throw 'Isolated MySQL readiness deadline exceeded'}
  $endpoint=Get-NetTCPConnection -LocalAddress 127.0.0.1 -LocalPort $port -State Listen -ErrorAction Stop
  $serverProcess=Get-CimInstance Win32_Process -Filter "ProcessId=$($endpoint.OwningProcess)"
  if(($serverProcess.ProcessId -ne $process.Id -and $serverProcess.ParentProcessId -ne $process.Id) -or !$serverProcess.CommandLine.Contains($data)){
   throw 'MySQL listener is not the process started by this helper'
  }
  $owner.ServerPID=[int]$endpoint.OwningProcess
  $uuid=& (Join-Path $MySQLBin 'mysql.exe') --no-defaults --protocol=TCP --host=127.0.0.1 "--port=$port" --user=root --password= --batch --skip-column-names --execute='CREATE DATABASE suining_driver; SELECT @@server_uuid'
  if($LASTEXITCODE -or !$uuid){throw 'Cannot establish isolated MySQL identity'}
  $owner.UUID=([string]$uuid).Trim()
  $owner | Select-Object Directory,Data,Port,ServerPID,UUID,@{n='LauncherPID';e={$_.Process.Id}} | ConvertTo-Json | Set-Content (Join-Path $directory 'owner.json') -Encoding utf8
  return $owner
 } catch {
  # Only terminate the launcher we created and its verified command-line children.
  $children=Get-CimInstance Win32_Process -Filter "ParentProcessId=$($process.Id)"
  foreach($child in $children){if($child.CommandLine -and $child.CommandLine.Contains($data)){Stop-Process -Id $child.ProcessId -ErrorAction SilentlyContinue}}
  if(!$process.HasExited){$process.Kill();$process.WaitForExit()}
  throw
 }
}

function Stop-DisposableTestMySQL {
 param([Parameter(Mandatory)]$Owner)
 $ErrorActionPreference='Stop'
 $endpoint=Get-NetTCPConnection -LocalAddress 127.0.0.1 -LocalPort $Owner.Port -State Listen -ErrorAction SilentlyContinue
 if(!$endpoint){if(!$Owner.Process.HasExited){throw 'Owned MySQL launcher still exists without its listener'};return}
 if($endpoint.OwningProcess -ne $Owner.ServerPID){throw 'Refusing to stop a different MySQL listener'}
 $identity=& (Join-Path $Owner.Bin 'mysql.exe') --no-defaults --protocol=TCP --host=127.0.0.1 "--port=$($Owner.Port)" --user=root --password= --batch --skip-column-names --execute='SELECT @@server_uuid'
 if($LASTEXITCODE -or ([string]$identity).Trim() -ne $Owner.UUID){throw 'Refusing to stop MySQL with a different UUID'}
 & (Join-Path $Owner.Bin 'mysqladmin.exe') --no-defaults --protocol=TCP --host=127.0.0.1 "--port=$($Owner.Port)" --user=root --password= shutdown
 if($LASTEXITCODE){throw "Owned MySQL shutdown failed; evidence: $($Owner.Directory)"}
 if(!$Owner.Process.WaitForExit(30000)){throw 'Owned MySQL launcher did not exit after shutdown'}
 if(Get-NetTCPConnection -LocalAddress 127.0.0.1 -LocalPort $Owner.Port -State Listen -ErrorAction SilentlyContinue){throw 'MySQL listener remains after shutdown'}
 # Keep logs/data for review. Do not delete paths or touch any system service.
}
