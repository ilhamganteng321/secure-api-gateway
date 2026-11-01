# Script pengujian: User yang dinonaktifkan ditolak meskipun access token masih aktif
$ErrorActionPreference = "Stop"

$baseUrl = "http://localhost:3000"
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$testEmail = "inactive_test_$timestamp@example.com"
$testPassword = "PasswordSangatKuat123!"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host " PENGUJIAN USER INAKTIF (SECURE API GATEWAY)" -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Registrasi user baru
Write-Host "`n[1] Mendaftarkan user baru: $testEmail ..." -ForegroundColor Yellow
$registerBody = @{
    email    = $testEmail
    password = $testPassword
} | ConvertTo-Json

$regResp = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/register" `
    -Method Post `
    -ContentType "application/json" `
    -Body $registerBody

$userID = $regResp.user.id
Write-Host "    -> Berhasil daftar! User ID: $userID" -ForegroundColor Green

# 2. Login untuk mendapatkan access token & refresh token
Write-Host "`n[2] Login user untuk mendapatkan access token ..." -ForegroundColor Yellow
$loginBody = @{
    email    = $testEmail
    password = $testPassword
} | ConvertTo-Json

$loginResp = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $loginBody

$accessToken = $loginResp.access_token
$refreshToken = $loginResp.refresh_token
Write-Host "    -> Login berhasil!" -ForegroundColor Green
Write-Host "    -> Access Token didapatkan (Expires in: $($loginResp.expires_in)s)" -ForegroundColor Gray

# 3. Akses endpoint terproteksi /api/v1/me saat user masih AKTIF
Write-Host "`n[3] Menguji GET /api/v1/me saat user masih AKTIF (is_active = TRUE) ..." -ForegroundColor Yellow
$headers = @{
    Authorization = "Bearer $accessToken"
}

try {
    $meResp = Invoke-RestMethod -Uri "$baseUrl/api/v1/me" -Method Get -Headers $headers
    Write-Host "    -> Status: HTTP 200 OK" -ForegroundColor Green
    Write-Host "    -> Response: user_id = $($meResp.user_id)" -ForegroundColor Green
} catch {
    Write-Host "    -> Gagal mengakses endpoint terproteksi: $_" -ForegroundColor Red
    exit 1
}

# 4. Nonaktifkan user di database PostgreSQL
Write-Host "`n[4] Menonaktifkan user di database (UPDATE users SET is_active = FALSE) ..." -ForegroundColor Yellow
$dbUrl = "postgres://gateway_app:admin123@localhost:5432/secure_api_gateway?sslmode=disable"
$sql = "UPDATE users SET is_active = FALSE WHERE id = $userID;"
& psql $dbUrl -c $sql | Out-Null
Write-Host "    -> User ID $userID berhasil dinonaktifkan di database!" -ForegroundColor Green

# 5. Uji kembali akses GET /api/v1/me dengan access token yang sama
Write-Host "`n[5] Menguji GET /api/v1/me dengan Access Token yang SAMA (belum expired) ..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/me" -Method Get -Headers $headers
    Write-Host "    -> PERINGATAN: Endpoint seharusnya menolak user inaktif, tapi mendapat response sukses!" -ForegroundColor Red
} catch {
    $statusCode = 0
    if ($_.Exception.Response) {
        $statusCode = [int]$_.Exception.Response.StatusCode
    }
    $responseBody = $_.ErrorDetails.Message
    if (-not $responseBody -and $_.Exception.Response) {
        $stream = $_.Exception.Response.GetResponseStream()
        if ($stream) {
            $stream.Position = 0
            $reader = New-Object System.IO.StreamReader($stream)
            $responseBody = $reader.ReadToEnd()
        }
    }

    Write-Host "    -> Status Code : HTTP $statusCode" -ForegroundColor Green
    Write-Host "    -> Response Body: $responseBody" -ForegroundColor Green

    if ($statusCode -eq 401 -and $responseBody -like '*user_inactive*') {
        Write-Host "    -> [SUKSES] User inaktif berhasil ditolak dengan HTTP 401 dan body 'user_inactive'!" -ForegroundColor Green
    } else {
        Write-Host "    -> [PERIKSA] Status: $statusCode, Body: $responseBody" -ForegroundColor Yellow
    }
}

# 6. Uji refresh token saat user inaktif
Write-Host "`n[6] Menguji POST /api/v1/auth/refresh saat user inaktif ..." -ForegroundColor Yellow
$refreshBody = @{
    refresh_token = $refreshToken
} | ConvertTo-Json

try {
    $resp = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/refresh" `
        -Method Post `
        -ContentType "application/json" `
        -Body $refreshBody
    Write-Host "    -> PERINGATAN: Refresh token seharusnya ditolak!" -ForegroundColor Red
} catch {
    $refStatusCode = 0
    if ($_.Exception.Response) {
        $refStatusCode = [int]$_.Exception.Response.StatusCode
    }
    $refBody = $_.ErrorDetails.Message

    Write-Host "    -> Status Code : HTTP $refStatusCode" -ForegroundColor Green
    Write-Host "    -> Response Body: $refBody" -ForegroundColor Green
    if ($refStatusCode -eq 401) {
        Write-Host "    -> [SUKSES] Refresh token juga ditolak saat user inaktif!" -ForegroundColor Green
    }
}

Write-Host "`n==========================================================" -ForegroundColor Cyan
Write-Host " KESIMPULAN: PENGUJIAN SELESAI & BERHASIL" -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan
