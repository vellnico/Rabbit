# Builds build\bin\Rabbit.exe.
#
# Sign-in keys are taken from environment variables or from the git-ignored
# file build.local next to this script (elyby.local is read too):
#
#   MS_CLIENT_ID=...         Azure app for Microsoft sign-in
#   ELY_CLIENT_ID=...        Ely.by app for browser sign-in
#   ELY_CLIENT_SECRET=...
#
# Without the Ely.by keys the launcher falls back to the login/password form;
# without MS_CLIENT_ID Microsoft sign-in is unavailable.

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

foreach ($name in 'elyby.local', 'build.local') {
    $local = Join-Path $PSScriptRoot $name
    if (Test-Path $local) {
        foreach ($line in Get-Content $local) {
            if ($line -match '^\s*(MS_CLIENT_ID|ELY_CLIENT_ID|ELY_CLIENT_SECRET)\s*=\s*(.+?)\s*$') {
                Set-Item "env:$($Matches[1])" $Matches[2]
            }
        }
    }
}

$pkg = 'github.com/vellnico/rabbit/internal/auth'
$ldflags = '-s -w'
if ($env:ELY_CLIENT_ID -and $env:ELY_CLIENT_SECRET) {
    $ldflags += " -X $pkg.ElyClientID=$env:ELY_CLIENT_ID -X $pkg.ElyClientSecret=$env:ELY_CLIENT_SECRET"
    Write-Host 'Ely.by OAuth: enabled'
} else {
    Write-Host 'Ely.by OAuth: not configured (password login will be used)'
}
if ($env:MS_CLIENT_ID) {
    $ldflags += " -X $pkg.MSClientID=$env:MS_CLIENT_ID"
    Write-Host 'Microsoft: enabled'
} else {
    Write-Host 'Microsoft: not configured'
}

wails build -clean -trimpath -ldflags $ldflags
