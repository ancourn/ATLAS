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
- docs -> collaborative docs service (9300)
- postgres -> 5432
- minio -> 9000
- nats -> 4222

## Features
- **Real-time Collaborative Docs**: WebSocket-powered document editing with CRDT synchronization
- **Unified Inbox**: Message aggregation service with NATS eventing
- **Identity Service**: Basic authentication and user management
- **Microservices Architecture**: Scalable Go services with Docker Compose

## Testing
- Landing page: http://localhost:3000
- Collaborative docs: http://localhost:3000/docs
- Service health endpoints:
  - ID service: http://localhost:9100/health
  - Inbox service: http://localhost:9200/health
  - Docs service: http://localhost:9300/health