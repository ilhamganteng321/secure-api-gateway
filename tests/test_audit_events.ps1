# ========================================================
# PENGUJIAN STATUS IMPLEMENTASI AUDIT EVENTS & LOGGING
# ========================================================
$ErrorActionPreference = "Continue"

Write-Host "`n========================================================" -ForegroundColor Cyan
Write-Host " MEMULAI PENGUJIAN CHECKLIST AUDIT EVENTS" -ForegroundColor Cyan
Write-Host "========================================================`n" -ForegroundColor Cyan

$baseUrl = "http://127.0.0.1:3000"
$dbUrl = "postgres://gateway_app:admin123@localhost:5432/secure_api_gateway?sslmode=disable"

if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match "^\s*DATABASE_URL\s*=\s*(.+)$") {
            $dbUrl = $matches[1].Trim()
        }
    }
}

$summary = [ordered]@{
    "Migration audit_events berhasil diterapkan"               = $false
    "go test ./... dan go vet ./... berhasil"                  = $false
    "Login berhasil tercatat di PostgreSQL"                     = $false
    "Login gagal tercatat tanpa password"                      = $false
    "Request tanpa token tercatat sebagai akses ditolak"       = $false
    "Refresh token reuse tercatat tanpa token mentah"           = $false
    "Audit log tidak menyimpan JWT, password, atau Auth header" = $false
    "Kegagalan menulis audit log dilaporkan lewat slog"        = $false
}

# --------------------------------------------------------
# Checklist 1: Migration audit_events berhasil diterapkan
# --------------------------------------------------------
Write-Host "[1/8] Memeriksa migration tabel 'audit_events' di PostgreSQL ..." -ForegroundColor Yellow
try {
    $checkTableSql = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'audit_events';"
    $tableExists = (& psql $dbUrl -t -A -c $checkTableSql 2>&1).Trim()
    
    if ($tableExists -ne "1") {
        Write-Host "  [INFO] Tabel audit_events belum ada, menerapkan migration..." -ForegroundColor Cyan
        & psql $dbUrl -f "migrations/000003_audit_events.up.sql" | Out-Null
        $tableExists = (& psql $dbUrl -t -A -c $checkTableSql 2>&1).Trim()
    }

    if ($tableExists -eq "1") {
        # Validasi kolom-kolom penting
        $columnsSql = "SELECT column_name FROM information_schema.columns WHERE table_name = 'audit_events' ORDER BY ordinal_position;"
        $columns = (& psql $dbUrl -t -A -c $columnsSql 2>&1) -split "`r?`n"
        $requiredCols = @("actor_user_id", "event_type", "outcome", "request_id", "http_method", "request_path", "status_code", "client_ip", "metadata", "created_at")
        
        $missing = @()
        foreach ($col in $requiredCols) {
            if ($columns -notcontains $col) { $missing += $col }
        }

        if ($missing.Count -eq 0) {
            Write-Host "  [OK] Tabel 'audit_events' dan seluruh kolom wajib berhasil diverifikasi!" -ForegroundColor Green
            $summary["Migration audit_events berhasil diterapkan"] = $true
        } else {
            Write-Host "  [FAIL] Kolom audit_events tidak lengkap. Hilang: $($missing -join ', ')" -ForegroundColor Red
        }
    } else {
        Write-Host "  [FAIL] Tabel 'audit_events' tidak ditemukan di database!" -ForegroundColor Red
    }
} catch {
    Write-Host "  [FAIL] Gagal menghubungkan ke PostgreSQL: $_" -ForegroundColor Red
}

# --------------------------------------------------------
# Checklist 2: go test ./... dan go vet ./... berhasil
# --------------------------------------------------------
Write-Host "`n[2/8] Menguji 'go vet ./...' dan 'go test ./...' ..." -ForegroundColor Yellow
$vetOutput = go vet ./... 2>&1
$vetCode = $LASTEXITCODE

$testOutput = go test ./... 2>&1
$testCode = $LASTEXITCODE

if ($vetCode -eq 0 -and $testCode -eq 0) {
    Write-Host "  [OK] go vet ./... dan go test ./... berhasil tanpa error!" -ForegroundColor Green
    $summary["go test ./... dan go vet ./... berhasil"] = $true
} else {
    Write-Host "  [FAIL] go vet exit code: $vetCode, go test exit code: $testCode" -ForegroundColor Red
    Write-Host "  Detail vet: $vetOutput" -ForegroundColor DarkRed
    Write-Host "  Detail test: $testOutput" -ForegroundColor DarkRed
}

# --------------------------------------------------------
# Checklist 8: Kegagalan menulis audit log dilaporkan lewat slog
# --------------------------------------------------------
Write-Host "`n[3/8] Menguji pelaporan kegagalan audit lewat slog (unit test) ..." -ForegroundColor Yellow
$slogTestOutput = go test -v -run "TestRecordRequest_FailureReportedViaSlog" ./internal/modules/audit/... 2>&1
$slogTestCode = $LASTEXITCODE

