# Builds build\bin\Rabbit.exe.
#
# Ely.by OAuth credentials (for "sign in with the browser") are taken from the
# ELY_CLIENT_ID / ELY_CLIENT_SECRET environment variables or from the
# git-ignored file elyby.local next to this script:
#
#   ELY_CLIENT_ID=rabbit-launcher
#   ELY_CLIENT_SECRET=...
#
# Without them the launcher falls back to the login/password form.

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$local = Join-Path $PSScriptRoot 'elyby.local'
if (Test-Path $local) {
    foreach ($line in Get-Content $local) {
        if ($line -match '^\s*(ELY_CLIENT_ID|ELY_CLIENT_SECRET)\s*=\s*(.+?)\s*$') {
            Set-Item "env:$($Matches[1])" $Matches[2]
        }
    }
}

$ldflags = '-s -w'
if ($env:ELY_CLIENT_ID -and $env:ELY_CLIENT_SECRET) {
    $pkg = 'github.com/vellnico/rabbit/internal/auth'
    $ldflags += " -X $pkg.ElyClientID=$env:ELY_CLIENT_ID -X $pkg.ElyClientSecret=$env:ELY_CLIENT_SECRET"
    Write-Host 'Ely.by OAuth: enabled'
} else {
    Write-Host 'Ely.by OAuth: not configured (password login will be used)'
}

wails build -clean -trimpath -ldflags $ldflags
