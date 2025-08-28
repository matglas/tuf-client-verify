# TUF Client Verify

A service that acts as a TUF client to verify access requests against a predefined TUF repository with embedded root of trust. This service is designed to be used as an auth callback by reverse proxies like nginx to authorize access to specific endpoints.

## Current Status: Phase 2 Complete ✅

This is a **Proof of Concept** that has successfully completed Phase 2 implementation with full TUF delegation-based verification.

### ✅ Phase 2 Completed Features

- ✅ Complete TUF repository with signed metadata and delegation
- ✅ TUF client integration with path verification against delegated metadata
- ✅ Real TUF-based authorization (replaces static responses)
- ✅ `/v2/library/*` paths allowed via TUF delegation to `registry-library` role
- ✅ All other paths denied with cryptographic verification
- ✅ nginx auth_request integration with proper 403 responses
- ✅ Docker containerization with TUF repository data
- ✅ Complete end-to-end verification

### Quick Verification

Run the automated system verification:
```bash
./scripts/verify-system.sh
```

## Development Setup

⚠️ **Important**: The TUF repository contains cryptographic keys and is not committed to version control.

**See [DEVELOPMENT.md](DEVELOPMENT.md) for complete setup instructions.**

### Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)
- jq (for JSON processing in verification scripts)

### Running with Docker Compose

1. Clone the repository:
```bash
git clone <repository-url>
cd tuf-client-verify
```

2. Generate the TUF repository (required):
```bash
go run scripts/generate-tuf-repo.go
```

3. Start the services:
```bash
docker-compose up --build -d
```

4. Verify the system:
```bash
./scripts/verify-system.sh
```

### Manual Testing

```bash
# Test allowed path (should return 200 + JSON manifest)
curl -v http://localhost/v2/library/alpine/manifests/latest

# Test denied path (should return 403 Forbidden)
curl -v http://localhost/v2/redis/manifests/latest

# Check TUF delegation configuration
curl -s http://localhost:8080/debug | jq .

# Test successful auth (should return 200 + JSON manifest)
curl -v http://localhost/v2/library/nginx/manifests/latest

# Test registry API version
curl -v http://localhost/v2/

# Test direct auth service health
curl -v http://localhost:8080/health
```

### Running Locally for Development

1. Install dependencies:
```bash
go mod download
```

2. Run the auth service:
```bash
go run cmd/tuf-client-verify/main.go
```

3. Test the auth endpoint directly:
```bash
curl -v http://localhost:8080/auth
curl -v http://localhost:8080/health
```

## Architecture (Phase 1)

```
┌─────────────┐    ┌─────────────┐    ┌──────────────────┐
│   Client    │───▶│    nginx    │───▶│ tuf-client-verify│
│             │    │             │    │   (auth service) │
└─────────────┘    └─────────────┘    └──────────────────┘
                         │
                         ▼
                   ┌─────────────┐
                   │Static JSON  │
                   │ Responses   │
                   └─────────────┘
```

### Flow

1. Client requests `/v2/library/alpine/manifests/latest`
2. nginx receives request and calls `/auth` on tuf-client-verify service
3. Auth service returns 200 (always allow in Phase 1)
4. nginx serves static JSON manifest from local files

## Service Endpoints

### Auth Service (tuf-client-verify:8080)
- `GET /auth` - Auth endpoint for nginx auth_request (always returns 200 in Phase 1)
- `GET /health` - Health check endpoint
- `GET /` - Service info

### nginx Proxy (localhost:80)
- `GET /v2/library/{image}/manifests/{tag}` - Container manifest API with auth
- `GET /v2/library/{image}/blobs/{digest}` - Container blob API with auth
- `GET /v2/` - Registry version check
- `GET /nginx-health` - nginx health check

## Configuration

### Environment Variables

- `PORT` - Port for the auth service (default: 8080)

### nginx Configuration

The nginx configuration in `examples/nginx/nginx.conf` implements:
- `auth_request` to `/auth` endpoint
- Static file serving for manifest responses
- Proper headers for container registry API compatibility

### Static Responses

Mock container registry responses are stored in `examples/nginx/static/manifests/`:
- `alpine-latest.json` - Alpine Linux manifest
- `ubuntu-20.04.json` - Ubuntu 20.04 manifest  
- `nginx-latest.json` - nginx manifest

## Development

### Project Structure

```
tuf-client-verify/
├── cmd/
│   └── tuf-client-verify/
│       └── main.go              # Main HTTP server
├── examples/
│   └── nginx/
│       ├── nginx.conf           # nginx configuration
│       └── static/
│           └── manifests/       # Mock JSON responses
├── docker-compose.yml           # Docker setup
├── Dockerfile                   # Container build
├── go.mod                       # Go dependencies
├── PLAN.md                      # Project roadmap
├── PROGRESS.md                  # Implementation tracking
└── README.md                    # This file
```

### Testing

```bash
# Build and test locally
go build -o tuf-client-verify cmd/tuf-client-verify/main.go
./tuf-client-verify

# Test with docker-compose
docker-compose up --build

# Run tests (when available)
go test ./...
```

## Next Steps (Phase 2)

- [ ] Integrate actual TUF repository with signed metadata
- [ ] Implement path verification against TUF delegations
- [ ] Add configuration management for TUF repository URLs
- [ ] Embed root of trust for TUF verification

## Roadmap

See [PLAN.md](PLAN.md) for the complete project roadmap and [PROGRESS.md](PROGRESS.md) for current implementation status.

## License

[License TBD]
