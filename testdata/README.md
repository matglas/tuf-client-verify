# testdata Directory

This directory contains the TUF repository with cryptographic keys and signed metadata. It is automatically excluded from version control by `.gitignore` for security reasons.

## Generation

To generate the TUF repository:

```bash
go run scripts/generate-tuf-repo.go
```

This will create:
- `repository/root.json` - Root metadata with key information
- `repository/targets.json` - Top-level targets with delegation
- `repository/snapshot.json` - Snapshot of current metadata versions  
- `repository/timestamp.json` - Timestamped hash of snapshot
- `repository/registry-library.json` - Delegated targets for /v2/library/*

## Security Note

⚠️ **Important**: This directory contains private cryptographic keys used for signing TUF metadata. These keys should never be committed to version control or shared.

In production environments:
- Use proper key management systems (HSMs)
- Implement key rotation policies
- Store keys securely with appropriate access controls
