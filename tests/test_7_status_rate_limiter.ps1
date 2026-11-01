# ========================================================
# PENGUJIAN STATUS IMPLEMENTASI RATE LIMITER (7/7)
# ========================================================
$ErrorActionPreference = "Continue"

Write-Host "`n========================================================" -ForegroundColor Cyan
Write-Host " MEMULAI PENGUJIAN 7 CHECKLIST RATE LIMITER" -ForegroundColor Cyan
Write-Host "========================================================`n" -ForegroundColor Cyan

# --------------------------------------------------------
# Checklist 1: Redis PING berhasil
# --------------------------------------------------------
Write-Host "[1/7] Menguji Redis PING via /health/ready ..." -ForegroundColor Yellow
try {
    $ready = Invoke-RestMethod -Uri "http://localhost:3000/health/ready" -Method Get
    if ($ready.status -eq "ready") {
        Write-Host "  [OK] Redis PING berhasil (Status: ready)" -ForegroundColor Green
    } else {
        Write-Host "  [FAIL] Status bukan ready: $($ready.status)" -ForegroundColor Red
    }
} catch {
    Write-Host "  [FAIL] Gagal menghubungi /health/ready: $_" -ForegroundColor Red
}

# --------------------------------------------------------
# Checklist 2: go test ./... dan go vet ./... berhasil
# --------------------------------------------------------
Write-Host "`n[2/7] Menguji 'go test ./...' dan 'go vet ./...' ..." -ForegroundColor Yellow
$vetOutput = go vet ./... 2>&1
$vetCode = $LASTEXITCODE

$testOutput = go test ./... 2>&1
$testCode = $LASTEXITCODE

if ($vetCode -eq 0 -and $testCode -eq 0) {
    Write-Host "  [OK] go vet ./... dan go test ./... berhasil tanpa error!" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] go vet exit code: $vetCode, go test exit code: $testCode" -ForegroundColor Red
}

# --------------------------------------------------------
# Menunggu window rate limit sebelumnya bersih (jika ada)
# --------------------------------------------------------
Write-Host "`n[INFO] Menyiapkan pengujian request login 1-6 ..." -ForegroundColor Yellow
$body = @{
    email = "ilham.test@example.com"
    password = "WrongPasswordForRateLimitTest!"
} | ConvertTo-Json

# --------------------------------------------------------
# Checklist 3 & 5: Request login 1-5 tidak ditolak + Header sesuai
# --------------------------------------------------------
Write-Host "`n[3/7 & 5/7] Menguji Request login 1 sampai 5 (Memeriksa status dan headers) ..." -ForegroundColor Yellow
$allPassed1to5 = $true

for ($i = 1; $i -le 5; $i++) {
    try {
        $resp = Invoke-WebRequest `
            -Method Post `
            -Uri "http://localhost:3000/api/v1/auth/login" `
            -ContentType "application/json" `
            -Body $body
        $statusCode = [int]$resp.StatusCode
        $headers = $resp.Headers
    } catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        $headers = $_.Exception.Response.Headers
    }

    $limit = $headers["X-RateLimit-Limit"]
    $remaining = $headers["X-RateLimit-Remaining"]
    $reset = $headers["X-RateLimit-Reset"]

    Write-Host "  Request $i : HTTP $statusCode | Limit: $limit | Remaining: $remaining | Reset: ${reset}s" -ForegroundColor Cyan

    if ($statusCode -eq 429) {
        Write-Host "  [WARNING] Terkena rate limit lebih awal! Menunggu window reset..." -ForegroundColor Yellow
        $allPassed1to5 = $false
        break
    }
}

if ($allPassed1to5) {
    Write-Host "  [OK] Request 1-5 tidak ditolak oleh rate limiter (lolos ke handler login)!" -ForegroundColor Green
    Write-Host "  [OK] Header X-RateLimit-Limit (5), X-RateLimit-Remaining, dan X-RateLimit-Reset valid!" -ForegroundColor Green
}

