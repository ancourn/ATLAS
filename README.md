# ATLAS — Enhanced Collaborative Platform
## Quickstart (local / VPS)
1. Clone into your VPS and `cd atlas/infra`
2. Build and run:
   docker compose up --build
3. Open http://<VPS_IP>:3000 to see the web app.

## Services
- **web** -> Next.js frontend (3000)
- **id** -> Enhanced identity service with JWT auth (9100)
- **inbox** -> Unified inbox service with NATS (9200)
- **docs** -> CRDT collaborative docs with Postgres (9300)
- **ai-router** -> AI copilot service (9400)
- **postgres** -> Persistent database (5432)
- **minio** -> Object storage (9000)
- **nats** -> Message bus (4222)

## ✨ New Features

### 🔐 Enhanced Authentication & Identity
- **JWT-based authentication** with secure token handling
- **User registration and login** with email/password
- **Session management** with automatic token refresh
- **User permissions** for document access control
- **Demo mode** for quick testing (demo@atlas.local / demo123)

### 🤖 AI-Powered Features
- **Inbox Copilot**: Automatic message summarization and reply suggestions
- **Document Intelligence**: AI-powered document analysis and key point extraction
- **Smart Search**: AI-enhanced document search with relevance ranking
- **Sentiment Analysis**: Understand tone and context of communications
- **Processing Metrics**: Track AI performance and processing times

### 📝 Enhanced CRDT Documents
- **Real-time Collaboration**: WebSocket-powered live editing
- **PostgreSQL Persistence**: Reliable document storage with versioning
- **Conflict Resolution**: CRDT-based update handling
- **User Permissions**: Granular access control per document
- **Version Control**: Track document changes and history
- **Auto-reconnection**: Seamless connectivity recovery

### 🏗️ Microservices Architecture
- **Service Isolation**: Each service runs independently
- **Database Integration**: Shared PostgreSQL with proper schemas
- **Event-Driven**: NATS message bus for service communication
- **Scalable Design**: Horizontal scaling capabilities
- **Health Monitoring**: Service health endpoints for observability

## 🧪 Testing & Development

### Web Application
- **Landing Page**: http://localhost:3000
- **Collaborative Docs**: http://localhost:3000/docs
- **Authentication**: Full login/register flow with demo mode

### API Endpoints
- **Identity Service**: http://localhost:9100
  - `POST /api/register` - User registration
  - `POST /api/login` - User login
  - `GET /api/whoami` - Current user info
  - `GET /api/user` - Protected user data
  - `GET /health` - Service health

- **Docs Service**: http://localhost:9300
  - `GET /api/documents` - List user documents
  - `POST /api/documents` - Create new document
  - `GET /api/documents/{id}` - Get document
  - `PUT /api/documents/{id}` - Update document
  - `WS /ws` - Real-time collaboration
  - `GET /health` - Service health

- **AI Router**: http://localhost:9400
  - `POST /api/ai/inbox/summarize` - Summarize inbox messages
  - `POST /api/ai/document/summarize` - Summarize document
  - `POST /api/ai/search` - AI-powered search
  - `GET /health` - Service health

### Demo Credentials
- **Email**: demo@atlas.local
- **Password**: demo123

## 🚀 Getting Started

### Prerequisites
- Docker and Docker Compose
- Node.js 18+ (for local development)
- Go 1.21+ (for service development)

### Development Setup
1. **Start all services**:
   ```bash
   cd atlas/infra
   docker compose up --build
   ```

2. **Test the application**:
   - Open http://localhost:3000
   - Try demo mode or create an account
   - Create and edit documents collaboratively
   - Test AI summarization features

3. **Monitor services**:
   ```bash
   docker compose logs -f web id docs ai-router
   ```

### Service Development
Each service can be developed independently:

```bash
# ID Service
cd atlas/services/id
go run ./cmd/idsvc

# Docs Service  
cd atlas/services/docs
go run ./cmd/docsvc

# AI Router
cd atlas/services/ai-router
go run ./cmd/ai-router

# Frontend
cd atlas/apps/web
npm run dev
```

## 🛠️ Technology Stack

### Backend Services (Go)
- **Gorilla Mux**: HTTP routing
- **PGX**: PostgreSQL driver
- **Gorilla WebSocket**: Real-time communication
- **JWT-GO**: Authentication tokens
- **NATS**: Message bus integration

### Frontend (Next.js)
- **React 18**: UI framework
- **TypeScript**: Type safety
- **Axios**: HTTP client
- **JS-Cookie**: Cookie management
- **Custom WebSocket**: Real-time features

### Infrastructure
- **PostgreSQL 15**: Primary database
- **NATS 2.10**: Message broker
- **MinIO**: Object storage
- **Docker**: Containerization
- **Docker Compose**: Orchestration

## 📋 Next Steps

### Immediate Enhancements
- [ ] Add real user authentication flow
- [ ] Implement proper Yjs CRDT library
- [ ] Add vector embeddings for AI search
- [ ] Implement document sharing and collaboration
- [ ] Add email bridge integration

### Advanced Features
- [ ] AI model marketplace integration
- [ ] Advanced workflow engine
- [ ] Mobile application
- [ ] Enterprise SSO integration
- [ ] Advanced analytics dashboard

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.