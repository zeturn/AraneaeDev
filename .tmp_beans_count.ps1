$viewsRoot = Join-Path (Get-Location) 'Front/src/views'
$routerFile = Join-Path (Get-Location) 'Front/src/router/index.js'
$importRegex = '(?m)^\s*import\b.*?/components/BeansDesign/'
$allVueFiles = Get-ChildItem -Path $viewsRoot -Recurse -File -Filter '*.vue'
$directImportFiles = @($allVueFiles | Where-Object { Select-String -Path $_.FullName -Pattern $importRegex -Quiet })
$nonImportFiles = @($allVueFiles | Where-Object { $_.FullName -notin $directImportFiles.FullName })
$total = $allVueFiles.Count
$direct = $directImportFiles.Count
$percent = if ($total -gt 0) { [math]::Round(($direct / $total) * 100, 2) } else { 0 }

$routerText = Get-Content -Path $routerFile -Raw
$importMap = @{}
[regex]::Matches($routerText, '(?m)^\s*import\s+([A-Za-z_$][\w$]*)\s+from\s+["'']([^"'']+)["'']') | ForEach-Object {
  $importMap[$_.Groups[1].Value] = $_.Groups[2].Value
}
$routePaths = New-Object System.Collections.Generic.List[string]
[regex]::Matches($routerText, 'component\s*:\s*\(\s*[^)]*=>\s*import\(\s*(?:/\*[\s\S]*?\*/\s*)?["'']([^"'']+)["'']\s*\)\s*\)') | ForEach-Object {
  [void]$routePaths.Add($_.Groups[1].Value)
}
[regex]::Matches($routerText, 'component\s*:\s*([A-Za-z_$][\w$]*)') | ForEach-Object {
  $name = $_.Groups[1].Value
  if ($importMap.ContainsKey($name)) { [void]$routePaths.Add($importMap[$name]) }
}

function Resolve-RoutePath([string]$p, [string]$routerDir, [string]$srcDir) {
  if ([string]::IsNullOrWhiteSpace($p)) { return $null }
  $candidate = $null
  if ($p.StartsWith('@/')) {
    $candidate = Join-Path $srcDir ($p.Substring(2) -replace '/', [IO.Path]::DirectorySeparatorChar)
  } elseif ($p.StartsWith('./') -or $p.StartsWith('../')) {
    $candidate = [IO.Path]::GetFullPath((Join-Path $routerDir ($p -replace '/', [IO.Path]::DirectorySeparatorChar)))
  } else {
    return $null
  }

  $try = @()
  if ([IO.Path]::HasExtension($candidate)) {
    $try += $candidate
  } else {
    $try += "$candidate.vue"
    $try += (Join-Path $candidate 'index.vue')
    $try += "$candidate.js"
  }
  foreach ($t in $try) { if (Test-Path $t) { return (Resolve-Path $t).Path } }
  return $null
}

$srcDir = Join-Path (Get-Location) 'Front/src'
$routerDir = Split-Path -Parent $routerFile
$routeFilesResolved = @($routePaths | ForEach-Object { Resolve-RoutePath $_ $routerDir $srcDir } | Where-Object { $_ } | Select-Object -Unique)
$routeFilesWithBeans = @($routeFilesResolved | Where-Object { Select-String -Path $_ -Pattern $importRegex -Quiet })

"TOTAL_VUE_FILES=$total"
"VUE_WITH_DIRECT_BEANSDESIGN_IMPORT=$direct"
"VUE_WITH_DIRECT_BEANSDESIGN_IMPORT_PERCENT=$percent"
"VUE_WITHOUT_DIRECT_BEANSDESIGN_IMPORT_COUNT=$($nonImportFiles.Count)"
"ROUTE_COMPONENT_FILES_REFERENCED=$($routeFilesResolved.Count)"
"ROUTE_COMPONENT_FILES_WITH_DIRECT_BEANSDESIGN_IMPORT=$($routeFilesWithBeans.Count)"
"FILES_WITHOUT_DIRECT_BEANSDESIGN_IMPORT:"
$nonImportFiles | ForEach-Object { $_.FullName.Substring(((Get-Location).Path.Length + 1)) }
