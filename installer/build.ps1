# Builds dist/TotalKrieg-setup.exe: the game, the latest MiusMissionMaker release
# and the Node.js from the PATH. Needs Go, Node.js and Inno Setup 6.
# Run from anywhere: installer/build.ps1 -Version 1.2.3
param([string]$Version = '0.0.0')

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$root = Split-Path $PSScriptRoot
$dist = Join-Path $root 'dist'
$app = Join-Path $dist 'app'
if (Test-Path $dist) { Remove-Item $dist -Recurse -Force }
New-Item -ItemType Directory -Path $app | Out-Null

# The token only lifts the API's rate limit, the releases are public.
$headers = @{ 'User-Agent' = 'TotalKrieg-build' }
if ($env:GH_TOKEN) { $headers.Authorization = "Bearer $env:GH_TOKEN" }
$release = Invoke-RestMethod -Headers $headers -Uri 'https://api.github.com/repos/xetra11/MiusMissionMaker/releases/latest'
$asset = $release.assets | Where-Object { $_.name -eq 'MiusMissionMaker-windows.zip' }
if (-not $asset) { throw "MiusMissionMaker $($release.tag_name) has no Windows ZIP" }
Write-Host "MiusMissionMaker $($release.tag_name)"
$zip = Join-Path $dist $asset.name
Invoke-WebRequest -UseBasicParsing -Uri $asset.browser_download_url -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $app
if (-not (Test-Path (Join-Path $app 'MiusMissionMaker/tools/scenario.js'))) {
    throw "MiusMissionMaker $($release.tag_name) has no tools/scenario.js"
}
if ($env:GITHUB_OUTPUT) { Add-Content -Path $env:GITHUB_OUTPUT -Value "missionmaker=$($release.tag_name)" }

Push-Location $root
try {
    go build -ldflags=-H=windowsgui -o dist/app/TotalKrieg.exe ./cmd/totalkrieg
    if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
} finally {
    Pop-Location
}

$node = (Get-Command node).Source
$nodeDir = Join-Path $app 'node'
New-Item -ItemType Directory -Path $nodeDir | Out-Null
Copy-Item $node -Destination $nodeDir
Copy-Item (Join-Path (Split-Path $node) 'LICENSE') -Destination $nodeDir
Write-Host "Node.js $(node --version)"

$iscc = Get-Command iscc -ErrorAction SilentlyContinue
$compiler = if ($iscc) { $iscc.Source } else { Join-Path ${env:ProgramFiles(x86)} 'Inno Setup 6/ISCC.exe' }
& $compiler "/DAppVersion=$Version" (Join-Path $PSScriptRoot 'TotalKrieg.iss')
if ($LASTEXITCODE -ne 0) { throw 'Inno Setup failed' }
