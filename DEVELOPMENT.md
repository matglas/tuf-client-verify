# Development Guide - TUF Client Verify

This document provides step-by-step instructions for setting up and running the TUF Client Verify system in development mode.

## Prerequisites

- Docker and Docker Compose
- Go 1.21+ 
- jq (for JSON processing in verification scripts)

## Development Setup

### 1. Clone and Setup

```bash
git clone <repository-url>
cd tuf-client-verify
```

### 2. Generate TUF Repository (Required)

The TUF repository contains cryptographic keys and is not committed to version control. You must generate it locally:

```bash
# Generate the TUF repository with signed metadata
go run scripts/generate-tuf-repo.go
```

This creates:
- `testdata/repository/` directory
- Signed TUF metadata files (root.json, targets.json, etc.)
- Delegation for `/v2/library/*` paths to `registry-library` role
- ed25519 cryptographic keys and signatures

**Important**: The `testdata/repository/` directory contains private keys and should never be committed to version control.

### 3. Build and Run Services

```bash
# Build and start all services
docker-compose up --build -d

# Check services are running
docker-compose ps
```

### 4. Verify Setup

Run the comprehensive verification script:

```bash
./scripts/verify-system.sh
```

## Development Workflow

### Local Development

For faster iteration during development:

```bash
# Run the auth service locally (outside Docker)
go run cmd/tuf-client-verify/main.go

# In another terminal, test directly
curl -H "X-Original-URI: /v2/library/alpine/manifests/latest" http://localhost:8080/auth
```

### Rebuilding After Changes

```bash
# Rebuild and restart services
docker-compose down
docker-compose up --build -d

# Verify changes
./scripts/verify-system.sh
```

### Regenerating TUF Repository

If you need to regenerate the TUF repository (e.g., after changing delegation rules):

```bash
# Remove existing repository
rm -rf testdata/repository/

# Generate new repository
go run scripts/generate-tuf-repo.go

# Rebuild services to include new repository
docker-compose down
docker-compose up --build -d
```

## System Architecture

### TUF Repository Structure

```
testdata/repository/
├── root.json              # Root metadata with key information
├── targets.json           # Top-level targets with delegation
├── snapshot.json          # Snapshot of current metadata versions
├── timestamp.json         # Timestamped hash of snapshot
└── registry-library.json  # Delegated targets for /v2/library/*
```

### Service Flow

1. **nginx** receives request (e.g., `/v2/library/alpine/manifests/latest`)
2. **nginx** makes auth_request to TUF auth service
3. **TUF auth service** verifies path against TUF delegation metadata
4. **TUF auth service** returns 200 (allow) or 403 (deny)
5. **nginx** serves content or denies access based on auth response

### Delegation Configuration

The system implements TUF delegation where:
- **Root metadata** defines the `registry-library` role
- **Targets metadata** delegates `/v2/library/*` paths to `registry-library`
- **registry-library.json** contains the allowed specific paths
- Only paths matching the delegation pattern are authorized

## Testing

### Unit Testing

```bash
# Test TUF client functionality
go run scripts/test-tuf-client.go
```

### Integration Testing

```bash
# Full system verification
./scripts/verify-system.sh
```

### Manual Testing

```bash
# Test allowed path
curl -v http://localhost/v2/library/alpine/manifests/latest
# Expected: 200 OK with manifest content

# Test denied path  
curl -v http://localhost/v2/redis/manifests/latest
# Expected: 403 Forbidden

# Check debug info
curl -s http://localhost:8080/debug | jq .
# Shows delegation configuration and allowed paths
```

## Configuration

### Environment Variables

- `PORT`: Auth service port (default: 8080)
- `TUF_REPO_PATH`: Path to TUF repository (default: testdata/repository)

### TUF Repository Configuration

Edit `scripts/generate-tuf-repo.go` to modify:
- Delegation patterns (currently `/v2/library/*`)
- Allowed paths in delegated metadata
- Key types and expiration dates

## Troubleshooting

### Service Won't Start

```bash
# Check service logs
docker-compose logs tuf-client-verify

# Common issues:
# - Missing testdata/repository/ (run generate-tuf-repo.go)
# - Port conflicts (check if port 8080/80 are in use)
```

### TUF Verification Errors

```bash
# Test TUF client directly
go run scripts/test-tuf-client.go

# Check TUF metadata
ls -la testdata/repository/
cat testdata/repository/root.json | jq .
```

### nginx Integration Issues

```bash
# Check nginx logs
docker-compose logs nginx

# Test auth service directly
curl -H "X-Original-URI: /test/path" http://localhost:8080/auth
```

## Security Considerations

⚠️ **Important Security Notes:**

1. **Private Keys**: The `testdata/repository/` contains private keys used for signing TUF metadata. This directory should never be committed or shared.

2. **Production Setup**: In production:
   - Use proper key management (HSMs, key rotation)
   - Implement secure key storage
   - Use network policies for service isolation
   - Enable proper logging and monitoring

3. **TUF Best Practices**: This implementation follows TUF specification for delegation but is simplified for PoC purposes.

## Next Steps (Phase 3)

- [ ] Production-ready error handling and logging
- [ ] Metrics and monitoring integration
- [ ] Comprehensive test suite
- [ ] Performance optimization
- [ ] Security hardening
- [ ] CI/CD pipeline integration
