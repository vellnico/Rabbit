# Builds build\bin\Rabbit.exe. With -Installer also builds
# build\bin\Rabbit-Setup.exe, the installer from installer\ (its UI is
# frontend\installer, styled like the launcher).
#
# Ely.by OAuth credentials (for "sign in with the browser") are taken from the
# ELY_CLIENT_ID / ELY_CLIENT_SECRET environment variables or from the
# git-ignored file build.local (or elyby.local) next to this script:
#
#   ELY_CLIENT_ID=rabbit-launcher
#   ELY_CLIENT_SECRET=...
#
# Without them the launcher falls back to the login/password form.

param([switch]$Installer)

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

foreach ($name in 'elyby.local', 'build.local') {
    $local = Join-Path $PSScriptRoot $name
    if (Test-Path $local) {
        foreach ($line in Get-Content $local) {
            if ($line -match '^\s*(ELY_CLIENT_ID|ELY_CLIENT_SECRET)\s*=\s*(.+?)\s*$') {
                Set-Item "env:$($Matches[1])" $Matches[2]
            }
        }
    }
}

# The version lives in wails.json; the launcher and the installer get it from here.
$version = (Get-Content wails.json -Raw | ConvertFrom-Json).info.productVersion

$pkg = 'github.com/vellnico/rabbit/internal/auth'
$ldflags = "-s -w -X github.com/vellnico/rabbit/internal/launch.LauncherVersion=$version"
if ($env:ELY_CLIENT_ID -and $env:ELY_CLIENT_SECRET) {
    $ldflags += " -X $pkg.ElyClientID=$env:ELY_CLIENT_ID -X $pkg.ElyClientSecret=$env:ELY_CLIENT_SECRET"
    Write-Host 'Ely.by OAuth: enabled'
} else {
    Write-Host 'Ely.by OAuth: not configured (password login will be used)'
}

function Invoke-Wails {
    wails @args
    if ($LASTEXITCODE -ne 0) { throw "wails $($args -join ' ') failed" }
}

function Compress-Gzip($src, $dst) {
    $in = [IO.File]::OpenRead((Resolve-Path $src))
    $out = [IO.File]::Create((Join-Path (Get-Location) $dst)) # .NET does not follow Push-Location
    $gz = New-Object IO.Compression.GZipStream($out, [IO.Compression.CompressionLevel]::Optimal)
    try { $in.CopyTo($gz) } finally { $gz.Dispose(); $out.Dispose(); $in.Dispose() }
}

Invoke-Wails build -clean -trimpath -ldflags $ldflags
if (-not $Installer) { return }

# ---- installer ----
$setupFlags = "-s -w -X main.version=$version"
Push-Location installer
try {
    # The version shown in the exe properties comes from installer\wails.json.
    $cfgText = Get-Content wails.json -Raw
    $cfg = $cfgText | ConvertFrom-Json
    if ($cfg.info.productVersion -ne $version) {
        $cfgText = $cfgText -replace '("productVersion"\s*:\s*")[^"]*(")', "`${1}$version`${2}"
        [IO.File]::WriteAllText((Join-Path (Get-Location) 'wails.json'), $cfgText)
    }

    New-Item -ItemType Directory -Force build\windows, payload | Out-Null
    Copy-Item ..\build\appicon.png build\appicon.png
    Copy-Item ..\build\windows\icon.ico build\windows\icon.ico

    # The UI is built here, not by wails: its frontend step empties dist\
    # when the frontend lives outside the project folder.
    pnpm --dir ..\frontend build:installer
    if ($LASTEXITCODE -ne 0) { throw 'installer frontend build failed' }

    # The uninstaller is the same program without the payload.
    Invoke-Wails build -clean -s -skipbindings -trimpath -ldflags $setupFlags -o uninstall.exe
    Compress-Gzip ..\build\bin\Rabbit.exe payload\Rabbit.exe.gz
    Compress-Gzip build\bin\uninstall.exe payload\uninstall.exe.gz

    Invoke-Wails build -s -skipbindings -trimpath -tags setup -ldflags $setupFlags -o Rabbit-Setup.exe
    Copy-Item build\bin\Rabbit-Setup.exe ..\build\bin\Rabbit-Setup.exe
} finally {
    Pop-Location
}
Write-Host "Installer: build\bin\Rabbit-Setup.exe"
