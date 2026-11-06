# TechGuild Chat System (Week 6)

Comprehensive, high-performance real-time chat and communication service for the TechGuild platform, implemented strictly following the **Week 6 Chat System Specification**.

---

## 🏛 Architecture Overview

Chat transport uses **Centrifugo v5**, a standalone real-time messaging server that runs alongside the Go backend as a separate deployable service in Kubernetes:
- **WebSocket Gateway:** Handles persistent client WebSocket connections (`wss://chat.techguild.io/connection/websocket`), presence, and reconnections out of the box.
- **Horizontal Scaling:** Uses Redis Pub/Sub engine for multi-pod synchronization. Messages published on any pod reach clients across all pods.
- **Durable Persistence:** PostgreSQL is the durable source of truth for messages, reactions, channel memberships, and pins.
- **Authentication:** Mints short-lived Centrifugo JWTs signed with HMAC-SHA256 (`token_hmac_secret_key`), which Centrifugo verifies locally.
- **Authorization & Access Control:** Centrifugo proxies channel subscription requests to the backend (`/v1/communication/centrifugo/subscribe-proxy`), allowing/denying channel joins based on membership.

---

## 📁 Directory Structure

All ChatSystem code, components, configurations, and deliverables are maintained exclusively within the `chatSystem` folder:

```
chatSystem/
├── cmd/
│   └── server/
│       └── main.go                 # Standalone ChatSystem microservice runner
├── config/
│   └── config.go                   # Environment configuration (Centrifugo, DB, Redis, R2, Limits)
├── controllers/
│   ├── channel_controller.go       # Endpoints 1-5 (Channel management)
│   ├── message_controller.go       # Endpoints 6-9 (Messages, edit, tombstone delete)
│   ├── file_controller.go          # Endpoint 10 (File sharing & validation)
│   ├── reaction_controller.go      # Endpoints 11-12 (Emoji reactions)
│   ├── pin_controller.go           # Endpoints 13-15 (Message pinning)
│   ├── search_controller.go        # Endpoint 16 (Scoped search)
│   └── centrifugo_controller.go    # Endpoints 17-18 (JWT issuance & subscribe proxy)
├── dto/
│   ├── channel_dto.go              # DTOs for channel management
│   ├── message_dto.go              # DTOs for messages, pagination & reactions
│   ├── file_dto.go                 # DTOs for file uploads & presigned URLs
│   ├── reaction_dto.go             # DTOs for adding/removing reactions
│   ├── pin_dto.go                  # DTOs for pinning & listing pinned messages
│   ├── search_dto.go               # DTOs for scoped message search
│   └── centrifugo_dto.go           # DTOs for token minting & subscribe proxy
├── jobs/
│   └── cleanup_job.go              # Daily cleanup & orphaned data maintenance
├── middleware/
│   ├── auth.go                     # JWT verification & Redis blacklist check
│   └── channel_auth.go             # Channel membership validation helpers
├── models/
│   ├── communication.go            # GORM models (communication_records, reactions, members, channels)
│   └── migrate.go                  # Auto-migration & database index setup
├── routes/
│   └── routes.go                   # Huma API route registration for all 18 endpoints
├── services/
│   ├── centrifugo_service.go       # Centrifugo HTTP client & token issuance
│   ├── channel_service.go          # Channel CRUD, DM creation & auto-creation hooks
│   ├── message_service.go          # Message persistence, 5-min edit window & tombstones
│   ├── file_service.go             # File validation (.pdf,.xml,.doc,.docx,.md), R2 & virus scan hook
│   ├── reaction_service.go         # Reaction management with unique constraint
│   ├── pin_service.go              # Fast indexed pinned messages
│   └── search_service.go           # Scoped message search
├── utils/
│   ├── centrifugo_jwt.go           # HMAC-SHA256 JWT builder for Centrifugo
│   ├── context.go                  # Context key extractors
│   ├── cursor.go                   # Opaque cursor encoding/decoding for pagination
│   └── validator.go                # Allowed extensions, MIME types, size & virus scan interface
├── k8s/
│   ├── centrifugo-configmap.yaml   # Centrifugo configuration
│   ├── centrifugo-deployment.yaml  # Multi-pod deployment with Redis engine
│   ├── centrifugo-service.yaml     # Internal ClusterIP service on port 8000
│   └── centrifugo-ingress.yaml     # Ingress with WebSocket upgrade headers
├── terraform/
│   ├── main.tf                     # Provider declarations & input variables
│   └── centrifugo.tf               # Complete K8s deployment, secrets, service & ingress entry
├── go.mod
└── go.sum
```

