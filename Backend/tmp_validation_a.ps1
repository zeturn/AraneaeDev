$ErrorActionPreference='"'"'Stop'"'"'

function Invoke-JsonApi {
  param(
    [string]$Method,
    [string]$Url,
    [object]$Body = $null,
    [string]$Token = $null
  )
  $headers = @{}
  if ($Token) { $headers['"'"'Authorization'"'"'] = "'"'"Bearer $Token"'"'" }
  try {
    if ($null -ne $Body) {
      $json = $Body | ConvertTo-Json -Depth 12 -Compress
      $resp = Invoke-WebRequest -Method $Method -Uri $Url -Headers $headers -ContentType '"'"'application/json'"'"' -Body $json
    } else {
      $resp = Invoke-WebRequest -Method $Method -Uri $Url -Headers $headers
    }
    return [pscustomobject]@{ Status = [int]$resp.StatusCode; Body = $resp.Content }
  } catch {
    $status = -1
    $body = $_.Exception.Message
    if ($_.Exception.Response) {
      $status = [int]$_.Exception.Response.StatusCode
      $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
      $body = $reader.ReadToEnd()
      $reader.Close()
    }
    return [pscustomobject]@{ Status = $status; Body = $body }
  }
}

function Invoke-UploadApi {
  param(
    [string]$Url,
    [string]$Token,
    [string]$FilePath
  )
  $headers = @{ Authorization = "'"'"Bearer $Token"'"'" }
  try {
    $resp = Invoke-WebRequest -Method Post -Uri $Url -Headers $headers -Form @{ file = Get-Item -LiteralPath $FilePath }
    return [pscustomobject]@{ Status = [int]$resp.StatusCode; Body = $resp.Content }
  } catch {
    $status = -1
    $body = $_.Exception.Message
    if ($_.Exception.Response) {
      $status = [int]$_.Exception.Response.StatusCode
      $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
      $body = $reader.ReadToEnd()
      $reader.Close()
    }
    return [pscustomobject]@{ Status = $status; Body = $body }
  }
}

$base='"'"'http://127.0.0.1:8180'"'"'
$api="'"'"$base/api/v1"'"'"

$health = Invoke-JsonApi -Method Get -Url "$base/healthz"
Write-Output "'"'"HEALTH_STATUS=$($health.Status)"'"'"
Write-Output "'"'"HEALTH_BODY=$($health.Body)"'"'"

$login = Invoke-JsonApi -Method Post -Url "$api/auth/login" -Body @{ username='"'"'admin'"'"'; password='"'"'admin123'"'"' }
Write-Output "'"'"LOGIN_STATUS=$($login.Status)"'"'"
Write-Output "'"'"LOGIN_BODY=$($login.Body)"'"'"
if ($login.Status -ne 200) { throw "'"'"Login failed"'"'" }
$loginObj = $login.Body | ConvertFrom-Json
$token = $loginObj.token
if (-not $token) { throw '"'"'No token returned'"'"' }

$tasksResp = Invoke-JsonApi -Method Get -Url "$api/tasks" -Token $token
Write-Output "'"'"TASKS_STATUS=$($tasksResp.Status)"'"'"
$tasksSnippet = if ($tasksResp.Body.Length -gt 800) { $tasksResp.Body.Substring(0,800) + '"'"'...'"'"' } else { $tasksResp.Body }
Write-Output "'"'"TASKS_BODY_SNIPPET=$tasksSnippet"'"'"
if ($tasksResp.Status -ne 200) { throw '"'"'Get tasks failed'"'"' }
$tasks = @($tasksResp.Body | ConvertFrom-Json)

$task = $null
if ($tasks.Count -gt 0) {
  $task = $tasks[0]
  Write-Output "'"'"TASK_SOURCE=existing"'"'"
  Write-Output "'"'"TASK_ID=$($task.id)"'"'"
} else {
  Write-Output "'"'"TASK_SOURCE=created"'"'"
  $projectName = "'"'"numeric-nq-proj-"'"'" + [Guid]::NewGuid().ToString('"'"'N'"'"').Substring(0,6)
  $projectResp = Invoke-JsonApi -Method Post -Url "$api/projects" -Token $token -Body @{ name=$projectName }
  Write-Output "'"'"PROJECT_STATUS=$($projectResp.Status)"'"'"
  Write-Output "'"'"PROJECT_BODY=$($projectResp.Body)"'"'"
  if ($projectResp.Status -ne 200) { throw '"'"'Create project failed'"'"' }
  $project = $projectResp.Body | ConvertFrom-Json

  $tmpRoot = Join-Path $env:TEMP ("'"'"araneae-nq-"'"'" + [Guid]::NewGuid().ToString('"'"'N'"'"'))
  New-Item -ItemType Directory -Path $tmpRoot | Out-Null
  $txtPath = Join-Path $tmpRoot '"'"'README.txt'"'"'
  Set-Content -LiteralPath $txtPath -Value '"'"'temporary artifact for numeric node_queue validation'"'"'
  $zipPath = Join-Path $tmpRoot '"'"'artifact.zip'"'"'
  Compress-Archive -Path $txtPath -DestinationPath $zipPath -Force

  $uploadResp = Invoke-UploadApi -Url "$api/projects/$($project.id)/upload" -Token $token -FilePath $zipPath
  Write-Output "'"'"UPLOAD_STATUS=$($uploadResp.Status)"'"'"
  Write-Output "'"'"UPLOAD_BODY=$($uploadResp.Body)"'"'"
  if ($uploadResp.Status -ne 200) { throw '"'"'Upload failed'"'"' }
  $version = $uploadResp.Body | ConvertFrom-Json

  $taskPayload = @{
    name = '"'"'numeric-nq-task'"'"'
    project_id = $project.id
    version_id = $version.id
    entry_command = '"'"'echo numeric-node-queue'"'"'
    cron_expr = '"'"''"'"'
    node_queue = '"'"'default'"'"'
  }
  $createTaskResp = Invoke-JsonApi -Method Post -Url "$api/tasks" -Token $token -Body $taskPayload
  Write-Output "'"'"CREATE_TASK_STATUS=$($createTaskResp.Status)"'"'"
  Write-Output "'"'"CREATE_TASK_BODY=$($createTaskResp.Body)"'"'"
  if ($createTaskResp.Status -ne 200) { throw '"'"'Create task failed'"'"' }
  $task = $createTaskResp.Body | ConvertFrom-Json
}

$schedulePayload = @{
  name = '"'"'numeric-node-queue-schedule'"'"'
  description = '"'"'validation numeric node_queue'"'"'
  enabled = $false
  task_id = $task.id
  node_queue = 1
  order = @{
    name = '"'"'numeric-node-queue-order'"'"'
    schedule = @(
      @{ task_id = $task.id; trigger = '"'"'api'"'"'; node = @('"'"'default'"'"') },
      @{ task_id = $task.id; trigger = '"'"'previous'"'"'; node = @('"'"'default'"'"') }
    )
  }
}

$scheduleResp = Invoke-JsonApi -Method Post -Url "$api/schedules" -Token $token -Body $schedulePayload
Write-Output "'"'"SCHEDULE_STATUS=$($scheduleResp.Status)"'"'"
Write-Output "'"'"SCHEDULE_BODY=$($scheduleResp.Body)"'"'"
$hasOldError = $scheduleResp.Body -match '"'"'cannot unmarshal number.*node_queue'"'"'
Write-Output "'"'"OLD_UNMARSHAL_ERROR_PRESENT=$hasOldError"'"'"
