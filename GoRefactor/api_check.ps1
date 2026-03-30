$ErrorActionPreference = "Stop"
function Invoke-ApiCheck {
    param(
        [string]$Name,
        [string]$Method,
        [string]$Uri,
        [hashtable]$Headers = @{},
        [object]$Body = $null,
        [string[]]$ExpectedAnyFields
    )

    $status = $null
    $json = $null
    $raw = $null
    $ok = $false
    $missingInfo = ""

    try {
        $params = @{ Method = $Method; Uri = $Uri; Headers = $Headers; ErrorAction = "Stop" }
        if ($null -ne $Body) {
            $params.Body = ($Body | ConvertTo-Json -Depth 10)
            $params.ContentType = "application/json"
        }
        $resp = Invoke-WebRequest @params
        $status = [int]$resp.StatusCode
        $raw = $resp.Content
        try { $json = $raw | ConvertFrom-Json -ErrorAction Stop } catch {}
    }
    catch {
        if ($_.Exception.Response) {
            $status = [int]$_.Exception.Response.StatusCode
            try {
                $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
                $raw = $reader.ReadToEnd()
                $reader.Close()
                try { $json = $raw | ConvertFrom-Json -ErrorAction Stop } catch {}
            } catch {}
        } else {
            $status = -1
            $missingInfo = $_.Exception.Message
        }
    }

    if ($json) {
        $propNames = @($json.PSObject.Properties.Name)
        $ok = ($ExpectedAnyFields | Where-Object { $propNames -contains $_ }).Count -gt 0
        if (-not $ok) { $missingInfo = "Top-level fields found: $($propNames -join ', ')" }
    } else {
        if (-not $missingInfo) { $missingInfo = "Response is not valid JSON or empty." }
    }

    [PSCustomObject]@{
        Step = $Name
        StatusCode = $status
        JsonFieldCheckPassed = $ok
        ExpectedAnyOf = ($ExpectedAnyFields -join ", ")
        Note = $missingInfo
        RawPreview = if ($raw) { if ($raw.Length -gt 250) { $raw.Substring(0,250) + "..." } else { $raw } } else { "" }
        Json = $json
    }
}

$base = "http://127.0.0.1:8180/api/v1"
$login = Invoke-ApiCheck -Name "POST /auth/login" -Method "POST" -Uri "$base/auth/login" -Body @{ username = "admin"; password = "admin123" } -ExpectedAnyFields @("token","access_token","data","user")

$token = $null
if ($login.Json) {
    if ($login.Json.PSObject.Properties.Name -contains "token") { $token = $login.Json.token }
    elseif ($login.Json.PSObject.Properties.Name -contains "access_token") { $token = $login.Json.access_token }
    elseif ($login.Json.PSObject.Properties.Name -contains "data") {
        $d = $login.Json.data
        if ($d -and $d.PSObject.Properties.Name -contains "token") { $token = $d.token }
        elseif ($d -and $d.PSObject.Properties.Name -contains "access_token") { $token = $d.access_token }
    }
}

$headers = @{}
if ($token) { $headers["Authorization"] = "Bearer $token" }

$users = Invoke-ApiCheck -Name "GET /users" -Method "GET" -Uri "$base/users" -Headers $headers -ExpectedAnyFields @("data","users","items","total")
$teams = Invoke-ApiCheck -Name "GET /teams/my_teams" -Method "GET" -Uri "$base/teams/my_teams" -Headers $headers -ExpectedAnyFields @("data","teams","items")

@($login,$users,$teams) | Select-Object Step,StatusCode,JsonFieldCheckPassed,ExpectedAnyOf,Note,RawPreview | ConvertTo-Json -Depth 6
