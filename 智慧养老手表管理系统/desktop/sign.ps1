param([Parameter(Mandatory=$true)][string]$FilePath)
$ErrorActionPreference='Stop'
$certificate=Get-ChildItem Cert:\CurrentUser\My | Where-Object { $_.Subject -eq 'CN=Suining Local Build' -and $_.HasPrivateKey -and $_.NotAfter -gt (Get-Date) } | Sort-Object NotAfter -Descending | Select-Object -First 1
if (-not $certificate) {
  $certificate=New-SelfSignedCertificate -Type CodeSigningCert -Subject 'CN=Suining Local Build' -CertStoreLocation Cert:\CurrentUser\My -KeyAlgorithm RSA -KeyLength 3072 -HashAlgorithm SHA256 -KeyExportPolicy NonExportable -NotAfter (Get-Date).AddYears(2) -FriendlyName 'Suining local release signing'
}
$signature=Set-AuthenticodeSignature -LiteralPath $FilePath -Certificate $certificate -HashAlgorithm SHA256
if (-not $signature.SignerCertificate) { throw "No signature written: $FilePath" }
# Self-signed certificates are not added to Trusted Root. NotTrusted is expected.
[pscustomobject]@{File=$FilePath;Status=$signature.Status.ToString();Thumbprint=$certificate.Thumbprint;Subject=$certificate.Subject} | ConvertTo-Json -Compress
