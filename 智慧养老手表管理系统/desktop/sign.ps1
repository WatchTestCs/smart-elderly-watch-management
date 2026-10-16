param(
  [Parameter(Mandatory=$true)][string]$FilePath,
  [string]$CertificateThumbprint=$env:SUINING_SIGNING_THUMBPRINT,
  [string]$TimestampServer=$env:SUINING_TIMESTAMP_SERVER,
  [switch]$RequireTrusted=($env:SUINING_REQUIRE_TRUSTED_SIGNATURE -eq '1')
)
$ErrorActionPreference='Stop'
if ($RequireTrusted -and -not $CertificateThumbprint) { throw 'Trusted release signing requires SUINING_SIGNING_THUMBPRINT.' }
if ($CertificateThumbprint) {
  $thumbprint=$CertificateThumbprint.Replace(' ','').ToUpperInvariant()
  $certificate=Get-ChildItem Cert:\CurrentUser\My,Cert:\LocalMachine\My | Where-Object { $_.Thumbprint -eq $thumbprint } | Select-Object -First 1
  if (-not $certificate -or -not $certificate.HasPrivateKey -or $certificate.NotAfter -le (Get-Date) -or $certificate.NotBefore -gt (Get-Date)) { throw 'The selected signing certificate is missing, expired, or has no accessible private key.' }
  if (-not ($certificate.EnhancedKeyUsageList | Where-Object { [string]$_.ObjectId -eq '1.3.6.1.5.5.7.3.3' })) { throw 'The selected certificate does not permit code signing.' }
  if (-not $TimestampServer) { throw 'Formal certificate signing requires SUINING_TIMESTAMP_SERVER from your certificate provider.' }
  if ($certificate.Subject -eq $certificate.Issuer) { throw 'A self-signed certificate cannot be selected for formal release signing.' }
} else {
  $certificate=Get-ChildItem Cert:\CurrentUser\My | Where-Object { $_.Subject -eq 'CN=Suining Local Build' -and $_.HasPrivateKey -and $_.NotAfter -gt (Get-Date) } | Sort-Object NotAfter -Descending | Select-Object -First 1
}
if (-not $certificate) {
  $certificate=New-SelfSignedCertificate -Type CodeSigningCert -Subject 'CN=Suining Local Build' -CertStoreLocation Cert:\CurrentUser\My -KeyAlgorithm RSA -KeyLength 3072 -HashAlgorithm SHA256 -KeyExportPolicy NonExportable -NotAfter (Get-Date).AddYears(2) -FriendlyName 'Suining local release signing'
}
$options=@{ LiteralPath=$FilePath; Certificate=$certificate; HashAlgorithm='SHA256' }
if ($TimestampServer) { $options.TimestampServer=$TimestampServer }
$signature=Set-AuthenticodeSignature @options
if (-not $signature.SignerCertificate) { throw "No signature written: $FilePath" }
if ($CertificateThumbprint -and ($signature.Status -ne 'Valid' -or -not $signature.TimeStamperCertificate)) { throw "Formal signature or timestamp verification failed: $($signature.Status)" }
# Self-signed certificates are not added to Trusted Root. NotTrusted is expected.
[pscustomobject]@{File=$FilePath;Status=$signature.Status.ToString();Thumbprint=$certificate.Thumbprint;Subject=$certificate.Subject;Timestamped=[bool]$signature.TimeStamperCertificate;TrustedOnThisMachine=($signature.Status -eq 'Valid')} | ConvertTo-Json -Compress
