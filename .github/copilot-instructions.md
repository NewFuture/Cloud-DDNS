# Cloud-DDNS - GitHub Copilot Instructions

This document provides instructions for GitHub Copilot to assist with development of the Cloud-DDNS project.

## Project Overview

Cloud-DDNS is a lightweight middleware service that bridges legacy network devices (routers, DVRs, NAS) supporting the GnuDIP protocol with modern cloud DNS providers (Alibaba Cloud, Tencent Cloud).

### Core Purpose
- Act as a GnuDIP protocol server (TCP port 3495)
- Provide HTTP simple mode alternative (port 8080)
- Bridge old devices with modern cloud DNS APIs
- Support pass-through authentication using cloud provider credentials

## Architecture

### Module Structure

```
pkg/
├── config/     # YAML configuration loading and user management
├── provider/   # Cloud DNS provider adapters with unified interface
└── server/     # GnuDIP TCP and HTTP protocol servers
```

### Key Components

1. **Configuration Module** (`pkg/config/`)
   - Loads YAML configuration files
   - Manages user credentials (AccessKey/SecretKey mapping)
   - Files: `config.go`, `config_test.go`

2. **Provider Module** (`pkg/provider/`)
   - Defines `Provider` interface: `UpdateRecord(domain, ip) error`
   - Implements cloud-specific adapters:
     - `aliyun.go` - Alibaba Cloud DNS via alidns-20150109 SDK
     - `tencent.go` - Tencent Cloud DNSPod via tencentcloud-sdk-go
   - Factory pattern in `provider.go`
   - Files: `provider.go`, `aliyun.go`, `tencent.go`, `provider_test.go`

3. **Server Module** (`pkg/server/`)
   - GnuDIP protocol TCP server with MD5 challenge-response
   - HTTP simple mode with query parameters **and Basic Auth fallback**
   - Mode-based request handling in `pkg/server/mode/`: a `Mode` interface standardizes parameters, resolves missing IPs from RemoteAddr, validates domain/IP, performs authentication, and delegates to providers via protocol-specific implementations (e.g., `base.go`, `dyndns.go`, etc.)
   - Files: `server.go`, `mode/base.go`, `mode/dyndns.go`, `server_test.go`

4. **Main Entry** (`main.go`)
   - Initializes configuration from `config.yaml`
   - Starts TCP and HTTP servers concurrently using goroutines
   - Manages server lifecycle

## Protocol Implementation

### GnuDIP TCP Protocol Flow
1. Server sends salt: `{unix_timestamp}.{unix_nanoseconds}`
2. Client responds: `User:Hash:Domain:ReqC:IP`
3. Hash validation: `MD5(User:Salt:Password)`
4. Server updates DNS via provider API
5. Response: `0` (success) or `1` (failure)

### HTTP Simple Mode
- Endpoints: `/`, `/update`, `/nic/update`, `/cgi-bin/gdipupdt.cgi`
- Authentication: HTTP Basic Auth **or** URL parameters (`user`/`pass` aliases)
- IP resolution: `addr`/`myip` optional; when empty uses client `RemoteAddr`; `reqc=1` forces `0.0.0.0`, `reqc=2` always uses `RemoteAddr`
- Returns: `good <ip>` / `badauth` / `notfqdn` / `911` or numeric `0/1/2` for CGI path

### Authentication Pattern
- **Pass-through authentication**: No credential translation
- Username = Cloud Provider AccessKey ID
- Password = Cloud Provider SecretKey/Token
- Direct API credential usage

## Development Guidelines

### Code Style
- Follow standard Go conventions (`go fmt`, `go vet`)
- Add comments for all exported functions and types
- Use meaningful variable names (avoid single-letter except in loops)
- Keep functions focused and single-purpose (max ~50 lines)

### Error Handling
- **Always** check errors explicitly - no silent failures
- Log errors with context using `log.Printf()`
- Return errors up the call stack
- Add nil checks before dereferencing pointers
- Validate input parameters at function entry

### Testing Strategy
- Write unit tests for all new features
- Aim for 100% coverage in core modules (config, provider interface)
- Use table-driven tests for multiple test cases
- Test error paths, not just happy paths
- Mock-free testing preferred (use real logic where possible)

### Design Patterns Used
- **Factory Pattern**: `provider.GetProvider()` creates provider instances
- **Interface Segregation**: Single method `UpdateRecord()` in Provider interface
- **Dependency Injection**: Pass config and providers as parameters
- **Error Propagation**: Explicit error returns throughout

## Common Development Tasks

### Adding a New Cloud Provider

1. Create new file: `pkg/provider/newprovider.go`
2. Implement Provider interface:
   ```go
   type NewProvider struct {
       apiKey    string
       apiSecret string
   }
   
   func NewNewProvider(key, secret string) *NewProvider {
       return &NewProvider{apiKey: key, apiSecret: secret}
   }
   
   func (p *NewProvider) UpdateRecord(domain string, ip string) error {
       // 1. Parse domain into base domain and subdomain
       // 2. Query existing DNS records
       // 3. Compare IPs (skip update if unchanged)
       // 4. Update or create A record via provider API
       return nil
   }
   ```
3. Register in `pkg/provider/provider.go`:
   ```go
   case "newprovider":
       return NewNewProvider(u.Username, u.Password), nil
   ```
4. Add tests in `pkg/provider/provider_test.go`
5. Update `config.yaml.example` with new provider example

### DNS Update Logic Pattern
1. Parse full domain (e.g., `sub.example.com`) into:
   - Base domain: `example.com`
   - Subdomain/RR: `sub` (or `@` for apex)