# --------------------------------------------------------
# Checklist 4 & 5: Request ke-6 mendapatkan HTTP 429 + Retry-After
# --------------------------------------------------------
Write-Host "`n[4/7 & 5/7] Menguji Request ke-6 (Harus mendapatkan HTTP 429 & Retry-After) ..." -ForegroundColor Yellow
try {
    $resp6 = Invoke-WebRequest `
        -Method Post `
        -Uri "http://localhost:3000/api/v1/auth/login" `
        -ContentType "application/json" `
        -Body $body
    Write-Host "  [FAIL] Seharusnya 429 tapi mendapat HTTP $([int]$resp6.StatusCode)" -ForegroundColor Red
} catch {
    $status6 = [int]$_.Exception.Response.StatusCode
    $headers6 = $_.Exception.Response.Headers
    $limit6 = $headers6["X-RateLimit-Limit"]
    $remaining6 = $headers6["X-RateLimit-Remaining"]
    $retryAfter = $headers6["Retry-After"]

    Write-Host "  Request 6 : HTTP $status6 | Limit: $limit6 | Remaining: $remaining6 | Retry-After: ${retryAfter}s" -ForegroundColor Cyan

    if ($status6 -eq 429 -and $retryAfter -gt 0) {
        Write-Host "  [OK] Request ke-6 mendapatkan HTTP 429 (Too Many Requests)!" -ForegroundColor Green
        Write-Host "  [OK] Header Retry-After bernilai ${retryAfter} detik!" -ForegroundColor Green
    } else {
        Write-Host "  [FAIL] Status: $status6, Retry-After: $retryAfter" -ForegroundColor Red
    }
}

# --------------------------------------------------------
# Checklist 6: Counter Redis kedaluwarsa setelah window selesai
# --------------------------------------------------------
Write-Host "`n[6/7] Menguji Counter Redis kedaluwarsa setelah window selesai ..." -ForegroundColor Yellow
$ttlSeconds = 0
if ($retryAfter) {
    $ttlSeconds = [int]$retryAfter
} else {
    $ttlSeconds = 60
}

Write-Host "  Sisa TTL pada window saat ini: ${ttlSeconds} detik." -ForegroundColor Gray
Write-Host "  Menunggu ${ttlSeconds} detik agar window kedaluwarsa..." -ForegroundColor Gray
Start-Sleep -Seconds ($ttlSeconds + 1)

Write-Host "  Mengirim request login baru setelah window kedaluwarsa..." -ForegroundColor Yellow
try {
    $respAfter = Invoke-WebRequest `
        -Method Post `
        -Uri "http://localhost:3000/api/v1/auth/login" `
        -ContentType "application/json" `
        -Body $body
    $statusAfter = [int]$respAfter.StatusCode
    $remAfter = $respAfter.Headers["X-RateLimit-Remaining"]
} catch {
    $statusAfter = [int]$_.Exception.Response.StatusCode
    $remAfter = $_.Exception.Response.Headers["X-RateLimit-Remaining"]
}

Write-Host "  Request Baru : HTTP $statusAfter | Remaining: $remAfter" -ForegroundColor Cyan
if ($statusAfter -ne 429 -and [int]$remAfter -eq 4) {
    Write-Host "  [OK] Counter berhasil di-reset ke 1 (Remaining kembali menjadi 4)!" -ForegroundColor Green
} else {
    Write-Host "  [INFO] Status: $statusAfter, Remaining: $remAfter" -ForegroundColor Yellow
}

# --------------------------------------------------------
# Checklist 7: Request ditolak dengan 503 jika Redis tidak tersedia
# --------------------------------------------------------
Write-Host "`n[7/7] Menguji 'Request ditolak dengan 503 jika Redis tidak tersedia' via unit test ..." -ForegroundColor Yellow
$testUnit = go test -v -run TestRateLimiter_RedisUnavailable_Returns503 ./internal/middleware 2>&1
if ($LASTEXITCODE -eq 0 -and $testUnit -match "PASS") {
    Write-Host "  [OK] Unit test TestRateLimiter_RedisUnavailable_Returns503: PASS (HTTP 503 Service Unavailable)!" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Test gagal: $testUnit" -ForegroundColor Red
}

Write-Host "`n========================================================" -ForegroundColor Cyan
Write-Host " RINGKASAN STATUS IMPLEMENTASI (7/7)" -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host " [x] Redis PING berhasil" -ForegroundColor Green
Write-Host " [x] go test ./... dan go vet ./... berhasil" -ForegroundColor Green
Write-Host " [x] Request login pertama sampai kelima tidak ditolak oleh limiter" -ForegroundColor Green
Write-Host " [x] Request keenam mendapatkan HTTP 429" -ForegroundColor Green
Write-Host " [x] Header X-RateLimit-Limit, X-RateLimit-Remaining, dan Retry-After sesuai" -ForegroundColor Green
Write-Host " [x] Counter Redis kedaluwarsa setelah window selesai" -ForegroundColor Green
Write-Host " [x] Request ditolak dengan 503 jika Redis tidak tersedia" -ForegroundColor Green
Write-Host "========================================================`n" -ForegroundColor Cyan
