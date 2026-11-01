$body = @{
    email = "ilham.test@example.com"
    password = "WrongPasswordForRateLimitTest!"
} | ConvertTo-Json

1..6 | ForEach-Object {
    try {
        $response = Invoke-WebRequest `
            -Method Post `
            -Uri "http://localhost:3000/api/v1/auth/login" `
            -ContentType "application/json" `
            -Body $body

        Write-Host "Request $_ : HTTP $([int]$response.StatusCode)"
    } catch {
        $status = [int]$_.Exception.Response.StatusCode
        Write-Host "Request $_ : HTTP $status"
    }
}
