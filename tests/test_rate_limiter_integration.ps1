# ========================================================
# PENGUJIAN RATE LIMITER (INTEGRATION TEST)
# Checklist:
# [ ] Test reguler tetap lolos dengan go test ./...
# [ ] Test rate limiter mengizinkan 2 request dan menolak request ketiga
# [ ] HTTP 429 memiliki header Retry-After
# [ ] Test 20 request konkuren menghasilkan 5 sukses dan 15 terkena limit
# [ ] go test -race -tags=integration berhasil
# ========================================================
$ErrorActionPreference = "Continue"

$repoRoot = (Resolve-Path "$PSScriptRoot\..").Path
Push-Location $repoRoot

Write-Host "`n========================================================" -ForegroundColor Cyan
Write-Host " MEMULAI PENGUJIAN RATE LIMITER (INTEGRATION)" -ForegroundColor Cyan
Write-Host "========================================================`n" -ForegroundColor Cyan

$summary = [ordered]@{
    "Test reguler tetap lolos dengan go test ./..."                                   = $false
    "Test rate limiter mengizinkan 2 request dan menolak request ketiga"              = $false
    "HTTP 429 memiliki header Retry-After"                                            = $false
    "Test 20 request konkuren menghasilkan 5 sukses dan 15 terkena limit"             = $false
    "go test -race -tags=integration berhasil"                                        = $false
}

# --------------------------------------------------------
# Helper: cek koneksi TCP ke endpoint Redis
# --------------------------------------------------------
function Test-TcpPort {
    param(
        [string]$HostName,
        [int]$Port,
        [int]$TimeoutMs = 800
    )

    $client = New-Object System.Net.Sockets.TcpClient
    try {
        $async = $client.BeginConnect($HostName, $Port, $null, $null)
        if (-not $async.AsyncWaitHandle.WaitOne($TimeoutMs)) {
            return $false
        }

        $client.EndConnect($async)
        return $true
    } catch {
        return $false
    } finally {
        $client.Close()
    }
}

function Test-RedisEndpoint {
    param(
        [string]$Addr,
        [int]$TimeoutMs = 800
    )

    if ([string]::IsNullOrWhiteSpace($Addr)) {
        return $false
    }

    $parts = $Addr.Split(":")
    if ($parts.Count -ne 2) {
        return $false
    }

    $port = 0
    if (-not [int]::TryParse($parts[1], [ref]$port)) {
        return $false
    }

    return (Test-TcpPort -HostName $parts[0] -Port $port -TimeoutMs $TimeoutMs)
}

# --------------------------------------------------------
# Menentukan target Redis untuk integration test
# --------------------------------------------------------
$envVars = @{}
if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$') {
            $envVars[$matches[1]] = $matches[2].Trim().Trim('"').Trim("'")
        }
    }
}

if ([string]::IsNullOrWhiteSpace($env:TEST_REDIS_ADDR)) {
    $localAddr = "127.0.0.1:6379"
    $remoteAddr = $envVars["REDIS_ADDR"]

    if (Test-RedisEndpoint -Addr $localAddr) {
        # Prioritaskan Redis lokal (integration test memakai DB 15 secara default).
        $env:TEST_REDIS_ADDR = $localAddr
        if ([string]::IsNullOrWhiteSpace($env:TEST_REDIS_DB)) {
            $env:TEST_REDIS_DB = "15"
        }
        Remove-Item Env:\TEST_REDIS_PASSWORD -ErrorAction SilentlyContinue
    } elseif (-not [string]::IsNullOrWhiteSpace($remoteAddr)) {
        # Fallback ke Redis dari .env. Redis terkelola umumnya hanya mendukung DB 0.
        $env:TEST_REDIS_ADDR = $remoteAddr
        if ([string]::IsNullOrWhiteSpace($env:TEST_REDIS_DB)) {
            $envDb = $envVars["REDIS_DB"]
            if ($envDb -match '^\d+$') {
                $env:TEST_REDIS_DB = $envDb
            } else {
                $env:TEST_REDIS_DB = "0"
            }
        }
        if ([string]::IsNullOrWhiteSpace($env:TEST_REDIS_PASSWORD) -and
            -not [string]::IsNullOrWhiteSpace($envVars["REDIS_PASSWORD"])) {
            $env:TEST_REDIS_PASSWORD = $envVars["REDIS_PASSWORD"]
        }
    } else {
        $env:TEST_REDIS_ADDR = "localhost:6379"
        if ([string]::IsNullOrWhiteSpace($env:TEST_REDIS_DB)) {
            $env:TEST_REDIS_DB = "15"
        }
    }
}

Write-Host "[INFO] Target Redis integration test : $($env:TEST_REDIS_ADDR) (DB: $($env:TEST_REDIS_DB))" -ForegroundColor Gray
if (-not (Test-RedisEndpoint -Addr $env:TEST_REDIS_ADDR)) {
    Write-Host "[WARNING] Redis tidak dapat dijangkau. Integration test kemungkinan akan gagal." -ForegroundColor Yellow
    Write-Host "          Set TEST_REDIS_ADDR/TEST_REDIS_PASSWORD/TEST_REDIS_DB jika perlu." -ForegroundColor Yellow
}

# --------------------------------------------------------
# Checklist 1: Test reguler tetap lolos dengan go test ./...
# --------------------------------------------------------
Write-Host "`n[1/5] Menjalankan 'go test ./...' (test reguler) ..." -ForegroundColor Yellow
$regularOutput = go test ./... 2>&1
$regularCode = $LASTEXITCODE

