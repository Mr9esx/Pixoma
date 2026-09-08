Set-StrictMode -Version 3.0
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$GitHubOwner = "Mr9esx"
$GitHubRepo = "Pixoma"

function Fail([string]$Message) {
    throw "pixoma installer: $Message"
}

$operatingSystem = "windows"
$binaryExtension = ".exe"

if ($env:PROCESSOR_ARCHITEW6432) {
    $detectedArchitecture = $env:PROCESSOR_ARCHITEW6432
} else {
    $detectedArchitecture = $env:PROCESSOR_ARCHITECTURE
}

$architecture = switch ($detectedArchitecture) {
    { $_ -in @("AMD64", "x86_64") } { "amd64" }
    { $_ -in @("ARM64", "arm64") } { "arm64" }
    default { Fail "unsupported architecture: $detectedArchitecture" }
}

$version = $env:PIXOMA_VERSION
if ([string]::IsNullOrWhiteSpace($version)) {
    $releaseUri = "https://github.com/$GitHubOwner/$GitHubRepo/releases.atom"
    try {
        $releaseFeed = (Invoke-WebRequest -Uri $releaseUri -Headers @{ "User-Agent" = "pixoma-installer" }).Content
        $versionMatch = [regex]::Match(
            $releaseFeed,
            '<link rel="alternate" type="text/html" href="https://github.com/[^/]+/[^/]+/releases/tag/([^"]+)"'
        )
        if ($versionMatch.Success) {
            $version = $versionMatch.Groups[1].Value
        }
    } catch {
        Fail "unable to resolve the latest release"
    }
}

if ([string]::IsNullOrWhiteSpace($version)) {
    Fail "the release metadata did not contain a tag name"
}

$releaseNumber = $version.TrimStart("v")
$archiveName = "pixoma_${releaseNumber}_${operatingSystem}_${architecture}.tar.gz"
$downloadUri = "https://github.com/$GitHubOwner/$GitHubRepo/releases/download/$version/$archiveName"
$temporaryDir = Join-Path ([IO.Path]::GetTempPath()) ("pixoma-install-" + [Guid]::NewGuid().ToString("N"))
$archivePath = Join-Path $temporaryDir $archiveName
$checksumPath = Join-Path $temporaryDir "checksums.txt"
$extractDir = Join-Path $temporaryDir "extract"

if ($env:PIXOMA_INSTALL_DIR) {
    $installDir = $env:PIXOMA_INSTALL_DIR
} else {
    $installDir = Join-Path $HOME ".pixoma/bin"
}

$controlPlaneBinary = "pixoma$binaryExtension"
$edgeBinary = "pixoma-edge-agent$binaryExtension"

try {
    New-Item -ItemType Directory -Path $temporaryDir -Force | Out-Null

    Write-Host "Downloading Pixoma $version for $operatingSystem/$architecture..."
    try {
        Invoke-WebRequest -Uri $downloadUri -OutFile $archivePath
    } catch {
        Fail "unable to download $downloadUri"
    }

    try {
        Invoke-WebRequest -Uri "https://github.com/$GitHubOwner/$GitHubRepo/releases/download/$version/checksums.txt" -OutFile $checksumPath
    } catch {
        Fail "unable to download checksums"
    }

    $expectedChecksum = (Get-Content $checksumPath |
        ForEach-Object { ($_ -split '\s+', 2) } |
        Where-Object { $_.Count -eq 2 -and $_[1].Trim() -eq $archiveName } |
        Select-Object -First 1)[0]

    if ([string]::IsNullOrWhiteSpace($expectedChecksum)) {
        Fail "checksum file does not contain $archiveName"
    }

    $actualChecksum = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToLowerInvariant()
    if ($actualChecksum -ne $expectedChecksum.ToLowerInvariant()) {
        Fail "checksum verification failed for $archiveName"
    }

    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
    tar -xzf $archivePath -C $extractDir
    if ($LASTEXITCODE -ne 0) {
        Fail "unable to extract $archiveName"
    }

    $isEdgeInstall = $env:PIXOMA_INSTALL -eq "edge"
    if ($isEdgeInstall) {
        foreach ($requiredVariable in @("CONTROL_PLANE_URL", "AGENT_TOKEN", "EDGE_ID")) {
            if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($requiredVariable))) {
                Fail "$requiredVariable is required for edge installation"
            }
        }

        if (-not (Test-Path (Join-Path $extractDir $edgeBinary))) {
            Fail "$edgeBinary is missing from the release archive"
        }
    } elseif (-not (Test-Path (Join-Path $extractDir $controlPlaneBinary)) -or -not (Test-Path (Join-Path $extractDir $edgeBinary))) {
        Fail "release archive is missing an expected binary"
    }

    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    Copy-Item -Path (Join-Path $extractDir $edgeBinary) -Destination (Join-Path $installDir $edgeBinary) -Force
    if (-not $isEdgeInstall) {
        Copy-Item -Path (Join-Path $extractDir $controlPlaneBinary) -Destination (Join-Path $installDir $controlPlaneBinary) -Force
    }

    $currentProcessPath = $env:Path
    if (($currentProcessPath -split ";") -notcontains $installDir) {
        $env:Path = "$installDir;$currentProcessPath"
    }

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $userPath) {
        [Environment]::SetEnvironmentVariable("Path", $installDir, "User")
    } elseif (($userPath -split ";") -notcontains $installDir) {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    }

    if ($isEdgeInstall) {
        Write-Host "`nPixoma edge agent installed: $(Join-Path $installDir $edgeBinary)"
        Write-Host "Start a new terminal, then run: pixoma-edge-agent"
    } else {
        Write-Host "`nPixoma control plane installed: $(Join-Path $installDir $controlPlaneBinary)"
        Write-Host "Start a new terminal, then run: pixoma"
        Write-Host "Open the admin URL printed by pixoma and complete the setup wizard."
    }
} finally {
    if (Test-Path $temporaryDir) {
        Remove-Item -LiteralPath $temporaryDir -Recurse -Force
    }
}
