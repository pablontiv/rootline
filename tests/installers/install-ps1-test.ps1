$ErrorActionPreference = "Stop"

$env:ROOTLINE_INSTALLER_TESTING = "1"
. (Join-Path $PSScriptRoot ".." ".." "install.ps1")

function Assert-Equal {
    param($Expected, $Actual, [string]$Message)

    if ($Expected -ne $Actual) {
        throw "${Message}: expected '$Expected', got '$Actual'"
    }
}

$script:ApiCalled = $false
function Invoke-WebRequest {
    [CmdletBinding()]
    param($Uri, $Method, $MaximumRedirection, [switch]$UseBasicParsing)

    return [pscustomobject]@{
        BaseResponse = [pscustomobject]@{
            RequestMessage = [pscustomobject]@{
                RequestUri = [Uri]"https://github.com/pablontiv/rootline/releases/tag/v9.8.7"
            }
        }
    }
}
function Invoke-RestMethod {
    param($Uri)
    $script:ApiCalled = $true
    throw "REST fallback must not run after redirect success"
}

$version = Get-LatestVersion
Assert-Equal "v9.8.7" $version "redirect tag"
Assert-Equal $false $script:ApiCalled "redirect REST usage"

$script:ApiCalled = $false
function Invoke-WebRequest {
    [CmdletBinding()]
    param($Uri, $Method, $MaximumRedirection, [switch]$UseBasicParsing)

    return [pscustomobject]@{
        BaseResponse = [pscustomobject]@{
            ResponseUri = [Uri]"https://github.com/pablontiv/rootline/releases"
        }
    }
}
function Invoke-RestMethod {
    param($Uri)
    $script:ApiCalled = $true
    return [pscustomobject]@{ tag_name = "v7.6.5" }
}

$version = Get-LatestVersion
Assert-Equal "v7.6.5" $version "REST fallback tag"
Assert-Equal $true $script:ApiCalled "fallback REST usage"

Write-Host "install.ps1 resolver tests passed"
