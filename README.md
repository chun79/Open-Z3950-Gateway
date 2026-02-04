# Open-Z3950-Gateway

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/Frontend-React_18-61DAFB?style=flat&logo=react)
![Wails](https://img.shields.io/badge/Desktop-Wails_v2-red?logo=wails)
![gRPC](https://img.shields.io/badge/API-gRPC%20%2F%20ConnectRPC-blue)
![License](https://img.shields.io/badge/License-MIT-green)

**Open-Z3950-Gateway** is a next-generation library gateway platform that bridges the gap between modern Web/Desktop applications and the legacy Z39.50 library protocol.

It features a **Hybrid Architecture** supporting both classic REST APIs and high-performance **gRPC Streaming**, along with a native Desktop application built with Wails.

---

## ✨ Key Features

### 🚀 High-Performance Search
*   **Federated Streaming Search**: Query multiple libraries (e.g., Library of Congress, Oxford, Harvard) concurrently. Results are **streamed** to the UI in real-time via gRPC (Web) or Native Events (Desktop).
*   **Z39.50 Scan**: Browse indexes (Title, Author, Subject) with infinite scrolling context.
*   **Hybrid Provider**: Seamlessly blends results from local databases (SQLite/Postgres) and remote Z39.50 targets.

### 💻 Native Desktop App
*   **Cross-Platform**: Runs natively on **macOS**, **Windows**, and **Linux** using Wails v2.
*   **Local-First**: Built-in SQLite database for offline bookshelf management and search history.
*   **Zero-Config**: Pre-configured with major world libraries; just download and run.

### 🌐 Modern Web Platform
*   **Responsive UI**: Built with React + Pico.css + Vite.
*   **Admin Dashboard**: Full ILL (Inter-Library Loan) request management workflow (Approve/Reject).
*   **Swagger UI**: Interactive API documentation available at `/swagger/index.html`.

---

## 🛠 Architecture

```mermaid
graph TD
    subgraph "Clients"
        Browser[Web Browser]
        Desktop[Desktop App (Wails)]
        ExtClient[External Z39.50 Client]
    end

    subgraph "Gateway Service (:8899)"
        Router[Gin Router]
        Auth[JWT Middleware]
        ConnectRPC[ConnectRPC Handler]
        RestAPI[REST Handlers]
        ZServer[Z39.50 TCP Server (:2100)]
        
        Router --> Auth
        Auth --> RestAPI
        Auth --> ConnectRPC
    end

    subgraph "Core Logic"
        Hybrid[Hybrid Provider]
        Proxy[Z39.50 Client]
        LocalDB[(SQLite / Postgres)]
    end

    Browser -->|HTTP/2 gRPC-Web| ConnectRPC
    Browser -->|HTTP/1.1 REST| RestAPI
    Desktop -->|Native Bindings| Proxy
    ExtClient -->|TCP/Z39.50| ZServer

    RestAPI --> Hybrid
    ConnectRPC --> Hybrid
    ZServer --> Hybrid

    Hybrid -->|Read/Write| LocalDB
    Hybrid -->|Search/Scan| Proxy

    Proxy -->|Z39.50 (ASN.1/BER)| Targets[Remote Libraries\n(LOC, Oxford, Yale...)]
```

---

## 🚀 Quick Start

### Option A: Docker (Web Service)
Ideal for server deployment. Includes Gateway + Webapp + Database.

```bash
# 1. Start services (defaults to SQLite mode)
docker compose up -d

# 2. Access Web UI
open http://localhost:8899

# 3. Access Swagger API Docs
open http://localhost:8899/swagger/index.html
```

### Option B: Desktop App (Local Development)
Ideal for personal use or development on macOS/Windows.

**Prerequisites**: Go 1.21+, Node.js 18+, Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

```bash
# 1. Enter desktop directory
cd desktop

# 2. Run in Dev Mode (Hot Reload)
wails dev

# 3. Build Production Binary
wails build
```

### Option C: Manual Backend Build
For hacking on the gateway logic without Docker.

```bash
# 1. Build Frontend
make frontend-install frontend-build

# 2. Embed & Build Backend
make build

# 3. Run Gateway
./gateway
```

---

## 📚 API Documentation

The Gateway provides a unified API surface for all clients.

### gRPC / ConnectRPC
*   **Endpoint**: `https://your-gateway/gateway.v1.GatewayService`
*   **Protocol**: HTTP/2 or HTTP/1.1 (via Connect protocol)
*   **Service Definition**: [proto/gateway/v1/gateway.proto](proto/gateway/v1/gateway.proto)

### REST API
*   `GET /api/search`: Standard search (JSON).
*   `GET /api/federated-search`: Concurrent search across multiple targets.
*   `GET /api/scan`: Browse index terms (e.g. for autocomplete).
*   `POST /api/ill-requests`: Submit a loan request.

View full interactive docs at **`/swagger/index.html`** after starting the server.

---

## ⚙️ Configuration

Configure via `.env` file or environment variables.

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | Web/API Port | `8899` |
| `ZSERVER_PORT` | Z39.50 Server Port | `2100` |
| `DB_PROVIDER` | `sqlite` or `postgres` | `sqlite` |
| `DB_PATH` | Path to SQLite DB file | `library.db` |
| `DB_DSN` | Postgres connection string | (Empty) |
| `GATEWAY_API_KEY` | Admin API Key (Bypass JWT) | - |

---

## 🤝 Contributing

1.  **Fork** the repository.
2.  **Clone** your fork.
3.  **Install tools**: `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` and `connectrpc.com/connect/cmd/protoc-gen-connect-go@latest`.
4.  **Create a branch** (`git checkout -b feat/new-feature`).
5.  **Commit** (`git commit -m 'feat: add amazing feature'`).
6.  **Push** (`git push origin feat/new-feature`).
7.  Open a **Pull Request**.

## License

MIT © 2026 Open-Z3950-Gateway Contributors