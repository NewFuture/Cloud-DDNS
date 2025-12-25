# Contributing to Cloud-DDNS

Thank you for your interest in contributing to Cloud-DDNS!

## Development Setup

1. Clone the repository:
```bash
git clone https://github.com/NewFuture/CloudDDNS.git
cd CloudDDNS
```

2. Install dependencies:
```bash
go mod download
```

3. Build the project:
```bash
go build -o cloud-ddns main.go
```

## Project Structure

- `main.go` - Application entry point
- `pkg/config/` - Configuration loading and management
- `pkg/provider/` - Cloud provider adapters (Aliyun, Tencent, etc.)
- `pkg/server/` - GnuDIP protocol implementation (TCP & HTTP)

## Provider credential mapping

| Provider  | Username (request/Basic Auth) | Password (request/Basic Auth)         | Notes                          |
|-----------|-------------------------------|---------------------------------------|--------------------------------|
| aliyun    | AccessKey ID                  | AccessKey Secret                      | Passed directly to AliDNS API  |
| tencent   | DNSPod ID/Token ID            | DNSPod Token value                    | Uses Tencent DNSPod API        |

## Adding a New Cloud Provider

To add support for a new cloud DNS provider:

1. Create a new file in `pkg/provider/` (e.g., `cloudflare.go`)
2. Implement the `Provider` interface:

```go
package provider

import "github.com/NewFuture/CloudDDNS/pkg/config"

type CloudflareProvider struct {
    apiToken string
}

func NewCloudflareProvider(token string) *CloudflareProvider {
    return &CloudflareProvider{apiToken: token}
}

func (p *CloudflareProvider) UpdateRecord(domain string, ip string) error {
    // Implement DNS record update logic here
    return nil
}
```

3. Register the provider in `pkg/provider/provider.go`:

```go
func GetProvider(u *config.UserConfig) (Provider, error) {
    switch u.Provider {
    case "aliyun":
        return NewAliyunProvider(u.Username, u.Password), nil
    case "tencent":
        return NewTencentProvider(u.Username, u.Password), nil
    case "cloudflare":  // Add your new provider
        return NewCloudflareProvider(u.Password), nil
    default:
        return nil, errors.New("unknown provider: " + u.Provider)
    }
}
```

## Testing

Run the test suite before submitting changes:

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report
make test-coverage
```

All tests must pass before submitting a pull request. The test suite includes:
- Unit tests for configuration module (100% coverage)
- Unit tests for provider factory and domain parsing
- Integration tests for TCP and HTTP servers
- Protocol logic validation tests

Example manual HTTP test:
```bash
curl "http://localhost:8080/?user=YOUR_KEY&pass=YOUR_SECRET&domn=test.example.com&addr=1.2.3.4"
```

## Code Quality Checks

Before committing changes, ensure your code passes all quality checks:

### 1. Code Formatting
```bash
# Check code formatting
make fmt-check

# Auto-format code
make fmt
```

### 2. Static Analysis
```bash
# Run go vet
make vet
```

### 3. Build Verification
```bash
# Build the binary
make build
```

## Pre-Commit Checklist

Before committing your changes, ensure:

- [ ] All unit tests pass (`make test`)
- [ ] Code is properly formatted (`make fmt-check`)
- [ ] Code passes static analysis (`make vet`)
- [ ] Build succeeds without errors (`make build`)
- [ ] New features have corresponding tests
- [ ] Documentation is updated if needed

## Code Style

- Follow standard Go formatting (use `go fmt` or `make fmt`)
- Add comments for all exported functions and types
- Keep functions focused and single-purpose (max ~50 lines)
- Use meaningful variable names
- Handle errors explicitly - no silent failures

## Submitting Changes

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## Questions?

Open an issue for any questions or discussions!
