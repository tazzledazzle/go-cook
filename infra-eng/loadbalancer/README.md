# LoadBalancer

### Implemented:

- Reverse-proxy routing to a configurable backend pool
- Round-robin selection with correct concurrent-safe retry logic (the bug we just fixed)
- Active health checks (periodic /healthz polling)
- Passive health checks (immediate failure detection via ErrorHandler)
- Graceful shutdown with connection draining
- Prometheus-style metrics per backend (requests, errors, latency, alive state)
- Validated under real concurrent load with hey, including a live backend failure