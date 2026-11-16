$ErrorActionPreference = "Stop"; $ProgressPreference = "SilentlyContinue"
$base = "https://github.com/wake/docket/releases/latest/download"
$arch = @{ AMD64 = "amd64"; ARM64 = "arm64" }[$env:PROCESSOR_ARCHITECTURE]
if (-not $arch) { throw "no dkt for $env:PROCESSOR_ARCHITECTURE" }
$name = "dkt-windows-$arch.exe"
$dir = Join-Path $env:LOCALAPPDATA "Programs\dkt"
$tmp = Join-Path $dir ".install-$PID"
New-Item -ItemType Directory -Force $tmp | Out-Null
try {
  Invoke-WebRequest "$base/$name" -OutFile "$tmp\$name" -UseBasicParsing
  Invoke-WebRequest "$base/checksums.txt" -OutFile "$tmp\checksums.txt" -UseBasicParsing
  $want = foreach ($l in Get-Content "$tmp\checksums.txt") { $f = -split $l; if ($f[1] -eq $name) { $f[0] } }
  $got = (Get-FileHash "$tmp\$name" -Algorithm SHA256).Hash
  if (-not $want -or $got -ne $want) { throw "SHA-256 of $name does not match checksums.txt" }
  & "$tmp\$name" --version
  if ($LASTEXITCODE -ne 0) { throw "$name does not run on this machine" }
  try { Move-Item -Force "$tmp\$name" "$dir\dkt.exe" }
  catch { throw "could not replace $dir\dkt.exe; Windows keeps it while a dkt runs: $_" }
  & "$dir\dkt.exe" --version
} finally {
  Remove-Item -Recurse -Force $tmp
}
