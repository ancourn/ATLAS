# ATLAS — starter monorepo
## Quickstart (local / VPS)
1. Clone into your VPS and `cd atlas/infra`
2. Build and run:
   docker compose up --build
3. Open http://<VPS_IP>:3000 to see the web app.
Services:
- web -> Next.js frontend (3000)
- id -> identity stub (9100)
- inbox -> unified inbox stub (9200)
- postgres -> 5432
- minio -> 9000
- nats -> 4222