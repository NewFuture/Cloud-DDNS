# Cloud-DDNS Project Overview

## Purpose
Cloud-DDNS is a lightweight middleware service that bridges legacy network devices (routers, DVRs, NAS) supporting the GnuDIP protocol with modern cloud DNS providers (Alibaba Cloud, Tencent Cloud).

## Architecture

### Core Components

1. **Configuration Module** (`pkg/config/`)
   - Loads YAML configuration files
   - Manages user credentials (AccessKey/SecretKey)
   - Maps users to cloud providers
   - Files: `config.go`, `config_test.go`

2. **Provider Module** (`pkg/provider/`)
   - Defines Provider interface for cloud DNS operations
   - Implements provider-specific adapters:
     - `aliyun.go` - Alibaba Cloud DNS implementation
     - `tencent.go` - Tencent Cloud DNSPod implementation
   - Factory pattern in `provider.go` for creating provider instances
   - Files: `provider.go`, `aliyun.go`, `tencent.go`, `provider_test.go`

3. **Server Module** (`pkg/server/`)
   - Implements GnuDIP protocol TCP server (port 3495)
   - Implements HTTP simple mode (port 8080)
   - Handles authentication using MD5 challenge-response
   - Extracts IP addresses from client connections
   - Files: `server.go`, `server_test.go`

4. **Main Entry** (`main.go`)
   - Initializes configuration
   - Starts TCP and HTTP servers concurrently
   - Manages server lifecycle

## Protocol Implementation

### GnuDIP TCP Protocol
1. Server sends salt (timestamp-based challenge)
2. Client responds with: `User:Hash:Domain:ReqC:IP`
3. Server validates hash: MD5(User:Salt:Password)
4. Server updates DNS record via cloud provider API

### HTTP Simple Mode
- Query parameters: `user`, `pass`, `domn`/`domain`, `addr`
- Auto-detects client IP if `addr` is empty
- Returns status code: 0 (success) or error

## Authentication Flow
- **Pass-through authentication**: Username = AccessKey ID, Password = SecretKey
- No intermediate credential storage
- Direct mapping to cloud provider credentials

## DNS Update Logic
1. Parse full domain into base domain and subdomain
2. Query existing DNS records
3. Compare current IP with new IP
4. Skip API call if IP unchanged (optimization)
5. Update or create A record

## Key Design Patterns
- **Factory Pattern**: Provider creation in `provider.GetProvider()`
- **Interface Segregation**: Single `UpdateRecord()` method in Provider interface
- **Dependency Injection**: Configuration and providers passed as needed
- **Error Handling**: Explicit error checks throughout, nil pointer safety

## Configuration Format
```yaml
server:
  tcp_port: 3495    # GnuDIP standard port
  http_port: 8080   # HTTP alternative

users:
  - username: "AccessKeyID"
    password: "SecretKey"
    provider: "aliyun|tencent"
```

## Testing Strategy
- Unit tests for each module (config, provider, server)
- Protocol logic tests (MD5, salt generation, message parsing)
- 100% coverage for configuration module
- Mock-free testing where possible (functional tests)

## Build and Deployment
- **Local**: `go build` or `make build`
- **Docker**: Multi-stage build with Go 1.25 + Alpine
- **CI/CD**: GitHub Actions for testing, building, and releases
- **Platforms**: Linux (amd64, arm64, armv7), macOS, Windows

## Dependencies
- `gopkg.in/yaml.v3` - YAML configuration parsing
- `github.com/alibabacloud-go/alidns-20150109/v4` - Alibaba Cloud DNS SDK
- `github.com/tencentcloud/tencentcloud-sdk-go` - Tencent Cloud SDK

## Extension Points
To add new cloud provider:
1. Create `pkg/provider/newprovider.go`
2. Implement `Provider` interface with `UpdateRecord(domain, ip) error`
3. Add case in `provider.GetProvider()` switch statement
4. Update config.yaml.example with new provider type