2. Query existing DNS records for the domain
3. Compare current IP with new IP
4. Skip API call if IP unchanged (optimization)
5. Call API to update or create A record

### Configuration Format
```yaml
server:
  tcp_port: 3495    # GnuDIP standard port
  http_port: 8080   # HTTP alternative

users:
  - username: "AccessKeyID"     # Cloud provider access key
    password: "SecretKey"       # Cloud provider secret
    provider: "aliyun|tencent"  # Provider type
```

## Build and Test Commands

### Local Development
```bash
make build         # Build binary
make test          # Run all tests
make test-verbose  # Run tests with verbose output
make test-coverage # Generate coverage report (coverage.html)
make fmt           # Format code
make fmt-check     # Check code formatting without modifying files
make vet           # Run go vet static analysis
make run           # Build and run
make clean         # Remove build artifacts
```

### Docker
```bash
make docker       # Build Docker image

# Run Docker container manually
docker run -d -p 3495:3495 -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml:ro --name cloud-ddns cloud-ddns:latest
```

### Manual Commands
```bash
go build -o cloud-ddns main.go        # Build
go test ./...                         # Test all packages
go test -v ./pkg/config              # Test specific package
go test -race -coverprofile=cov.txt  # Test with race detection
go fmt ./...                          # Format code
go vet ./...                          # Static analysis
```

### Pre-Commit Checklist

**IMPORTANT**: Before committing any changes, always ensure:

1. **Unit tests pass**: `make test`
2. **Code is formatted**: `make fmt-check` (run `make fmt` to auto-format)
3. **Static analysis passes**: `make vet`
4. **Build succeeds**: `make build`

Quick pre-commit check:
```bash
make test && make fmt-check && make vet && make build
```

## Testing Examples

### HTTP Protocol Test
```bash
curl "http://localhost:8080/?user=YOUR_KEY&pass=YOUR_SECRET&domn=test.example.com&addr=1.2.3.4"
```

### TCP Protocol Test (via netcat)
```bash
nc localhost 3495
# Server sends: 1234567890.9876543210
# Compute hash: echo -n "user:1234567890.9876543210:password" | md5sum
# Send: user:computed_hash:domain.com:0:1.2.3.4
```

## Debugging Tips

### Common Issues

**Port already in use:**
```bash
lsof -i :3495
kill -9 <PID>
```

**Config file not found:**
- Ensure `config.yaml` exists in working directory
- Config must not be committed (contains secrets)

**Authentication failures:**
- Verify AccessKey/SecretKey are correct
- Check MD5 hash calculation: `MD5(username:salt:password)`
- Remove extra whitespace from credentials

### Logging
Server logs to stdout. Capture with:
```bash
./cloud-ddns 2>&1 | tee cloud-ddns.log
```

## CI/CD

### GitHub Actions Workflows

**build.yml** (on push/PR):
- Tests on Go 1.23, 1.24, 1.25
- Checks code formatting (`gofmt`)
- Runs go vet static analysis
- Runs golangci-lint
- Builds binary
- Runs tests with race detector
- Uploads coverage to Codecov

**release.yml** (on version tags):
- Builds for: Linux (amd64, arm64, armv7), macOS (amd64, arm64), Windows (amd64)
- Creates GitHub release with binaries
- Builds and pushes multi-arch Docker images to GHCR

### Creating a Release
```bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
# GitHub Actions handles the rest
```

## Dependencies

### Core Dependencies
- `gopkg.in/yaml.v3` - YAML configuration parsing
- `github.com/alibabacloud-go/alidns-20150109/v4` - Alibaba Cloud DNS SDK
- `github.com/tencentcloud/tencentcloud-sdk-go` - Tencent Cloud SDK

### Go Version
- Minimum: Go 1.25
- Tested on: Go 1.23, 1.24, 1.25

## Important Coding Patterns

### Provider Factory Pattern
```go
func GetProvider(u *config.UserConfig) (Provider, error) {
    switch u.Provider {
    case "aliyun":
        return NewAliyunProvider(u.Username, u.Password), nil
    case "tencent":
        return NewTencentProvider(u.Username, u.Password), nil
    default:
        return nil, errors.New("unknown provider: " + u.Provider)
    }
}
```

### Error Handling Pattern
```go
// Always check errors
result, err := someFunction()
if err != nil {
    log.Printf("Error context: %v", err)
    return err
}

// Nil pointer checks
if pointer != nil && *pointer == value {
    // Safe to dereference
}
```

### Testing Pattern
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"case1", "input1", "output1", false},
        {"error_case", "bad", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Project Conventions

### File Organization
- One primary type per file (e.g., `aliyun.go` has `AliyunProvider`)
- Tests in `*_test.go` files alongside source
- Interfaces defined in separate files (e.g., `provider.go`)

### Naming Conventions
- Exported names: PascalCase (`UpdateRecord`, `AliyunProvider`)
- Unexported names: camelCase (`accessKey`, `handleConnection`)
- Acronyms: all caps if exported (`HTTPPort`), lowercase if unexported (`tcpPort`)

### Import Order
1. Standard library
2. External dependencies
3. Internal packages

Example:
```go
import (
    "fmt"
    "strings"
    
    "gopkg.in/yaml.v3"
    
    "github.com/NewFuture/CloudDDNS/pkg/config"
)
```

## Security Considerations

- Config file (`config.yaml`) contains secrets - never commit
- Credentials passed as plain text in HTTP mode - use HTTPS in production
- MD5 used for backward compatibility with GnuDIP protocol
- Always validate domain format before API calls
- Log errors but don't expose secrets in logs

## License

Apache License 2.0 - See LICENSE file for details.
