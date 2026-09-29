param([string]$Version, [string]$InstallDir, [switch]$Force, [switch]$NoModifyPath)
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
if ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne 'X64') { throw 'Windows x86_64 is required.' }
if (!$InstallDir) { $InstallDir = $env:SHOSHINVIM_INSTALL_DIR }
if (!$InstallDir) { $InstallDir = $env:XDG_BIN_HOME }
if (!$InstallDir) { $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\ShoshinVim' }
$InstallDir = [IO.Path]::GetFullPath($InstallDir)
$repo = 'https://github.com/shoshindev/shoshinvim'
if (!$Version) { $release = Invoke-RestMethod 'https://api.github.com/repos/shoshindev/shoshinvim/releases/latest'; if ($release.prerelease) { throw 'Unexpected prerelease' }; $Version = $release.tag_name }
if ($Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$') { throw 'Invalid release version' }
$asset = "shoshinvim_$($Version.Substring(1))_x86_64-pc-windows-msvc.zip"
$temp = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $temp | Out-Null
try {
  Invoke-WebRequest "$repo/releases/download/$Version/$asset" -OutFile (Join-Path $temp 'archive.zip')
  Invoke-WebRequest "$repo/releases/download/$Version/SHA256SUMS" -OutFile (Join-Path $temp 'SHA256SUMS')
  $rows = @(Get-Content (Join-Path $temp 'SHA256SUMS') | Where-Object { ($_ -split '\s+')[1] -eq $asset })
  $archiveHash = (Get-FileHash (Join-Path $temp 'archive.zip') -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($rows.Count -ne 1 -or ($rows[0] -split '\s+')[0] -ne $archiveHash) { throw 'Archive checksum mismatch; nothing installed' }
  Add-Type -AssemblyName System.IO.Compression.FileSystem
  $zip = [IO.Compression.ZipFile]::OpenRead((Join-Path $temp 'archive.zip'))
  try {
    $entries = @($zip.Entries | Where-Object { $_.FullName -ceq 'shoshinvim.exe' })
    if ($entries.Count -ne 1) { throw 'Expected exactly one executable' }
    [IO.Compression.ZipFileExtensions]::ExtractToFile($entries[0], (Join-Path $temp 'shoshinvim.exe'))
  } finally { $zip.Dispose() }
  $dest = Join-Path $InstallDir 'shoshinvim.exe'
  $stateBase = $env:XDG_STATE_HOME; if (!$stateBase) { $stateBase = $env:LOCALAPPDATA }
  $stateDir = Join-Path $stateBase 'shoshinvim'
  $receiptPath = Join-Path $stateDir 'install.json'
  foreach ($path in @($dest, $receiptPath)) { if ((Test-Path $path) -and ((Get-Item $path).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing reparse-point destination' } }
  if (Test-Path $dest) {
    if (!$Force -or !(Test-Path $receiptPath)) { throw 'Existing binary requires a valid script receipt and -Force' }
    $old = Get-Content $receiptPath -Raw | ConvertFrom-Json
    if ($old.source -ne 'script' -or $old.path -ne $dest -or $old.sha256 -ne (Get-FileHash $dest -Algorithm SHA256).Hash) { throw 'Receipt does not own executable' }
    Copy-Item $dest (Join-Path $temp 'previous.exe')
  }
  New-Item -ItemType Directory -Force -Path $InstallDir, $stateDir | Out-Null
  $staged = Join-Path $InstallDir ('.shoshinvim-' + [Guid]::NewGuid().ToString() + '.exe')
  Copy-Item (Join-Path $temp 'shoshinvim.exe') $staged
  $receipt = @{version=$Version;source='script';path=$dest;sha256=(Get-FileHash $staged -Algorithm SHA256).Hash.ToLowerInvariant();archive_sha256=$archiveHash;timestamp=[DateTime]::UtcNow.ToString('o')}
  $receiptTemp = Join-Path $stateDir ('.receipt-' + [Guid]::NewGuid().ToString())
  [IO.File]::WriteAllText($receiptTemp, ($receipt | ConvertTo-Json))
  try {
    if (Test-Path $dest) { [IO.File]::Replace($staged, $dest, $null) } else { [IO.File]::Move($staged, $dest) }
    if (Test-Path $receiptPath) { [IO.File]::Replace($receiptTemp, $receiptPath, $null) } else { [IO.File]::Move($receiptTemp, $receiptPath) }
  } catch {
    if (Test-Path (Join-Path $temp 'previous.exe')) { Copy-Item (Join-Path $temp 'previous.exe') $dest -Force }
    elseif (Test-Path $dest) { Remove-Item $dest }
    throw
  } finally { Remove-Item $staged, $receiptTemp -ErrorAction SilentlyContinue }
  Write-Output "Installed $Version to $dest. Add $InstallDir to PATH if needed. No startup files were changed."
} finally { Remove-Item $temp -Recurse -Force }