---

## 📋 Implemented API Endpoints (18 Total)

### 1. Channel Management
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/v1/communication/channels` | List user's channels (quest, party, dm), grouped by type |
| `POST` | `/v1/communication/channels` | Create 1:1 DM channel (or return existing) |
| `GET` | `/v1/communication/channels/:id` | Channel info (budget, duration, milestone count, member count) |
| `GET` | `/v1/communication/channels/:id/members` | List channel members with roles |
| `PATCH` | `/v1/communication/channels/:id/mute` | Mute/unmute channel notifications |

### 2. Messages
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/v1/communication/channels/:id/messages` | Get message history (cursor-based pagination, reactions, tombstones) |
| `POST` | `/v1/communication/channels/:id/messages` | Send message (REST fallback, supports `reply_to_id`, broadcasts to Centrifugo) |
| `PATCH` | `/v1/communication/channels/:id/messages/:msg_id` | Edit message (server-enforced 5-minute window, stores original in metadata) |
| `DELETE` | `/v1/communication/channels/:id/messages/:msg_id` | Delete message (soft delete with tombstone for audit continuity) |

### 3. Files
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/v1/communication/channels/:id/files` | Upload file (`.pdf`, `.xml`, `.doc`, `.docx`, `.md`) with type & size validation and virus scanning hook |

### 4. Reactions
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/v1/communication/channels/:id/messages/:msg_id/reactions` | Add emoji reaction (unique per message/user/emoji) |
| `DELETE` | `/v1/communication/channels/:id/messages/:msg_id/reactions/:reaction_id` | Remove reaction (scoped to reaction owner) |

### 5. Pinning
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/v1/communication/channels/:id/messages/:msg_id/pin` | Pin message |
| `DELETE` | `/v1/communication/channels/:id/messages/:msg_id/pin` | Unpin message |
| `GET` | `/v1/communication/channels/:id/pinned` | List pinned messages (fast indexed retrieval) |

### 6. Search
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/v1/communication/search?q=` | Search messages (channel-scoped or global, strictly scoped to user's member channels) |

### 7. Centrifugo Bridge
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/v1/communication/token` | Issue Centrifugo connection JWT signed with HMAC-SHA256 |
| `POST` | `/v1/communication/centrifugo/subscribe-proxy` | Centrifugo internal proxy hook checking channel join permissions |

---

## 🗄️ Database Tables (`PostgreSQL`)

1. **`communication_records`**:
   - `id` (UUID PK), `quest_id` (UUID), `party_id` (UUID), `from_user_id` (UUID), `to_user_id` (UUID), `channel_id` (varchar), `channel_type` (varchar), `reply_to_id` (UUID), `content_ref` (text), `content_text` (text), `message_type` (varchar), `metadata` (JSONB), `is_pinned` (boolean), `deleted` (boolean), `created_at` (timestamptz), `edited_at` (timestamptz).
2. **`message_reactions`**:
   - `id` (UUID PK), `message_id` (UUID), `user_id` (UUID), `emoji` (varchar), `created_at` (timestamptz).
   - Composite Unique Index: `(message_id, user_id, emoji)`.
3. **`channel_members`**:
   - `id` (UUID PK), `channel_id` (varchar), `user_id` (UUID), `role` (varchar), `muted` (boolean), `joined_at` (timestamptz).
   - Composite Unique Index: `(channel_id, user_id)`.
4. **`chat_channels`**:
   - `id` (varchar PK), `type` (varchar), `name` (varchar), `quest_id` (UUID), `party_id` (UUID), `created_by` (UUID), `created_at` (timestamptz), `updated_at` (timestamptz).

---

## ⚙️ How to Run

### Run Unit Tests
```bash
cd chatSystem
go test -v ./utils/...
```

### Build & Run the Chat Service
```bash
cd chatSystem
go run ./cmd/server
```
The server will start on the configured `PORT` (default `:8080`), connect to PostgreSQL and Redis, run schema migrations, start the background maintenance jobs, and expose all 18 endpoints under `/v1/communication/...`.