if ($slogTestCode -eq 0 -and ($slogTestOutput -match "PASS")) {
    Write-Host "  [OK] Kegagalan menulis audit log diverifikasi dilaporkan via slog tanpa menimpa respons HTTP!" -ForegroundColor Green
    $summary["Kegagalan menulis audit log dilaporkan lewat slog"] = $true
} else {
    Write-Host "  [FAIL] Unit test slog failure gagal! Exit code: $slogTestCode" -ForegroundColor Red
    Write-Host "  $slogTestOutput" -ForegroundColor DarkRed
}

# --------------------------------------------------------
# Memeriksa atau menjalankan server API Gateway
# --------------------------------------------------------
Write-Host "`n[INFO] Memeriksa status HTTP Server di $baseUrl ..." -ForegroundColor Cyan
$serverProc = $null
$serverStartedByTest = $false

try {
    $ready = Invoke-RestMethod -Uri "$baseUrl/health/ready" -Method Get -TimeoutSec 2
    if ($ready.status -eq "ready") {
        Write-Host "  [OK] API Gateway sudah aktif dan siap digunakan." -ForegroundColor Green
    }
} catch {
    $repoRoot = (Resolve-Path "$PSScriptRoot\..").Path
    $serverProc = Start-Process -FilePath "go" -ArgumentList "run", "cmd/api/main.go" -WorkingDirectory $repoRoot -PassThru
    $serverStartedByTest = $true

    for ($i = 0; $i -lt 15; $i++) {
        Start-Sleep -Milliseconds 800
        try {
            $ready = Invoke-RestMethod -Uri "$baseUrl/health/ready" -Method Get -TimeoutSec 1
            if ($ready.status -eq "ready") {
                Write-Host "  [OK] API Gateway berhasil dijalankan (PID: $($serverProc.Id))." -ForegroundColor Green
                break
            }
        } catch {}
    }
}

