# Development Guide for Cloud-DDNS

## Getting Started

### Prerequisites
- Go 1.25 or later
- Docker (optional, for containerized deployment)
- Make (optional, for convenience commands)

### Local Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/NewFuture/CloudDDNS.git
   cd CloudDDNS
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Create configuration**
   ```bash
   cp config.yaml.example config.yaml
   # Edit config.yaml with your credentials
   ```

4. **Build the project**
   ```bash
   make build
   # or
   go build -o cloud-ddns main.go
   ```

5. **Run tests**
   ```bash
   make test
   # or
   go test ./...
   ```

## Project Structure

```
Cloud-DDNS/
├── main.go                    # Application entry point
├── config.yaml.example        # Example configuration
├── go.mod                     # Go module dependencies
├── pkg/
│   ├── config/               # Configuration management
│   │   ├── config.go         # YAML loader and user lookup
│   │   └── config_test.go    # Configuration tests
│   ├── provider/             # Cloud provider adapters
│   │   ├── provider.go       # Interface and factory
│   │   ├── aliyun.go         # Alibaba Cloud implementation
│   │   ├── tencent.go        # Tencent Cloud implementation
│   │   └── provider_test.go  # Provider tests
│   └── server/               # Protocol servers
│       ├── server.go         # TCP and HTTP servers
│       └── server_test.go    # Protocol logic tests
├── .github/workflows/        # CI/CD pipelines
│   ├── build.yml            # Build and test workflow
│   └── release.yml          # Release workflow
└── .copilot/                # Copilot documentation
    ├── project-overview.md  # Architecture overview
    └── development-guide.md # This file
```

## Common Development Tasks

### Running the Server Locally
```bash
# Start the server (requires config.yaml)
make run
# or
go run main.go
```

### Running Tests
```bash
# Run all tests
make test

# Run with verbose output
make test-verbose

# Generate coverage report
make test-coverage
# Opens coverage.html in browser
```

### Adding a New Cloud Provider

1. **Create provider file**
   ```bash
   touch pkg/provider/newprovider.go
   ```

2. **Implement the Provider interface**
   ```go
   package provider
   
   type NewProvider struct {
       apiKey string
       apiSecret string
   }
   
   func NewNewProvider(key, secret string) *NewProvider {
       return &NewProvider{apiKey: key, apiSecret: secret}
   }
   
   func (p *NewProvider) UpdateRecord(domain string, ip string) error {
       // 1. Parse domain into base domain and subdomain
       // 2. Query existing records (if needed)
       // 3. Check if IP changed
       // 4. Update or create DNS record
       return nil
   }
   ```

3. **Register in factory**
   ```go
   // In pkg/provider/provider.go
   func GetProvider(u *config.UserConfig) (Provider, error) {
       switch u.Provider {
       case "aliyun":
           return NewAliyunProvider(u.Username, u.Password), nil
       case "tencent":
           return NewTencentProvider(u.Username, u.Password), nil
       case "newprovider":  // Add this
           return NewNewProvider(u.Username, u.Password), nil
       default:
           return nil, errors.New("unknown provider: " + u.Provider)
       }
   }
   ```

4. **Add tests**
   ```bash
   # Add test in pkg/provider/provider_test.go
   ```

5. **Update documentation**
   - Add provider to README.md
   - Update config.yaml.example

### Building Docker Image
```bash
# Build image
make docker
# or
docker build -t cloud-ddns:latest .

# Run with docker-compose
make docker-up
```

### Testing GnuDIP Protocol

**Via HTTP** (easier for testing):
```bash
curl "http://localhost:8080/?user=YOUR_KEY&pass=YOUR_SECRET&domn=test.example.com&addr=1.2.3.4"
```

**Via TCP** (requires GnuDIP client or netcat):
```bash
# Connect to server
nc localhost 3495
# Server sends salt: 1234567890.9876543210
# Compute MD5: echo -n "user:1234567890.9876543210:password" | md5sum
# Send: user:computed_hash:domain.com:0:1.2.3.4
```

## Code Style and Standards

### Go Conventions
- Follow standard Go formatting: `go fmt ./...`
- Use meaningful variable names
- Add comments for exported functions
- Keep functions focused and single-purpose

### Error Handling
- Always check errors explicitly
- Log errors with context
- Return errors up the stack
- Add nil checks before dereferencing pointers

### Testing
- Write tests for new features
- Aim for high coverage in core modules
- Use table-driven tests for multiple cases
- Test error paths, not just happy paths

## Debugging Tips

### Enable Verbose Logging
The server logs to stdout. Run with:
```bash
./cloud-ddns 2>&1 | tee cloud-ddns.log
```

### Common Issues

**Port already in use**
```bash
# Find process using port
lsof -i :3495
# Kill process
kill -9 <PID>
```

**Config not found**
- Ensure config.yaml exists in current directory
- Or specify path: `CONFIG_PATH=./path/to/config.yaml ./cloud-ddns`

**Authentication failures**
- Verify AccessKey/SecretKey are correct
- Check salt calculation: MD5(username:salt:password)
- Ensure no extra whitespace in credentials

## CI/CD Pipeline

### GitHub Actions Workflows

**build.yml** - Runs on every push/PR:
- Tests on Go 1.23, 1.24, 1.25
- Runs linter (golangci-lint)
- Builds binary
- Runs tests with race detector
- Uploads coverage to Codecov

**release.yml** - Runs on version tags (v*):
- Builds for multiple platforms
- Creates GitHub release
- Builds and pushes Docker images

### Creating a Release
```bash
# Tag the commit
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0

# GitHub Actions will automatically:
# - Build binaries for all platforms
# - Create GitHub release with artifacts
# - Build and push Docker images
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Run linter: `go fmt ./...`
6. Commit with clear messages
7. Push and create Pull Request

## Getting Help

- Check existing issues: https://github.com/NewFuture/Cloud-DDNS/issues
- Review documentation in `.copilot/` directory
- Read code comments in source files