if ($regularCode -eq 0) {
    Write-Host "  [OK] go test ./... berhasil tanpa error!" -ForegroundColor Green
    $summary["Test reguler tetap lolos dengan go test ./..."] = $true
} else {
    Write-Host "  [FAIL] go test ./... gagal (exit code: $regularCode)" -ForegroundColor Red
    Write-Host "  $regularOutput" -ForegroundColor DarkRed
}

# --------------------------------------------------------
# Checklist 2 & 3: 2 request diizinkan, request ke-3 ditolak,
#                 dan HTTP 429 memiliki header Retry-After
# --------------------------------------------------------
Write-Host "`n[2/5 & 3/5] Menguji limit 2 request + header Retry-After ..." -ForegroundColor Yellow
Write-Host "  Menjalankan TestRateLimiter_EnforcesLimit ..." -ForegroundColor Gray

$enforceOutput = go test -tags=integration -count=1 -v -run "^TestRateLimiter_EnforcesLimit$" ./internal/middleware 2>&1
$enforceCode = $LASTEXITCODE

if ($enforceCode -eq 0 -and ($enforceOutput -match "PASS")) {
    Write-Host "  [OK] Request ke-1 dan ke-2 mendapat HTTP 200, request ke-3 mendapat HTTP 429!" -ForegroundColor Green
    Write-Host "  [OK] Assertion header Retry-After pada HTTP 429 berhasil dilewati!" -ForegroundColor Green
    $summary["Test rate limiter mengizinkan 2 request dan menolak request ketiga"] = $true
    $summary["HTTP 429 memiliki header Retry-After"] = $true
} else {
    Write-Host "  [FAIL] TestRateLimiter_EnforcesLimit gagal (exit code: $enforceCode)" -ForegroundColor Red
    Write-Host "  $enforceOutput" -ForegroundColor DarkRed
}

# --------------------------------------------------------
# Checklist 4: 20 request konkuren => 5 sukses, 15 terkena limit
# --------------------------------------------------------
Write-Host "`n[4/5] Menguji 20 request konkuren (limit 5) ..." -ForegroundColor Yellow
Write-Host "  Menjalankan TestRateLimiter_ConcurrentRequests ..." -ForegroundColor Gray

$concurrentOutput = go test -tags=integration -count=1 -v -run "^TestRateLimiter_ConcurrentRequests$" ./internal/middleware 2>&1
$concurrentCode = $LASTEXITCODE

if ($concurrentCode -eq 0 -and ($concurrentOutput -match "PASS")) {
    Write-Host "  [OK] 20 request konkuren menghasilkan tepat 5 sukses dan 15 HTTP 429!" -ForegroundColor Green
    $summary["Test 20 request konkuren menghasilkan 5 sukses dan 15 terkena limit"] = $true
} else {
    Write-Host "  [FAIL] TestRateLimiter_ConcurrentRequests gagal (exit code: $concurrentCode)" -ForegroundColor Red
    Write-Host "  $concurrentOutput" -ForegroundColor DarkRed
}

# --------------------------------------------------------
# Checklist 5: go test -race -tags=integration berhasil
# --------------------------------------------------------
Write-Host "`n[5/5] Menjalankan 'go test -race -tags=integration ./...' ..." -ForegroundColor Yellow

$previousCgo = $env:CGO_ENABLED
$env:CGO_ENABLED = "1"

$raceOutput = go test -race -tags=integration -count=1 ./... 2>&1
$raceCode = $LASTEXITCODE

if ([string]::IsNullOrWhiteSpace($previousCgo)) {
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
} else {
    $env:CGO_ENABLED = $previousCgo
}

if ($raceCode -eq 0) {
    Write-Host "  [OK] go test -race -tags=integration ./... berhasil tanpa data race!" -ForegroundColor Green
    $summary["go test -race -tags=integration berhasil"] = $true
} else {
    Write-Host "  [FAIL] go test -race -tags=integration ./... gagal (exit code: $raceCode)" -ForegroundColor Red
    Write-Host "  $raceOutput" -ForegroundColor DarkRed

    if ($raceOutput -match "requires cgo|C compiler|gcc|executable file not found") {
        Write-Host "  [INFO] Race detector memerlukan CGO_ENABLED=1 dan compiler C (gcc/mingw-w64) di PATH." -ForegroundColor Yellow
        Write-Host "         Contoh: install MSYS2/MinGW-w64, lalu jalankan ulang script ini." -ForegroundColor Yellow
    }
}

# --------------------------------------------------------
# RINGKASAN HASIL PENGUJIAN
# --------------------------------------------------------
Write-Host "`n========================================================" -ForegroundColor Cyan
Write-Host " RINGKASAN HASIL PENGUJIAN RATE LIMITER" -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan

$allPass = $true
foreach ($key in $summary.Keys) {
    if ($summary[$key]) {
        Write-Host "[x] $key" -ForegroundColor Green
    } else {
        Write-Host "[ ] $key" -ForegroundColor Red
        $allPass = $false
    }
}

Write-Host "========================================================" -ForegroundColor Cyan
if ($allPass) {
    Write-Host " STATUS: SEMUA PENGUJIAN BERHASIL (5/5 PASS)!" -ForegroundColor Green
} else {
    Write-Host " STATUS: ADA PENGUJIAN YANG GAGAL. SILAKAN PERIKSA LOG DI ATAS." -ForegroundColor Red
}
Write-Host "========================================================`n" -ForegroundColor Cyan

Pop-Location

if (-not $allPass) {
    exit 1
}
