# Cloud-DDNS Development Agent

name: cloud-ddns-dev
description: Development agent for Cloud-DDNS project with Go 1.25 and GitHub integration

## Tools and Servers

### GitHub MCP Server
- **Purpose**: Repository operations, issue tracking, PR management
- **Capabilities**:
  - Browse repository structure
  - Read and search code
  - Access issues and pull requests
  - View commit history
  - Check workflow status

## Runtime Configuration

### Go 1.25
- **Language**: Go
- **Version**: 1.25 (latest stable)
- **Environment**:
  - GOROOT: Standard Go 1.25 installation
  - GOPATH: Default workspace
  - GO111MODULE: on (module mode enabled)

## Project Context

### Key Files for Understanding
1. `.copilot/project-overview.md` - Architecture and design patterns
2. `.copilot/development-guide.md` - Development workflows
3. `README.md` - User documentation (bilingual)
4. `CONTRIBUTING.md` - Contribution guidelines
5. `go.mod` - Dependencies and Go version

### Module Structure
- `pkg/config/` - Configuration management (YAML, user lookup)
- `pkg/provider/` - Cloud DNS provider adapters (Aliyun, Tencent)
- `pkg/server/` - Protocol servers (GnuDIP TCP, HTTP)
- `main.go` - Application entry point

### Important Patterns
- **Provider Interface**: All cloud providers implement `UpdateRecord(domain, ip) error`
- **Factory Pattern**: `provider.GetProvider()` creates provider instances
- **Pass-through Auth**: Username=AccessKey, Password=SecretKey (no translation)
- **Protocol**: GnuDIP with MD5 challenge-response authentication

## Development Workflow

### Common Tasks
```bash
# Build
make build

# Test
make test
make test-coverage

# Run
make run

# Docker
make docker
make docker-up
```

### Adding Features
1. Understand existing patterns in `.copilot/project-overview.md`
2. Follow module structure in `pkg/`
3. Write tests alongside code
4. Update documentation in README.md
5. Run `make test` before committing

### Code Standards
- Follow Go conventions (`go fmt`, `go vet`)
- Add comments for exported functions
- Check for nil pointers before dereferencing
- Return errors explicitly (no silent failures)
- Log errors with context

## Testing Strategy

### Unit Tests
- Config: 100% coverage target
- Provider: Factory and constructor tests
- Server: Protocol logic, parsing, authentication

### Running Tests
```bash
go test ./...                    # All tests
go test -v ./pkg/config          # Specific package
go test -race -coverprofile=...  # With race detection
```

## Cloud Providers

### Supported
- **Aliyun** (Alibaba Cloud DNS): Uses alidns-20150109 SDK
- **Tencent** (DNSPod): Uses tencentcloud-sdk-go

### Adding New Provider
1. Create `pkg/provider/newprovider.go`
2. Implement `Provider` interface
3. Add to factory in `provider.GetProvider()`
4. Write tests
5. Update config.yaml.example

## Protocols

### GnuDIP TCP (Port 3495)
1. Server sends salt: `timestamp.nanoseconds`
2. Client sends: `User:MD5Hash:Domain:RequestCode:IP`
3. Hash validation: MD5(User:Salt:Password)
4. Response: `0` (success) or `1` (failure)

### HTTP (Port 8080)
- **Endpoint**: `/?user=KEY&pass=SECRET&domn=DOMAIN&addr=IP`
- **Auto-IP**: Omit `addr` to use client IP
- **Response**: `0` or error message

## CI/CD

### GitHub Actions
- **build.yml**: Test on Go 1.23/1.24/1.25, lint, build, Docker
- **release.yml**: Multi-platform binaries, Docker images on tag push

### Release Process
```bash
git tag -a v1.0.0 -m "Release 1.0.0"
git push origin v1.0.0
# Automated builds and releases
```

## Dependencies

### Core
- `gopkg.in/yaml.v3` - Config parsing
- `github.com/alibabacloud-go/alidns-20150109/v4` - Aliyun DNS
- `github.com/tencentcloud/tencentcloud-sdk-go` - Tencent DNS

### Development
- Standard Go toolchain
- Make (optional convenience)
- Docker (optional deployment)

## License

Apache License 2.0 - See LICENSE file
