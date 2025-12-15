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

Currently, the project uses manual testing. To test your changes:

1. Create a test configuration file
2. Run the server
3. Test with a real device or use curl for HTTP endpoint

Example HTTP test:
```bash
curl "http://localhost:8080/?user=YOUR_KEY&pass=YOUR_SECRET&domn=test.example.com&addr=1.2.3.4"
```

## Code Style

- Follow standard Go formatting (use `go fmt`)
- Add comments for exported functions and types
- Keep functions focused and single-purpose

## Submitting Changes

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## Questions?

Open an issue for any questions or discussions!
