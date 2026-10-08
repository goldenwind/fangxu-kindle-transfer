param(
    [string]$ReleaseDir = ''
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$config = Get-Content -LiteralPath (Join-Path $projectRoot 'src-tauri/tauri.conf.json') -Raw -Encoding utf8 | ConvertFrom-Json
if (!$ReleaseDir) {
    $ReleaseDir = Join-Path $projectRoot 'src-tauri/target/x86_64-pc-windows-msvc/release'
}
$ReleaseDir = (Resolve-Path -LiteralPath $ReleaseDir).Path
$client = Join-Path $ReleaseDir 'fangxu-kindle-transfer-desktop.exe'
$service = Join-Path $ReleaseDir 'fangxu-kindle-service.exe'
foreach ($binary in @($client, $service)) {
    if (!(Test-Path -LiteralPath $binary -PathType Leaf)) {
        throw "Missing portable binary: $binary. Build the Windows x86-64 desktop client first."
    }
}

$outputDir = Join-Path $ReleaseDir 'bundle/portable'
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
$archive = Join-Path $outputDir "fangxu-kindle-transfer_$($config.version)_windows-x64-portable.zip"
$stagingDir = Join-Path ([System.IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString())
$appDir = Join-Path $stagingDir 'Fangxu-Kindle-Transfer'
try {
    New-Item -ItemType Directory -Path $appDir -Force | Out-Null
    Copy-Item -LiteralPath $client -Destination (Join-Path $appDir 'Fangxu-Kindle-Transfer.exe')
    Copy-Item -LiteralPath $service -Destination $appDir
    # Include any runtime DLLs emitted next to the application, never build/deps files.
    Get-ChildItem -LiteralPath $ReleaseDir -Filter '*.dll' -File | Copy-Item -Destination $appDir
    Copy-Item -LiteralPath (Join-Path $projectRoot 'docs/windows-portable.txt') -Destination (Join-Path $appDir 'README.txt')
    Copy-Item -LiteralPath (Join-Path $projectRoot 'LICENSE') -Destination (Join-Path $appDir 'LICENSE.txt')
    Compress-Archive -LiteralPath $appDir -DestinationPath $archive -CompressionLevel Optimal -Force
    Write-Output "Windows portable package: $archive"
} finally {
    if (Test-Path -LiteralPath $stagingDir) {
        Remove-Item -LiteralPath $stagingDir -Recurse -Force
    }
}
