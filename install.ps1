$ErrorActionPreference = "Stop"

$Repo = "pablontiv/rootline"
$Binary = "rootline"

function Main {
    $arch = Get-Arch
    $version = Get-LatestVersion
    $installDir = Get-InstallDir
    Install-Binary -Version $version -Arch $arch -InstallDir $installDir
    Verify-Installation -InstallDir $installDir
}

function Get-Arch {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
    }
}

function Get-LatestVersion {
    Write-Host "Fetching latest version..."
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $version = $release.tag_name
    if (-not $version) {
        throw "Could not determine latest version. Check https://github.com/$Repo/releases"
    }
    Write-Host "Latest version: $version"
    return $version
}

function Get-InstallDir {
    if ($env:ROOTLINE_INSTALL_DIR) {
        return $env:ROOTLINE_INSTALL_DIR
    }

    $dir = Join-Path $env:LOCALAPPDATA "rootline" "bin"

    if (-not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$dir*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$dir", "User")
        $env:Path = "$env:Path;$dir"
        Write-Host "Added $dir to user PATH"
    }

    return $dir
}

function Install-Binary {
    param($Version, $Arch, $InstallDir)

    $versionNum = $Version.TrimStart("v")
    $archive = "${Binary}_${versionNum}_windows_${Arch}.zip"
    $url = "https://github.com/$Repo/releases/download/$Version/$archive"

    $tmpDir = Join-Path $env:TEMP "rootline-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    try {
        $zipPath = Join-Path $tmpDir $archive

        Write-Host "Downloading $archive..."
        Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing

        # Verify checksum — mandatory, abort on mismatch.
        Write-Host "Verifying checksum..."
        $checksumUrl = "https://github.com/$Repo/releases/download/$Version/checksums.txt"
        $checksumPath = Join-Path $tmpDir "checksums.txt"
        try {
            Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumPath -UseBasicParsing
        } catch {
            throw "Could not fetch checksums.txt: $_"
        }
        $expectedLine = Select-String -Path $checksumPath -Pattern ([regex]::Escape($archive)) | Select-Object -First 1
        if (-not $expectedLine) {
            throw "Archive '$archive' not found in checksums.txt"
        }
        $expectedHash = ($expectedLine.Line -split '\s+')[0].ToUpper()
        $actualHash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToUpper()
        if ($actualHash -ne $expectedHash) {
            throw "Checksum mismatch for ${archive}: expected $expectedHash, got $actualHash"
        }
        Write-Host "Checksum verified."

        Write-Host "Extracting..."
        Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force

        $binaryPath = Join-Path $tmpDir "$Binary.exe"
        if (-not (Test-Path $binaryPath)) {
            throw "Binary not found in archive"
        }

        Write-Host "Installing to $InstallDir..."
        Copy-Item -Path $binaryPath -Destination (Join-Path $InstallDir "$Binary.exe") -Force
    }
    finally {
        Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Verify-Installation {
    param($InstallDir)

    $exe = Join-Path $InstallDir "$Binary.exe"
    if (Test-Path $exe) {
        $ver = & $exe --version 2>&1
        Write-Host "Installed $ver to $exe"
    }
    else {
        throw "Installation failed: $exe not found"
    }
}

Main