try {
    $timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $testEmail = "audit_user_${timestamp}@example.com"
    $testPassword = "AuditStrongPassword123!"
    $wrongPassword = "WrongPasswordAuditFail999!"

    # 1. Registrasi user baru
    Write-Host "`n[INFO] Menyiapkan user pengujian: $testEmail ..." -ForegroundColor Cyan
    $regBody = @{
        email    = $testEmail
        password = $testPassword
    } | ConvertTo-Json

    $regResp = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/register" -Method Post -ContentType "application/json" -Body $regBody
    $userID = $regResp.user.id
    Write-Host "  [OK] User terdaftar dengan ID: $userID" -ForegroundColor Green

    # --------------------------------------------------------
    # Checklist 3: Login berhasil tercatat di PostgreSQL
    # --------------------------------------------------------
    Write-Host "`n[4/8] Menguji pencatatan Login BERHASIL di PostgreSQL ..." -ForegroundColor Yellow
    $loginBody = @{
        email    = $testEmail
        password = $testPassword
    } | ConvertTo-Json

    $loginResp = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/login" -Method Post -ContentType "application/json" -Body $loginBody
    $accessToken = $loginResp.access_token
    $refreshToken = $loginResp.refresh_token

    Start-Sleep -Milliseconds 500

    $loginSuccessSql = "SELECT id, actor_user_id, event_type, outcome, status_code FROM audit_events WHERE event_type = 'auth.login' AND outcome = 'success' AND actor_user_id = $userID ORDER BY id DESC LIMIT 1;"
    $loginAuditRow = (& psql $dbUrl -t -A -F"|" -c $loginSuccessSql 2>&1).Trim()

    if ($loginAuditRow) {
        $parts = $loginAuditRow -split '\|'
        $auditId = $parts[0]
        $actorId = $parts[1]
        $evType = $parts[2]
        $outcome = $parts[3]
        $statusCode = $parts[4]

        if ($actorId -eq "$userID" -and $evType -eq "auth.login" -and $outcome -eq "success" -and $statusCode -eq "200") {
            Write-Host "  [OK] Event 'auth.login' berhasil tercatat di PostgreSQL! (ID: $auditId, Actor: $actorId, Status: $statusCode)" -ForegroundColor Green
            $summary["Login berhasil tercatat di PostgreSQL"] = $true
        } else {
            Write-Host "  [FAIL] Field audit login tidak sesuai: $loginAuditRow" -ForegroundColor Red
        }
    } else {
        Write-Host "  [FAIL] Event audit login sukses tidak ditemukan di database!" -ForegroundColor Red
    }

    # --------------------------------------------------------
    # Checklist 4: Login gagal tercatat tanpa password
    # --------------------------------------------------------
    Write-Host "`n[5/8] Menguji pencatatan Login GAGAL tanpa password ..." -ForegroundColor Yellow
    $failLoginBody = @{
        email    = $testEmail
        password = $wrongPassword
    } | ConvertTo-Json

    try {
        Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/login" -Method Post -ContentType "application/json" -Body $failLoginBody | Out-Null
        Write-Host "  [FAIL] Seharusnya gagal login tetapi response 200 OK" -ForegroundColor Red
    } catch {
        # Diharapkan 401
    }

    Start-Sleep -Milliseconds 500

    $loginFailSql = "SELECT id, event_type, outcome, status_code, metadata::text FROM audit_events WHERE event_type = 'auth.login' AND outcome = 'failure' ORDER BY id DESC LIMIT 1;"
    $loginFailRow = (& psql $dbUrl -t -A -F"|" -c $loginFailSql 2>&1).Trim()

    if ($loginFailRow) {
        $parts = $loginFailRow -split '\|'
        $failAuditId = $parts[0]
        $failEvType = $parts[1]
        $failOutcome = $parts[2]
        $failStatusCode = $parts[3]
        $failMetadata = $parts[4]

        $passwordLeaked = $failMetadata.Contains($wrongPassword) -or $loginFailRow.Contains($wrongPassword)

        if ($failStatusCode -eq "401" -and $failMetadata.Contains("invalid_credentials") -and (-not $passwordLeaked)) {
            Write-Host "  [OK] Event login gagal tercatat dengan benar (Status: 401, Reason: invalid_credentials)" -ForegroundColor Green
            Write-Host "  [OK] Password TIDAK tersimpan pada metadata atau record audit!" -ForegroundColor Green
            $summary["Login gagal tercatat tanpa password"] = $true
        } else {
            Write-Host "  [FAIL] Data audit login gagal tidak valid atau password bocor! Row: $loginFailRow" -ForegroundColor Red
        }
    } else {
        Write-Host "  [FAIL] Event audit login gagal tidak ditemukan di database!" -ForegroundColor Red
    }

    # --------------------------------------------------------
    # Checklist 5: Request tanpa token tercatat sebagai akses ditolak
    # --------------------------------------------------------
    Write-Host "`n[6/8] Menguji Request tanpa token dicatat sebagai akses ditolak ..." -ForegroundColor Yellow
    try {
        Invoke-RestMethod -Uri "$baseUrl/api/v1/me" -Method Get | Out-Null
        Write-Host "  [FAIL] Seharusnya 401 Unauthorized tanpa token" -ForegroundColor Red
    } catch {
        # Diharapkan 401
    }

    Start-Sleep -Milliseconds 500

    $noTokenSql = "SELECT id, event_type, outcome, status_code, metadata::text FROM audit_events WHERE event_type = 'auth.access_denied' AND outcome = 'denied' AND metadata::text LIKE '%missing_or_invalid_authorization_header%' ORDER BY id DESC LIMIT 1;"
    $noTokenRow = (& psql $dbUrl -t -A -F"|" -c $noTokenSql 2>&1).Trim()

    if ($noTokenRow) {
        $parts = $noTokenRow -split '\|'
        $ntId = $parts[0]
        $ntEv = $parts[1]
        $ntOutcome = $parts[2]
        $ntStatus = $parts[3]

        if ($ntStatus -eq "401" -and $ntOutcome -eq "denied") {
            Write-Host "  [OK] Akses tanpa token tercatat sebagai 'auth.access_denied' (Status: 401, Reason: missing_or_invalid_authorization_header)!" -ForegroundColor Green
            $summary["Request tanpa token tercatat sebagai akses ditolak"] = $true
        } else {
            Write-Host "  [FAIL] Field audit tidak sesuai: $noTokenRow" -ForegroundColor Red
        }
    } else {
        Write-Host "  [FAIL] Event akses tanpa token tidak tercatat di database!" -ForegroundColor Red
    }

    # --------------------------------------------------------
    # Checklist 6: Refresh token reuse tercatat tanpa token mentah
    # --------------------------------------------------------
    Write-Host "`n[7/8] Menguji Refresh Token Reuse tercatat tanpa token mentah ..." -ForegroundColor Yellow
    
    # Refresh sah pertama (merotasi token)
    $refreshBody = @{ refresh_token = $refreshToken } | ConvertTo-Json
    $refResp1 = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/refresh" -Method Post -ContentType "application/json" -Body $refreshBody
    $newAccessToken = $refResp1.access_token
    $newRefreshToken = $refResp1.refresh_token

    # Ulangi request dengan REFRESH TOKEN LAMA (seharusnya dideteksi sebagai reuse)
    try {
        Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/refresh" -Method Post -ContentType "application/json" -Body $refreshBody | Out-Null
        Write-Host "  [FAIL] Token lama yang sudah dirotasi seharusnya ditolak!" -ForegroundColor Red
    } catch {
        # Diharapkan 401 refresh_token_reuse_detected
    }

    Start-Sleep -Milliseconds 500

    $reuseSql = "SELECT id, event_type, outcome, status_code, metadata::text FROM audit_events WHERE event_type = 'auth.refresh_reuse_detected' AND outcome = 'denied' ORDER BY id DESC LIMIT 1;"
    $reuseRow = (& psql $dbUrl -t -A -F"|" -c $reuseSql 2>&1).Trim()

    if ($reuseRow) {
        $parts = $reuseRow -split '\|'
        $ruId = $parts[0]
        $ruStatus = $parts[3]
        $ruMeta = $parts[4]

        $tokenLeaked = $reuseRow.Contains($refreshToken) -or $reuseRow.Contains($newRefreshToken)

        if ($ruStatus -eq "401" -and $ruMeta.Contains("rotated_token_reused") -and (-not $tokenLeaked)) {
            Write-Host "  [OK] Refresh token reuse berhasil dideteksi dan dicatat (Status: 401, Reason: rotated_token_reused)!" -ForegroundColor Green
            Write-Host "  [OK] Token mentah TIDAK tersimpan pada metadata atau record audit!" -ForegroundColor Green
            $summary["Refresh token reuse tercatat tanpa token mentah"] = $true
        } else {
            Write-Host "  [FAIL] Data audit reuse tidak valid atau token mentah bocor! Row: $reuseRow" -ForegroundColor Red
        }
    } else {
        Write-Host "  [FAIL] Event refresh token reuse tidak ditemukan di database!" -ForegroundColor Red
    }

    # --------------------------------------------------------
    # Checklist 7: Audit log tidak menyimpan JWT, password, atau Authorization header
    # --------------------------------------------------------
    Write-Host "`n[8/8] Memeriksa bahwa audit log tidak membocorkan JWT, Password, atau Auth Header ..." -ForegroundColor Yellow
    $recentAuditSql = "SELECT id, request_path, metadata::text FROM audit_events ORDER BY id DESC LIMIT 20;"
    $recentRows = & psql $dbUrl -t -A -c $recentAuditSql 2>&1

    $leakDetected = $false
    $leakReasons = @()

    foreach ($row in $recentRows) {
        if ($row.Contains("Bearer ")) {
            $leakDetected = $true
            $leakReasons += "Header Authorization / Bearer ditemukan pada log audit"
        }
        if ($testPassword -and $row.Contains($testPassword)) {
            $leakDetected = $true
            $leakReasons += "Password user ditemukan pada log audit"
        }
        if ($wrongPassword -and $row.Contains($wrongPassword)) {
            $leakDetected = $true
            $leakReasons += "Password salah ditemukan pada log audit"
        }
        if ($refreshToken -and $row.Contains($refreshToken)) {
            $leakDetected = $true
            $leakReasons += "Refresh token ditemukan pada log audit"
        }
        if ($accessToken -and $row.Contains($accessToken)) {
            $leakDetected = $true
            $leakReasons += "Access token JWT ditemukan pada log audit"
        }
        if ($row -match "ey[A-Za-z0-9-_=]{20,}\.[A-Za-z0-9-_=]{20,}") {
            $leakDetected = $true
            $leakReasons += "Format token JWT terdeteksi pada log audit"
        }
    }

    if (-not $leakDetected) {
        Write-Host "  [OK] Log audit aman: Bersih dari JWT, Password mentah, maupun Header Authorization!" -ForegroundColor Green
        $summary["Audit log tidak menyimpan JWT, password, atau Auth header"] = $true
    } else {
        Write-Host "  [FAIL] Kebocoran data sensitif terdeteksi pada log audit: $($leakReasons -join '; ')" -ForegroundColor Red
    }

} finally {
    if ($serverStartedByTest -and $serverProc) {
        Write-Host "`n[INFO] Menghentikan API Gateway latar belakang (PID: $($serverProc.Id)) ..." -ForegroundColor Cyan
        Stop-Process -Id $serverProc.Id -Force -ErrorAction SilentlyContinue
    }
}

# --------------------------------------------------------
# RINGKASAN HASIL PENGUJIAN
# --------------------------------------------------------
Write-Host "`n========================================================" -ForegroundColor Cyan
Write-Host " RINGKASAN HASIL PENGUJIAN AUDIT LOGGING" -ForegroundColor Cyan
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
    Write-Host " STATUS: SEMUA PENGUJIAN BERHASIL (8/8 PASS)!" -ForegroundColor Green
} else {
    Write-Host " STATUS: ADA PENGUJIAN YANG GAGAL. SILAKAN PERIKSA LOG DI ATAS." -ForegroundColor Red
}
Write-Host "========================================================`n" -ForegroundColor Cyan
