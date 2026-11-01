# Secure API Gateway - Runbook

## 1. Gateway tidak bisa diakses

1. Periksa proses gateway.
2. Periksa /health/live.
3. Periksa application logs.
4. Periksa port dan konfigurasi server.
5. Periksa dependency readiness.

## 2. PostgreSQL unavailable

1. Periksa koneksi PostgreSQL.
2. Periksa connection pool.
3. Periksa database logs.
4. Periksa kapasitas disk.
5. Pulihkan koneksi dan verifikasi readiness.

## 3. Redis unavailable

1. Periksa Redis connectivity.
2. Periksa memory usage.
3. Periksa rate limiter behavior.
4. Terapkan kebijakan fail-open/fail-closed.
5. Verifikasi request rate setelah recovery.

## 4. Upstream timeout

1. Periksa upstream health.
2. Periksa upstream latency.
3. Periksa circuit breaker state.
4. Periksa distributed trace.
5. Pulihkan upstream dan verifikasi traffic.

## 5. Database recovery

1. Siapkan database restore target.
2. Restore backup.
3. Jalankan migration jika diperlukan.
4. Verifikasi jumlah record dan integritas data.
5. Jalankan smoke test.
6. Alihkan traffic setelah validasi.
