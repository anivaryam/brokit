$ErrorActionPreference = "Stop"

$Repo = "anivaryam/brokit"
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "brokit\bin" }

# Detect architecture
$OSArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
if (-not $OSArch) { $OSArch = $env:PROCESSOR_ARCHITECTURE }
Write-Host "Detected OS architecture: $OSArch"
$Arch = switch ($OSArch) {
    "X64"      { "amd64" }
    "AMD64"    { "amd64" }
    "Arm64"    { "arm64" }
    "X86"      { "386" }
    "Arm"      { "arm" }
    "Arm32"    { "arm" }
    default    { Write-Error "Unsupported architecture: $OSArch"; exit 1 }
}

# Get latest version
$Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Version = $Release.tag_name
if (-not $Version) {
    Write-Error "Failed to fetch latest version"
    exit 1
}

# Download
$Url = "https://github.com/$Repo/releases/download/$Version/brokit_windows_$Arch.zip"
Write-Host "Downloading brokit $Version for windows/$Arch..."

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "brokit-install"
if (Test-Path $TmpDir) { Remove-Item -Recurse -Force $TmpDir }
New-Item -ItemType Directory -Path $TmpDir | Out-Null

$ZipPath = Join-Path $TmpDir "brokit.zip"
Invoke-WebRequest -Uri $Url -OutFile $ZipPath

# Extract
Expand-Archive -Path $ZipPath -DestinationPath $TmpDir -Force

# Install
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}
Move-Item -Force (Join-Path $TmpDir "brokit.exe") (Join-Path $InstallDir "brokit.exe")

# Cleanup
Remove-Item -Recurse -Force $TmpDir

Write-Host "brokit $Version installed to $InstallDir\brokit.exe"

# Add to PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "Added $InstallDir to your PATH (restart your terminal to apply)"
}
