# Performance Report

Baseline to be updated after running Go benchmarks and k6 load tests.

## Benchmark command

```powershell
go test ./internal/gateway -run '^$' -bench . -benchmem -count=5
```

## k6 command

```powershell
$env:API_KEY = "YOUR_TEST_API_KEY"
k6 run tests/load/gateway.js
```

## Targets

- Health endpoint p95: < 50 ms
- Gateway proxy p95: < 300 ms
- Error rate under normal load: < 1%
- Race conditions: 0
- Goroutine leak: none

## Notes

Record RPS, p50, p95, p99, error rate, CPU, memory, and goroutine count here after each run.
