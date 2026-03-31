$ErrorActionPreference='"'"'Stop'"'"'
$ProgressPreference='"'"'SilentlyContinue'"'"'

function Invoke-Api {
  param([string]$Method,[string]$Url,[string]$Token=$null,[object]$Body=$null)
  $h=@{}
  if($Token){ $h.Authorization = "Bearer $Token" }
  try {
    if($null -ne $Body){
      $json = $Body | ConvertTo-Json -Depth 12 -Compress
      $r = Invoke-WebRequest -Method $Method -Uri $Url -Headers $h -ContentType '"'"'application/json'"'"' -Body $json -ErrorAction Stop
    } else {
      $r = Invoke-WebRequest -Method $Method -Uri $Url -Headers $h -ErrorAction Stop
    }
    return [pscustomobject]@{ Status=[int]$r.StatusCode; Body=$r.Content }
  } catch {
    $status = -1
    $body = $_.Exception.Message
    if($_.Exception.Response){
      $status = [int]$_.Exception.Response.StatusCode
      $sr = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
      $body = $sr.ReadToEnd(); $sr.Close()
    }
    return [pscustomobject]@{ Status=$status; Body=$body }
  }
}

function Get-List {
  param([string]$Json)
  if([string]::IsNullOrWhiteSpace($Json)){ return @() }
  try { $obj = $Json | ConvertFrom-Json -Depth 20 } catch { return @() }
  if($obj -is [System.Array]){ return @($obj) }
  if($obj -is [pscustomobject]){
    if($obj.PSObject.Properties.Name -contains '"'"'items'"'"'){ return @($obj.items) }
    if($obj.PSObject.Properties.Name -contains '"'"'data'"'"'){
      if($obj.data -is [System.Array]){ return @($obj.data) }
      if($obj.data -and ($obj.data.PSObject.Properties.Name -contains '"'"'items'"'"')){ return @($obj.data.items) }
    }
  }
  return @()
}

function Short([string]$s,[int]$n=700){
  if($null -eq $s){ return '"'"''"'"' }
  if($s.Length -le $n){ return $s }
  return $s.Substring(0,$n) + '"'"'...'"'"'
}

$base = '"'"'http://127.0.0.1:8180'"'"'
$api  = "$base/api/v1"

$health = Invoke-Api -Method GET -Url "$base/healthz"
Write-Output "1) GET /healthz -> $($health.Status) $(Short $health.Body 200)"

$login = Invoke-Api -Method POST -Url "$api/auth/login" -Body @{ username='"'"'admin'"'"'; password='"'"'admin123'"'"' }
$token = $null
try {
  $lo = $login.Body | ConvertFrom-Json -Depth 20
  if($lo.token){ $token = $lo.token }
  elseif($lo.access_token){ $token = $lo.access_token }
  elseif($lo.data){ if($lo.data.token){$token=$lo.data.token} elseif($lo.data.access_token){$token=$lo.data.access_token} }
} catch {}
Write-Output "2) POST /api/v1/auth/login -> $($login.Status) token=$([bool]$token)"
if(-not $token){ Write-Output "BLOCKER: login/token failed. body=$(Short $login.Body 260)"; Pop-Location; exit 0 }

$tasksResp = Invoke-Api -Method GET -Url "$api/tasks" -Token $token
$tasks = Get-List $tasksResp.Body
Write-Output "3) GET /api/v1/tasks -> $($tasksResp.Status) count=$($tasks.Count)"
$task = $null
if($tasks.Count -gt 0){
  $task = $tasks[0]
  Write-Output "   using task_id=$($task.id)"
} else {
  $projectsResp = Invoke-Api -Method GET -Url "$api/projects" -Token $token
  $projects = Get-List $projectsResp.Body
  $chosenProject = $null
  $chosenVersion = $null
  foreach($p in $projects){
    $vResp = Invoke-Api -Method GET -Url "$api/projects/$($p.id)/versions" -Token $token
    $versions = Get-List $vResp.Body
    if($versions.Count -gt 0){ $chosenProject = $p; $chosenVersion = $versions[0]; break }
  }
  if(-not $chosenVersion){ Write-Output '"'"'BLOCKER: tasks empty and no existing version available via /projects/{id}/versions.'"'"'; Pop-Location; exit 0 }
  $newTaskResp = Invoke-Api -Method POST -Url "$api/tasks" -Token $token -Body @{ name=('"'"'quick-task-'"'"'+[guid]::NewGuid().ToString('"'"'N'"'"').Substring(0,6)); project_id=$chosenProject.id; version_id=$chosenVersion.id; entry_command='"'"'echo ok'"'"'; cron_expr='"'"''"'"'; node_queue='"'"'default'"'"' }
  Write-Output "   create task -> $($newTaskResp.Status) $(Short $newTaskResp.Body 220)"
  if($newTaskResp.Status -lt 200 -or $newTaskResp.Status -ge 300){ Write-Output '"'"'BLOCKER: failed to create task quickly.'"'"'; Pop-Location; exit 0 }
  try { $task = $newTaskResp.Body | ConvertFrom-Json } catch { Write-Output '"'"'BLOCKER: create-task response parse failed.'"'"'; Pop-Location; exit 0 }
}

if(-not $task.id -or -not $task.project_id -or -not $task.version_id -or -not $task.entry_command){
  Write-Output '"'"'BLOCKER: selected task missing id/project_id/version_id/entry_command.'"'"'
  Pop-Location
  exit 0
}

$nm = '"'"'nq-'"'"' + [guid]::NewGuid().ToString('"'"'N'"'"').Substring(0,8)
$scheduleBody = @{
  name = $nm
  description = '"'"'numeric node_queue check'"'"'
  task_id = $task.id
  project_id = $task.project_id
  version_id = $task.version_id
  entry_command = $task.entry_command
  cron_expr = '"'"''"'"'
  node_queue = 1
  enabled = $false
  order = @{
    name = "$nm-order"
    schedule = @(
      @{ task_id = $task.id; trigger = '"'"'api'"'"'; node = @('"'"'default'"'"') }
    )
  }
}

$sched = Invoke-Api -Method POST -Url "$api/schedules" -Token $token -Body $scheduleBody
Write-Output "4) POST /api/v1/schedules -> $($sched.Status)"
Write-Output "   body=$(Short $sched.Body 900)"
$old = $sched.Body -match '"'"'cannot unmarshal number into Go struct field createScheduleRequest\.node_queue of type string'"'"'
Write-Output "5) old_error_present=$old"
