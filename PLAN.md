# TUF Client Verify - Project Plan

## Project Overview

**Goal**: Build an authentication endpoint service that acts as**Performance**: Sub-100ms response time for auth requests
**Reliability**: 99.9% uptime in production scenarios TUF client to verify access requests against a predefined TUF repository with embedded root of trust. This service will be used as an auth callback by reverse proxies like nginx to authorize access to specific endpoints.

**Core Functionality**:

- HTTP service that receives auth requests from nginx (or other proxies)
- Extracts requested path from nginx auth_request 
- Verifies if the path exists as a target in TUF metadata
- Returns HTTP 200 (authorized) or 403 (forbidden) responses
- Embeds root of trust for TUF repository verification

**Auth Flow (PoC)**:
1. User requests `/v2/library/alpine/manifests/latest` from nginx
2. nginx sends auth_request to `/auth` with original path in headers
3. Service extracts path and checks TUF metadata with delegation for `/v2/library/*`
4. Returns 200 if path matches delegated targets, 403 otherwise

**TUF Repository Structure (PoC)**:
- Root metadata with delegation to `/v2/library/*` paths
- Delegated targets metadata containing specific library resources
- Example targets: 
  - `/v2/library/alpine/manifests/latest`
  - `/v2/library/ubuntu/manifests/20.04` 
  - `/v2/library/nginx/manifests/latest`
- Generated using go-tuf repository implementation

## Technical Stack
- **Language**: Go
- **Framework**: Standard library HTTP server (initially)
- **TUF Library**: github.com/theupdateframework/go-tuf/v2
- **Deployment**: Docker container
- **Testing**: nginx integration example

## Project Phases

### Phase 1: Basic HTTP Auth Endpoint (MVP)
**Timeline**: Week 1-2

**Goals**:
- Create basic HTTP server with `/auth` endpoint
- Return static HTTP 200 responses (always allow)
- Implement nginx auth_request configuration with static responses
- nginx serves static JSON responses for `/v2/library/*` paths
- Docker containerization

**Deliverables**:
- `cmd/tuf-client-verify/main.go` - Basic HTTP server
- `docker-compose.yml` - nginx + auth service setup
- `examples/nginx/` - nginx configuration with auth_request + static responses
- `examples/nginx/static/` - Mock registry API responses (JSON files)
- Basic README with setup instructions

### Phase 2: TUF Repository Setup & Integration
**Timeline**: Week 3-4

**Goals**:
- Create dummy TUF repository with signed metadata and `/v2/library/*` delegation
- Set up local file serving for TUF repository via nginx
- Integrate go-tuf client library
- Implement path verification against delegated TUF metadata
- Add configuration for TUF repository URL and embedded root of trust

**Deliverables**:
- `internal/tuf/` - TUF client wrapper with delegation support
- `testdata/repository/` - Dummy TUF repository with delegation
- `scripts/generate-tuf-repo.go` - Go program using go-tuf repository implementation
- `config/` - Configuration management
- Updated nginx config to serve TUF repository files
- Updated auth endpoint with actual TUF verification

### Phase 3: Production Features & Testing
**Timeline**: Week 5-6

**Goals**:
- Add comprehensive error handling and logging
- Implement metrics and health endpoints
- Comprehensive test suite
- Performance testing

**Deliverables**:
- `internal/metrics/` - Prometheus metrics
- `tests/` - Integration and unit tests
- Performance benchmarks
- Production deployment guide

## Project Structure
```
tuf-client-verify/
├── cmd/
│   └── tuf-client-verify/
│       └── main.go
├── internal/
│   ├── auth/           # Auth endpoint handlers
│   ├── tuf/            # TUF client wrapper
│   ├── config/         # Configuration management
│   └── metrics/        # Prometheus metrics
├── examples/
│   ├── nginx/          # nginx configuration examples
│   └── docker/         # Docker deployment examples
├── testdata/
│   └── repository/     # Test TUF repository
├── tests/
│   ├── integration/    # Integration tests
│   └── unit/          # Unit tests
├── scripts/
│   └── setup-test-repo.sh
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

## Success Criteria

1. **Functional**: nginx can successfully use the service for auth_request callbacks
2. **Security**: All TUF metadata verification follows best practices
3. **Performance**: Sub-100ms response time for auth requests (with caching)
4. **Reliability**: 99.9% uptime in production scenarios
5. **Documentation**: Complete setup and deployment guides

## Risk Mitigation

- **TUF Complexity**: Start with go-tuf examples, implement incrementally
- **nginx Integration**: Begin with simple proxy setup, test early
- **Performance**: Implement caching from Phase 2, monitor metrics
- **Security**: Embed root of trust, verify all signatures properly

## Next Steps

1. Set up basic project structure with Go modules
2. Implement Phase 1 MVP with basic HTTP server
3. Create docker-compose setup with nginx example
4. Iterate based on testing and feedback

## Demo Scenarios

**Successful Auth**:
- `curl -v http://localhost/v2/library/alpine/manifests/latest` → 200 + static JSON response
- `curl -v http://localhost/v2/library/ubuntu/manifests/20.04` → 200 + static JSON response
- `curl -v http://localhost/v2/library/nginx/manifests/latest` → 200 + static JSON response

**Failed Auth**:
- `curl -v http://localhost/v2/library/redis/manifests/latest` → 403 (not in delegation)
- `curl -v http://localhost/v1/some/other/path` → 403 (wrong API version)
- `curl -v http://localhost/api/admin/panel` → 403 (completely different path)
